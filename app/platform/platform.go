// Package platform identifies which streaming source a movie link
// belongs to. It is the single home for the slugs that land in the
// generated JSON and for the human-readable names rendered into the
// README. Adding a new source — Netflix, Amazon Prime, Vimeo — means
// touching this one package and nothing else.
package platform

import (
	"github.com/EngineeringKiosk/awesome-software-engineering-movies/youtube"
)

// Slug constants. The string values land in the generated JSON, so
// changing them is a breaking change for downstream consumers.
const (
	YouTube = "youtube"
)

// displayNames maps slugs to the human-readable names rendered into
// the README. Slugs missing from this map are returned by Display
// unchanged so the rendered output degrades gracefully when a future
// platform is referenced before its display name is wired up.
var displayNames = map[string]string{
	YouTube: "YouTube",
}

// Detect inspects a link and returns the platform slug it belongs to,
// or ok=false when the link does not match any known platform.
//
// Today only YouTube is recognised; future sources slot in as
// additional cases below. The YouTube branch defers to
// youtube.ParseVideoID, which already understands every URL shape
// the project accepts (`youtu.be`, `youtube.com/watch?v=`, `/embed/`,
// `/shorts/`).
func Detect(link string) (string, bool) {
	if _, ok := youtube.ParseVideoID(link); ok {
		return YouTube, true
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
