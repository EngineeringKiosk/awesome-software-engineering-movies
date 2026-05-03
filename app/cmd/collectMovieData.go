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
			info.Description = v.Description
			info.Duration = v.Duration
			info.PublishedAt = v.PublishedAt
			info.Channel = v.Channel
			info.ViewCount = v.ViewCount

			// Language is YAML-overridable: only fall back to the
			// API-declared audio language when the YAML left it empty.
			if len(info.Language) == 0 {
				if code := normalizeLanguageCode(v.DefaultAudioLanguage); code != "" {
					info.Language = []string{code}
				} else {
					log.Printf("WARNING: %s has no language in YAML and YouTube did not return defaultAudioLanguage; leaving empty", info.Name)
				}
			}
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
