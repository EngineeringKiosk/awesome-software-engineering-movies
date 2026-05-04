package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/EngineeringKiosk/awesome-software-engineering-movies/imdb"
	libIO "github.com/EngineeringKiosk/awesome-software-engineering-movies/io"
	"github.com/EngineeringKiosk/awesome-software-engineering-movies/platform"
	"github.com/EngineeringKiosk/awesome-software-engineering-movies/youtube"
)

// imdbRefreshInterval is how long an IMDb rating row is considered
// fresh enough to keep without refetching. Anything older triggers a
// refetch; anything younger is left alone unless --force-imdb-refresh
// is set. 30 days lines up with the monthly enrichment workflow this
// command is invoked from.
const imdbRefreshInterval = 30 * 24 * time.Hour

const (
	imageFolder      = "images"
	defaultUserAgent = "EngineeringKiosk-awesome-software-engineering-movies"
	youtubeAPIKeyEnv = "YOUTUBE_API_KEY"
)

// collectMovieDataCmd represents the collectMovieData command
var collectMovieDataCmd = &cobra.Command{
	Use:   "collectMovieData",
	Short: "Collects additional data per movie from the YouTube Data API",
	Long: `We only have basic data about each movie from the YAML file.
To make the whole project more useful, we aim to enrich each entry with
information from the YouTube Data API: title, description, duration,
channel, view count and thumbnail.

This command updates the JSON files in place and downloads thumbnails
into the images subdirectory.`,
	RunE: cmdCollectMovieData,
}

func init() {
	rootCmd.AddCommand(collectMovieDataCmd)

	collectMovieDataCmd.Flags().String("json-directory", "", "Directory on where to store the json files")
	collectMovieDataCmd.Flags().String("youtube-api-key", "", "YouTube Data API v3 key (falls back to YOUTUBE_API_KEY env var)")
	collectMovieDataCmd.Flags().Bool("force-imdb-refresh", false, "Refresh IMDb ratings for every entry with an imdbID, ignoring the 30-day cache window")

	err := collectMovieDataCmd.MarkFlagRequired("json-directory")
	if err != nil {
		log.Fatalf("Error marking flag as required: %v", err)
	}
}

func cmdCollectMovieData(cmd *cobra.Command, args []string) error {
	jsonDir, err := cmd.Flags().GetString("json-directory")
	if err != nil {
		return err
	}

	apiKey, err := cmd.Flags().GetString("youtube-api-key")
	if err != nil {
		return err
	}
	if apiKey == "" {
		apiKey = os.Getenv(youtubeAPIKeyEnv)
	}
	if apiKey == "" {
		return fmt.Errorf("YouTube API key not provided: pass --youtube-api-key or set %s", youtubeAPIKeyEnv)
	}

	forceIMDb, err := cmd.Flags().GetBool("force-imdb-refresh")
	if err != nil {
		return err
	}

	ctx := context.Background()

	log.Printf("Reading files with extension %s from directory %s", libIO.JSONExtension, jsonDir)
	jsonFiles, err := libIO.GetAllFilesFromDirectory(jsonDir, libIO.JSONExtension)
	if err != nil {
		return err
	}
	log.Printf("%d files found with extension %s in directory %s", len(jsonFiles), libIO.JSONExtension, jsonDir)

	// Load every JSON file once; we need both the video ID list (for
	// the batched API call) and the per-entry struct (to enrich).
	type entry struct {
		path string
		info *MovieInformation
	}
	var entries []entry
	var ids []string
	for _, f := range jsonFiles {
		absJsonFilePath := filepath.Join(jsonDir, f.Name())
		jsonFileContent, err := os.ReadFile(absJsonFilePath)
		if err != nil {
			return err
		}
		info := &MovieInformation{}
		if err := json.Unmarshal(jsonFileContent, info); err != nil {
			return fmt.Errorf("unmarshal %s: %w", absJsonFilePath, err)
		}
		entries = append(entries, entry{path: absJsonFilePath, info: info})
		// YouTube enrichment runs only on YouTube entries. Other
		// platforms (Netflix, Amazon Prime Video, bpb, …) do not have
		// a YouTube video ID by construction.
		if info.Platform != platform.YouTube {
			continue
		}
		// VideoID is not persisted in JSON; derive it from Link on
		// load. The parse-failure warning fires here, where the
		// failure actually matters (we are about to ask the API for
		// it).
		if id, ok := youtube.ParseVideoID(info.Link); ok {
			info.VideoID = id
			ids = append(ids, id)
		} else {
			log.Printf("WARNING: could not parse YouTube video ID from %q in %s", info.Link, absJsonFilePath)
		}
	}

	yt, err := youtube.NewClient(ctx, apiKey)
	if err != nil {
		return err
	}

	log.Printf("Fetching video details for %d ID(s) from YouTube ...", len(ids))
	videos, err := yt.GetVideoDetails(ctx, ids)
	if err != nil {
		return fmt.Errorf("youtube enrichment failed: %w", err)
	}
	log.Printf("Fetched %d video record(s)", len(videos))

	byID := make(map[string]youtube.Video, len(videos))
	for _, v := range videos {
		byID[v.ID] = v
	}

	imageDir := filepath.Join(jsonDir, imageFolder)
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return err
	}

	now := time.Now().UTC()
	nowRFC3339 := now.Format(time.RFC3339)

	for _, e := range entries {
		info := e.info

		// Mirror the gate from the ID-collection loop above: only
		// YouTube entries go through the YouTube enrichment branch
		// at all. Other platforms keep their existing cached values
		// (which the IMDb pass below may still update independently).
		if info.Platform != platform.YouTube {
			continue
		}

		v, ok := byID[info.VideoID]
		if !ok {
			log.Printf("WARNING: YouTube returned no record for %s (%s); keeping cached values", info.Name, info.VideoID)
		} else {
			info.Title = v.Title
			info.Duration = v.Duration
			info.PublishedAt = v.PublishedAt
			info.Channel = v.Channel

			views := v.ViewCount
			info.Views.YouTube = &views
			info.Ratings.YouTube = &YouTubeRating{
				LikeCount:   v.LikeCount,
				RefreshedAt: nowRFC3339,
			}

			// Description is YAML-overridable: only use the API value
			// when the YAML left it empty. Same precedence as Language.
			if len(info.Description) == 0 {
				info.Description = v.Description
			}

			// Language is YAML-curated and unioned with the API's
			// defaultAudioLanguage: YAML order is preserved, the API
			// code is appended only if not already present.
			var apiLang []string
			if code := normalizeLanguageCode(v.DefaultAudioLanguage); code != "" {
				apiLang = []string{code}
			}
			info.Language = mergeLanguageCodes(info.Language, apiLang)
			if len(info.Language) == 0 {
				log.Printf("WARNING: %s has no language in YAML and YouTube did not return defaultAudioLanguage; leaving empty", info.Name)
			}

			// Subtitles follow the same union rule against the
			// captions.list result.
			info.Subtitles = mergeLanguageCodes(info.Subtitles, normalizeLanguageCodes(v.Subtitles))
		}

	}

	// Thumbnail population runs as its own pass against every entry,
	// not just YouTube ones — non-YouTube entries can still get a
	// poster via youtubeTrailerForThumbnail, and the placeholder
	// fallback covers the rest.
	for _, e := range entries {
		populateThumbnail(e.info, imageDir)
	}

	// IMDb enrichment is selective: the dataset is only fetched when
	// at least one entry needs it, so quiet runs (no IMDb data missing
	// or stale) skip the multi-megabyte download entirely.
	infos := make([]*MovieInformation, 0, len(entries))
	for _, e := range entries {
		infos = append(infos, e.info)
	}
	enrichWithIMDbRatings(ctx, infos, forceIMDb, now)

	// One write per entry, after both enrichment passes have run.
	// Doing this in a single loop avoids writing each file twice when
	// IMDb data changes alongside YouTube data.
	for _, e := range entries {
		log.Printf("Write %s to disk ...", e.path)
		if err := libIO.WriteJSONFile(e.path, e.info); err != nil {
			return err
		}
	}

	return nil
}

// needsIMDbRefresh reports whether the entry's IMDb rating block
// should be (re)fetched. Pure function so the decision is unit-
// testable in isolation from the network.
//
// Returns false for entries with no imdbID — those are the YouTube-
// only majority of the catalogue and should never trigger a dataset
// download.
func needsIMDbRefresh(info *MovieInformation, force bool, now time.Time) bool {
	if info.IMDbID == "" {
		return false
	}
	if force {
		return true
	}
	if info.Ratings.IMDb == nil {
		return true
	}
	last, err := time.Parse(time.RFC3339, info.Ratings.IMDb.RefreshedAt)
	if err != nil {
		// Unparseable timestamp = treat as missing. Better to
		// overshoot once than carry stale data behind a corrupt
		// field.
		return true
	}
	return now.Sub(last) > imdbRefreshInterval
}

// enrichWithIMDbRatings fills in info.Ratings.IMDb for entries whose
// IMDb data is missing or stale. The function is best-effort:
// download or parse failures are logged and the caller continues with
// whatever data is already on disk, mirroring how the rest of this
// command handles per-entry failures.
func enrichWithIMDbRatings(ctx context.Context, infos []*MovieInformation, force bool, now time.Time) {
	type pending struct {
		info *MovieInformation
		id   string
	}
	var todo []pending
	for _, info := range infos {
		if needsIMDbRefresh(info, force, now) {
			todo = append(todo, pending{info: info, id: info.IMDbID})
		}
	}
	if len(todo) == 0 {
		log.Printf("IMDb ratings: nothing to refresh, skipping dataset download")
		return
	}

	ids := make([]string, 0, len(todo))
	for _, p := range todo {
		ids = append(ids, p.id)
	}
	log.Printf("IMDb ratings: fetching dataset to refresh %d entr(ies)", len(todo))

	ratings, err := imdb.FetchRatings(ctx, ids)
	if err != nil {
		log.Printf("WARNING: imdb dataset fetch failed: %v; keeping any cached IMDb ratings", err)
		return
	}

	nowRFC3339 := now.Format(time.RFC3339)
	for _, p := range todo {
		r, ok := ratings[p.id]
		if !ok {
			log.Printf("WARNING: imdb dataset has no row for %s (%s); skipping", p.info.Name, p.id)
			continue
		}
		p.info.Ratings.IMDb = &IMDbRating{
			AverageRating: r.AverageRating,
			NumVotes:      r.NumVotes,
			RefreshedAt:   nowRFC3339,
		}
	}
}

// normalizeLanguageCodes maps normalizeLanguageCode over a slice and
// drops empties. It does not deduplicate — that is mergeLanguageCodes'
// job, which keeps the dedupe logic in one place.
func normalizeLanguageCodes(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if c := normalizeLanguageCode(t); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// mergeLanguageCodes returns the union of yamlCodes and apiCodes,
// preserving yamlCodes order first, then appending any apiCodes
// not already present. Both inputs are normalized via
// normalizeLanguageCode; empty results are dropped.
func mergeLanguageCodes(yamlCodes, apiCodes []string) []string {
	seen := make(map[string]struct{}, len(yamlCodes)+len(apiCodes))
	var out []string
	for _, list := range [][]string{yamlCodes, apiCodes} {
		for _, raw := range list {
			c := normalizeLanguageCode(raw)
			if c == "" {
				continue
			}
			if _, dup := seen[c]; dup {
				continue
			}
			seen[c] = struct{}{}
			out = append(out, c)
		}
	}
	return out
}

// normalizeLanguageCode reduces a BCP-47 tag to its base ISO 639-1
// code (e.g. "en-US" → "en", "de" → "de"). Returns "" if the input
// is empty or has no leading subtag.
func normalizeLanguageCode(tag string) string {
	tag = strings.TrimSpace(strings.ToLower(tag))
	if tag == "" {
		return ""
	}
	if i := strings.IndexByte(tag, '-'); i > 0 {
		return tag[:i]
	}
	return tag
}

// placeholderImage is the repo-relative path of the bundled SVG that
// gets rendered when no YouTube thumbnail is available for an entry.
// The file is a static asset committed under generated/images/, not
// produced by this command.
const placeholderImage = "images/placeholder.svg"

// populateThumbnail fills info.Image using a layered fallback:
//
//  1. Primary YouTube link (info.VideoID).
//  2. Curated YouTube trailer URL (info.YouTubeTrailerForThumbnail).
//  3. Bundled placeholder SVG when the entry has no other image.
//
// The cache-keep semantic from the previous implementation is
// preserved: if every YouTube candidate fails but info.Image is
// already set to a previously-downloaded thumbnail, we leave it
// alone. The placeholder is only chosen when there is nothing else.
func populateThumbnail(info *MovieInformation, imageDir string) {
	var candidates []string
	if info.VideoID != "" {
		candidates = append(candidates, info.VideoID)
	}
	if info.YouTubeTrailerForThumbnail != "" {
		if id, ok := youtube.ParseVideoID(info.YouTubeTrailerForThumbnail); ok {
			candidates = append(candidates, id)
		} else {
			log.Printf("WARNING: %s: youtubeTrailerForThumbnail %q is not a YouTube URL; ignoring",
				info.Name, info.YouTubeTrailerForThumbnail)
		}
	}

	for _, id := range candidates {
		imgPath, err := downloadThumbnail(id, info.Slug, imageDir)
		if err == nil {
			info.Image = filepath.Join(imageFolder, filepath.Base(imgPath))
			return
		}
		log.Printf("WARNING: thumbnail download for %s via %s failed: %v",
			info.Name, id, err)
	}

	// All YouTube candidates failed (or none were available). Keep
	// any cached image; otherwise fall back to the static placeholder.
	if info.Image == "" {
		info.Image = placeholderImage
	}
}

// downloadThumbnail tries maxresdefault.jpg first, then falls back to
// hqdefault.jpg (which YouTube guarantees is present for every public
// video). On success it returns the absolute path of the saved file.
func downloadThumbnail(videoID, slug, imageDir string) (string, error) {
	dst := filepath.Join(imageDir, slug+".jpg")
	candidates := []string{
		fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", videoID),
		fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID),
	}

	var lastErr error
	for _, url := range candidates {
		if err := downloadFile(url, dst); err == nil {
			return dst, nil
		} else {
			lastErr = err
		}
	}
	return "", lastErr
}

func downloadFile(address, fileName string) error {
	client := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 30 * time.Second,
		},
	}

	req, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return errors.New("received non-200 status: " + resp.Status)
	}

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return err
	}
	return nil
}
