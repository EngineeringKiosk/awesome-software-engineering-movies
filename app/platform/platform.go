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
	case strings.HasPrefix(host, "amazon.") &&
		(strings.HasPrefix(path, "/gp/video/") || strings.HasPrefix(path, "/video/")):
		return AmazonPrimeVideo, true

	// bpb: Bundeszentrale für politische Bildung media library.
	// Scoped to /mediathek/ — the same host hosts non-video content
	// (essays, dossiers) we don't want to misclassify.
	case host == "bpb.de" && strings.HasPrefix(path, "/mediathek/"):
		return BPB, true
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
