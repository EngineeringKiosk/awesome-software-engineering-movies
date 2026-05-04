package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/EngineeringKiosk/awesome-software-engineering-movies/youtube"
)

// MovieInformation is the unified record we persist to JSON for each
// curated entry. The yaml-tagged fields are the maintainer-authored
// inputs; the rest are filled in by collectMovieData from the
// YouTube Data API.
type MovieInformation struct {
	Name string `yaml:"name" json:"name"`
	// Slug is derived from Name by convertYamlToJson and used for
	// filenames, README anchors and image paths.
	Slug string `yaml:"-" json:"slug"`

	Link string `yaml:"link" json:"link"`
	// VideoID is parsed from Link by convertYamlToJson.
	VideoID string `yaml:"-" json:"videoID"`

	// Language is a list of ISO 639-1 codes (e.g. ["en"], ["en", "de"]).
	// Curated by hand because the YouTube API does not reliably expose
	// the audio language of a video.
	Language []string `yaml:"language" json:"language"`
	// Subtitles is a list of ISO 639-1 codes for the subtitle/caption
	// tracks the video is available with. Curated in YAML and unioned
	// with the languages reported by the YouTube captions.list API.
	Subtitles []string `yaml:"subtitles" json:"subtitles"`
	Tags      []string `yaml:"tags"      json:"tags"`

	// API-enriched fields below this line.
	Title string `yaml:"-" json:"title"`
	// Description is optional in YAML. If supplied, the YAML value
	// overrides whatever the YouTube API returns; if omitted, the
	// API's snippet.description is used. Same precedence rule as
	// Language.
	Description string          `yaml:"description,omitempty" json:"description"`
	Duration    string          `yaml:"-"                     json:"duration"` // ISO-8601, e.g. PT43M22S
	PublishedAt string          `yaml:"-" json:"publishedAt"`
	Channel     youtube.Channel `yaml:"-" json:"channel"`
	// Ratings groups rating signals by source. Today only YouTube is
	// populated (likeCount); a future second source — e.g. an external
	// review aggregator — would slot in as another key.
	Ratings Ratings `yaml:"-" json:"ratings,omitzero"`
	// Views groups view counts by source. Today only YouTube is
	// populated. Per-source attribution makes the provenance explicit
	// in the JSON itself.
	Views Views  `yaml:"-" json:"views,omitzero"`
	Image string `yaml:"-" json:"image"`
}

// YouTubeRating holds rating-like signals YouTube exposes via
// videos.list.statistics. Aggregate rating numbers are no longer
// public (dislikes were removed in 2021), so likeCount is the only
// signal worth recording.
type YouTubeRating struct {
	LikeCount int64 `json:"likeCount"`
}

// Ratings is the per-source rating container. Keys are omitted when
// the corresponding source has not been queried, so presence of a key
// doubles as a source-attribution marker.
type Ratings struct {
	YouTube *YouTubeRating `json:"youtube,omitempty"`
}

// Views is the per-source view-count container. Same omitempty
// convention as Ratings.
type Views struct {
	YouTube *int64 `json:"youtube,omitempty"`
}

// TagsAsList returns the tags joined with ", " for README rendering.
func (m *MovieInformation) TagsAsList() string {
	return strings.Join(m.Tags, ", ")
}

// LanguagesAsList returns the language codes joined with ", ".
func (m *MovieInformation) LanguagesAsList() string {
	return strings.Join(m.Language, ", ")
}

// SubtitlesAsList returns the subtitle codes joined with ", ".
func (m *MovieInformation) SubtitlesAsList() string {
	return strings.Join(m.Subtitles, ", ")
}

// DurationHumanReadable converts the ISO-8601 duration into an
// approximate, scannable form for the README. Seconds are dropped
// (used only for rounding); the result is prefixed with "ca." and
// suffixed with "min." (with a "h" hour segment when applicable):
//
//	PT31M50S    → "ca. 32 min."
//	PT45S       → "ca. 1 min."
//	PT2H        → "ca. 2 h 0 min."
//	PT1H23M45S  → "ca. 1 h 24 min."
//
// Returns the raw value back if parsing fails so the README still
// renders.
func (m *MovieInformation) DurationHumanReadable() string {
	d, err := parseISO8601Duration(m.Duration)
	if err != nil {
		return m.Duration
	}

	// Round to nearest minute. Anything > 0 but < 30 s rounds up to
	// 1 min so we never render the meaningless "ca. 0 min."
	totalMinutes := int((d + 30*time.Second) / time.Minute)
	if totalMinutes == 0 && d > 0 {
		totalMinutes = 1
	}

	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	if hours > 0 {
		return fmt.Sprintf("ca. %d h %d min.", hours, minutes)
	}
	return fmt.Sprintf("ca. %d min.", minutes)
}

// parseISO8601Duration parses the YouTube duration format. It only
// supports the subset YouTube actually emits: PT[xH][yM][zS]. Days
// and longer units never appear for video durations.
func parseISO8601Duration(s string) (time.Duration, error) {
	if !strings.HasPrefix(s, "PT") {
		return 0, fmt.Errorf("not an ISO-8601 duration: %q", s)
	}
	rest := s[2:]
	if rest == "" {
		return 0, fmt.Errorf("empty duration body: %q", s)
	}

	var total time.Duration
	num := ""
	for _, r := range rest {
		switch {
		case r >= '0' && r <= '9':
			num += string(r)
		case r == 'H' || r == 'M' || r == 'S':
			if num == "" {
				return 0, fmt.Errorf("missing number before %c in %q", r, s)
			}
			n := 0
			if _, err := fmt.Sscanf(num, "%d", &n); err != nil {
				return 0, err
			}
			switch r {
			case 'H':
				total += time.Duration(n) * time.Hour
			case 'M':
				total += time.Duration(n) * time.Minute
			case 'S':
				total += time.Duration(n) * time.Second
			}
			num = ""
		default:
			return 0, fmt.Errorf("unexpected character %c in duration %q", r, s)
		}
	}
	if num != "" {
		return 0, fmt.Errorf("trailing number with no unit in %q", s)
	}
	return total, nil
}
