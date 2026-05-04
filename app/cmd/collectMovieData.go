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

	libIO "github.com/EngineeringKiosk/awesome-software-engineering-movies/io"
	"github.com/EngineeringKiosk/awesome-software-engineering-movies/youtube"
)

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
		if info.VideoID != "" {
			ids = append(ids, info.VideoID)
		} else {
			log.Printf("WARNING: %s has no videoID; skipping API enrichment", absJsonFilePath)
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

	for _, e := range entries {
		info := e.info

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
			info.Ratings.YouTube = &YouTubeRating{LikeCount: v.LikeCount}

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

		if info.VideoID != "" {
			imgPath, err := downloadThumbnail(info.VideoID, info.Slug, imageDir)
			switch {
			case err == nil:
				info.Image = filepath.Join(imageFolder, filepath.Base(imgPath))
			case info.Image != "":
				log.Printf("WARNING: thumbnail download for %s failed: %v. Keeping cached image %s.", info.VideoID, err, info.Image)
			default:
				log.Printf("WARNING: thumbnail download for %s failed and no cached image exists: %v", info.VideoID, err)
			}
		}

		log.Printf("Write %s to disk ...", e.path)
		if err := libIO.WriteJSONFile(e.path, info); err != nil {
			return err
		}
	}

	return nil
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
