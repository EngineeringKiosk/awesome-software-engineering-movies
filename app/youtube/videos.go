package youtube

import (
	"context"
	"fmt"
	"log"

	youtubeapi "google.golang.org/api/youtube/v3"
)

// Channel describes the channel a video belongs to.
type Channel struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Video is the subset of the YouTube videos.list response we care
// about. Decoupled from the SDK type so we control the JSON shape
// that gets persisted to disk.
type Video struct {
	ID          string
	Title       string
	Description string
	Duration    string // ISO-8601, e.g. PT43M22S
	PublishedAt string // RFC3339
	Channel     Channel
	ViewCount   int64
	LikeCount   int64
	Thumbnail   string // highest-resolution thumbnail URL the API offers
	// DefaultAudioLanguage is the uploader-declared audio language as
	// a BCP-47 tag (e.g. "en", "en-US"). Often empty: it is an
	// optional field on the YouTube upload form, especially missing
	// on older videos.
	DefaultAudioLanguage string
	// Subtitles is the deduplicated list of caption-track languages
	// reported by captions.list, as raw BCP-47 tags. Normalization to
	// ISO 639-1 happens in the caller.
	Subtitles []string
}

// videosListMaxIDs is the YouTube API hard limit for videos.list IDs
// per call.
const videosListMaxIDs = 50

// GetVideoDetails fetches metadata for the given video IDs. The list
// is automatically batched into calls of up to 50 IDs (the API limit).
// IDs that the API does not return (private, deleted, region-blocked)
// are simply absent from the result — the caller decides how to
// handle that.
func (c *Client) GetVideoDetails(ctx context.Context, ids []string) ([]Video, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var out []Video
	parts := []string{"snippet", "contentDetails", "statistics"}

	for start := 0; start < len(ids); start += videosListMaxIDs {
		end := min(start+videosListMaxIDs, len(ids))

		call := c.svc.Videos.List(parts).Id(ids[start:end]...).Context(ctx)
		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("youtube: videos.list: %w", err)
		}

		for _, item := range resp.Items {
			if item == nil || item.Snippet == nil || item.ContentDetails == nil {
				continue
			}

			v := Video{
				ID:          item.Id,
				Title:       item.Snippet.Title,
				Description: item.Snippet.Description,
				Duration:    item.ContentDetails.Duration,
				PublishedAt: item.Snippet.PublishedAt,
				Channel: Channel{
					ID:    item.Snippet.ChannelId,
					Title: item.Snippet.ChannelTitle,
				},
				Thumbnail:            bestThumbnail(item.Snippet.Thumbnails),
				DefaultAudioLanguage: item.Snippet.DefaultAudioLanguage,
			}
			if item.Statistics != nil {
				v.ViewCount = int64(item.Statistics.ViewCount)
				v.LikeCount = int64(item.Statistics.LikeCount)
			}

			// captions.list has no batch form — one call per video. At
			// 50 quota units each it is the heaviest part of this
			// command, but the curated movie list is small enough that
			// the total quota stays well within the daily budget.
			subs, err := c.GetCaptionLanguages(ctx, v.ID)
			if err != nil {
				log.Printf("WARNING: youtube: captions.list for %s failed: %v; leaving subtitles empty", v.ID, err)
			} else {
				v.Subtitles = subs
			}

			out = append(out, v)
		}
	}

	return out, nil
}

// GetCaptionLanguages returns the deduplicated set of language codes
// (raw BCP-47 tags) for the caption tracks attached to a video. Empty
// slice when the video has no captions. Auto-generated tracks are
// included — they appear in the API response just like uploaded
// tracks.
func (c *Client) GetCaptionLanguages(ctx context.Context, videoID string) ([]string, error) {
	resp, err := c.svc.Captions.List([]string{"snippet"}, videoID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("youtube: captions.list: %w", err)
	}

	seen := make(map[string]struct{}, len(resp.Items))
	var out []string
	for _, item := range resp.Items {
		if item == nil || item.Snippet == nil {
			continue
		}
		lang := item.Snippet.Language
		if lang == "" {
			continue
		}
		if _, dup := seen[lang]; dup {
			continue
		}
		seen[lang] = struct{}{}
		out = append(out, lang)
	}
	return out, nil
}

// bestThumbnail walks the YouTube thumbnail variants from largest to
// smallest and returns the first non-empty URL.
func bestThumbnail(t *youtubeapi.ThumbnailDetails) string {
	if t == nil {
		return ""
	}
	for _, candidate := range []*youtubeapi.Thumbnail{t.Maxres, t.Standard, t.High, t.Medium, t.Default} {
		if candidate != nil && candidate.Url != "" {
			return candidate.Url
		}
	}
	return ""
}
