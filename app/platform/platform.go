// Package platform identifies which streaming source a movie link
// belongs to. It is the single home for the slugs that land in the
// generated JSON and for the human-readable names rendered into the
// README. Adding a new source means touching this one package and
// nothing else.
package platform

import (
	"net/url"
	"strings"

	"github.com/EngineeringKiosk/awesome-software-engineering-movies/youtube"
)

// Slug constants. The string values land in the generated JSON, so
// changing them is a breaking change for downstream consumers.
const (
	YouTube          = "youtube"
	Netflix          = "netflix"
	AmazonPrimeVideo = "amazon_prime_video"
	BPB              = "bpb"
	HBOMax           = "hbo_max"
	AppleTV          = "apple_tv"
	RTLPlus          = "rtl_plus"
)

// displayNames maps slugs to the human-readable names rendered into
// the README. Slugs missing from this map are returned by Display
// unchanged so the rendered output degrades gracefully when a future
// platform is referenced before its display name is wired up.
var displayNames = map[string]string{
	YouTube:          "YouTube",
	Netflix:          "Netflix",
	AmazonPrimeVideo: "Amazon Prime Video",
	BPB:              "Bundeszentrale für politische Bildung",
	HBOMax:           "HBO Max",
	AppleTV:          "Apple TV",
	RTLPlus:          "RTL+",
}

// Detect inspects a link and returns the platform slug it belongs to,
// or ok=false when the link does not match any known platform.
//
// New platforms slot in as additional cases below. The YouTube branch
// defers to youtube.ParseVideoID, which already understands every URL
// shape the project accepts (`youtu.be`, `youtube.com/watch?v=`,
// `/embed/`, `/shorts/`).
func Detect(link string) (string, bool) {
	if _, ok := youtube.ParseVideoID(link); ok {
		return YouTube, true
	}

	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil || u.Host == "" {
		return "", false
	}
	host := strings.ToLower(u.Host)
	// Strip a leading "www." so detector cases can compare bare
	// brand hosts without listing both forms.
	host = strings.TrimPrefix(host, "www.")
	path := u.Path

	switch {
	// Netflix: only catalogue pages of the form /title/<id>. Browse
	// or marketing URLs land elsewhere on the same host and shouldn't
	// be misclassified.
	case host == "netflix.com" && strings.HasPrefix(path, "/title/"):
		return Netflix, true

	// Amazon Prime Video: any amazon.<tld> host where the path lives
	// under /gp/video/ or /video/. The host alone is not enough —
	// amazon.de also serves books and physical goods.
	// primevideo.com is the dedicated Prime Video host (any /detail/
	// page); it is video-only so the host check is sufficient.
	case strings.HasPrefix(host, "amazon.") &&
		(strings.HasPrefix(path, "/gp/video/") || strings.HasPrefix(path, "/video/")):
		return AmazonPrimeVideo, true
	case host == "primevideo.com" && strings.Contains(path, "/detail/"):
		return AmazonPrimeVideo, true

	// bpb: Bundeszentrale für politische Bildung media library.
	// Scoped to /mediathek/ — the same host hosts non-video content
	// (essays, dossiers) we don't want to misclassify.
	case host == "bpb.de" && strings.HasPrefix(path, "/mediathek/"):
		return BPB, true

	// HBO Max: WB's streaming service, with both the legacy
	// play.hbomax.com domain and the newer max.com brand.
	case (host == "play.hbomax.com" || host == "hbomax.com" || host == "max.com") &&
		(strings.HasPrefix(path, "/video/") || strings.HasPrefix(path, "/title/") || strings.HasPrefix(path, "/movie/") || strings.HasPrefix(path, "/show/")):
		return HBOMax, true

	// Apple TV: tv.apple.com hosts both the rental store and the
	// Apple TV+ subscription catalogue. URLs include a locale segment
	// (e.g. /de/movie/...), so match on the substring rather than
	// the path prefix.
	case host == "tv.apple.com" &&
		(strings.Contains(path, "/movie/") || strings.Contains(path, "/show/") || strings.Contains(path, "/episode/")):
		return AppleTV, true

	// RTL+: German free + premium streaming service.
	case host == "plus.rtl.de":
		return RTLPlus, true
	}

	return "", false
}

// Display returns the human-readable platform name for a slug
// (e.g. "youtube" → "YouTube"). Empty input returns empty output.
// Unknown slugs are returned verbatim — better to render a slightly
// unpolished label than to swallow data the template asked for.
func Display(slug string) string {
	if name, ok := displayNames[slug]; ok {
		return name
	}
	return slug
}

// IsKnown reports whether slug is one of the platforms the tooling
// has a detector and display name for. Unknown slugs are accepted
// into the schema (callers may want to pre-declare a platform
// before its detector lands), but URL/slug consistency is only
// checked for known ones.
func IsKnown(slug string) bool {
	_, ok := displayNames[slug]
	return ok
}
