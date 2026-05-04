package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/EngineeringKiosk/awesome-software-engineering-movies/platform"
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
	// VideoID is YouTube-specific and lives only as a runtime field:
	// not persisted to JSON, not curated in YAML. collectMovieData
	// derives it from Link on load via youtube.ParseVideoID and uses
	// it for the videos.list API call, the response→entry join, and
	// the thumbnail URL. Other platforms leave it empty.
	VideoID string `yaml:"-" json:"-"`
	// Platform identifies which source the link lives on (e.g.
	// "youtube"). Optional in YAML — convertYamlToJson auto-fills it
	// from the link via the platform package when omitted. The YAML
	// value, when set, always wins; convertYamlToJson logs a warning
	// when YAML and link disagree.
	Platform string `yaml:"platform,omitempty" json:"platform"`

	// Language is a list of ISO 639-1 codes (e.g. ["en"], ["en", "de"]).
	// Curated by hand because the YouTube API does not reliably expose
	// the audio language of a video.
	Language []string `yaml:"language" json:"language"`
	// Subtitles is a list of ISO 639-1 codes for the subtitle/caption
	// tracks the video is available with. Curated in YAML and unioned
	// with the languages reported by the YouTube captions.list API.
	Subtitles []string `yaml:"subtitles" json:"subtitles"`
	Tags      []string `yaml:"tags"      json:"tags"`
	// Localized maps ISO 639-1 codes (e.g. "de", "es") to per-language
	// overrides. The top-level Name and Link are always the English
	// version; entries here describe alternate-language versions of
	// the same content (a different upload, a translated title, or
	// both). Optional in YAML; absent for entries that only exist in
	// one language. Map omits itself from JSON when empty.
	Localized map[string]LocalizedVersion `yaml:"localized,omitempty" json:"localized,omitempty"`

	// API-enriched fields below this line.
	Title string `yaml:"-" json:"title"`
	// Description is optional in YAML. If supplied, the YAML value
	// overrides whatever the YouTube API returns; if omitted, the
	// API's snippet.description is used. Same precedence rule as
	// Language.
	Description string `yaml:"description,omitempty" json:"description"`
	// IMDbID is the IMDb tconst (e.g. "tt3268458"). Optional in YAML
	// and only set for entries that are also catalogued on IMDb.
	// Drives the IMDb rating lookup in collectMovieData; entries
	// without it never trigger an IMDb dataset download.
	IMDbID string `yaml:"imdbID,omitempty"      json:"imdbID,omitempty"`
	// YouTubeTrailerForThumbnail is an optional YouTube URL used as
	// a fallback source for the entry's poster. When the primary
	// link is not a YouTube video (or the YouTube thumbnail download
	// fails), the tooling extracts the video ID from this URL and
	// pulls hqdefault.jpg / maxresdefault.jpg via the i.ytimg.com
	// URL pattern. If neither path yields an image, the README
	// renders a bundled placeholder.
	YouTubeTrailerForThumbnail string          `yaml:"youtubeTrailerForThumbnail,omitempty" json:"youtubeTrailerForThumbnail,omitempty"`
	Duration                   string          `yaml:"-"                                    json:"duration"` // ISO-8601, e.g. PT43M22S
	PublishedAt                string          `yaml:"-" json:"publishedAt"`
	Channel                    youtube.Channel `yaml:"-" json:"channel"`
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

// LocalizedVersion holds the per-language overrides for one
// alternate version of an entry. All fields are individually
// optional: maintainers fill in whichever differs from the top-level
// English version (a translated title, a translated description, a
// different upload, or any combination).
type LocalizedVersion struct {
	Title       string `yaml:"title,omitempty"       json:"title,omitempty"`
	Link        string `yaml:"link,omitempty"        json:"link,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Platform is autodetected from Link by convertYamlToJson when
	// empty, with the same precedence rules as MovieInformation.Platform:
	// YAML wins when set; the resolver warns on disagreement. Stays
	// empty for description-only or title-only overrides because
	// there is no link to detect against.
	Platform string `yaml:"platform,omitempty" json:"platform,omitempty"`
}

// YouTubeRating holds rating-like signals YouTube exposes via
// videos.list.statistics. Aggregate rating numbers are no longer
// public (dislikes were removed in 2021), so likeCount is the only
// signal worth recording.
type YouTubeRating struct {
	LikeCount int64 `json:"likeCount"`
	// RefreshedAt is the RFC3339 UTC timestamp the value was last
	// written from the YouTube Data API. Informational; YouTube data
	// is refreshed on every collectMovieData run, so it always trails
	// the most recent run by at most a few seconds.
	RefreshedAt string `json:"refreshedAt"`
}

// IMDbRating holds the rating signals IMDb exposes through its
// public non-commercial ratings dataset (title.ratings.tsv.gz). Only
// the two columns that aren't the tconst are persisted.
type IMDbRating struct {
	AverageRating float64 `json:"averageRating"`
	NumVotes      int64   `json:"numVotes"`
	// RefreshedAt is the RFC3339 UTC timestamp the dataset row was
	// last copied into this file. Drives the staleness check in
	// collectMovieData: entries older than 30 days trigger a refetch.
	RefreshedAt string `json:"refreshedAt"`
}

// Ratings is the per-source rating container. Keys are omitted when
// the corresponding source has not been queried, so presence of a key
// doubles as a source-attribution marker.
type Ratings struct {
	YouTube *YouTubeRating `json:"youtube,omitempty"`
	IMDb    *IMDbRating    `json:"imdb,omitempty"`
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

// PlatformDisplay returns the human-readable platform name for the
// README, e.g. "YouTube" for the slug "youtube". Empty / unknown
// slugs degrade gracefully via platform.Display.
func (m *MovieInformation) PlatformDisplay() string {
	return platform.Display(m.Platform)
}

// HasYouTubeLikes reports whether the YouTube rating block is
// populated. Templates use this to avoid emitting a "YouTube likes"
// line for entries that haven't been enriched yet.
func (m *MovieInformation) HasYouTubeLikes() bool {
	return m.Ratings.YouTube != nil
}

// YouTubeLikeCountFormatted renders the YouTube like count with
// thousands separators, e.g. "33,455". Returns "" when the rating
// block is absent so a careless template still produces clean output.
func (m *MovieInformation) YouTubeLikeCountFormatted() string {
	if m.Ratings.YouTube == nil {
		return ""
	}
	return formatGroupedInt(m.Ratings.YouTube.LikeCount)
}

// HasIMDbRating reports whether the IMDb rating block is populated.
func (m *MovieInformation) HasIMDbRating() bool {
	return m.Ratings.IMDb != nil
}

// IMDbRatingFormatted renders the IMDb score and vote count for the
// README, e.g. "8.0 / 10 (27,000 votes)". The rating uses one decimal
// place to match IMDb's own display style.
func (m *MovieInformation) IMDbRatingFormatted() string {
	if m.Ratings.IMDb == nil {
		return ""
	}
	r := m.Ratings.IMDb
	return fmt.Sprintf("%.1f / 10 (%s votes)", r.AverageRating, formatGroupedInt(r.NumVotes))
}

// formatGroupedInt returns a comma-separated decimal representation
// of n (e.g. 1234567 → "1,234,567"). Inlined rather than pulling
// golang.org/x/text/message for one display helper.
func formatGroupedInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// Insert commas from the right.
	var b strings.Builder
	first := len(s) % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(s[:first])
	for i := first; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
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
