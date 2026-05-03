package youtube

import (
	"net/url"
	"strings"
)

// ParseVideoID extracts the YouTube video ID from a URL.
// Supports:
//   - https://www.youtube.com/watch?v=<id>
//   - https://youtube.com/watch?v=<id>
//   - https://youtu.be/<id>
//   - https://www.youtube.com/embed/<id>
//   - https://www.youtube.com/shorts/<id>
//
// Returns the bare video ID and ok=true on success, or "" and
// ok=false if the URL is malformed or not a recognised YouTube URL.
func ParseVideoID(rawURL string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", false
	}

	switch strings.ToLower(u.Host) {
	case "youtu.be":
		id := strings.Trim(u.Path, "/")
		if id == "" {
			return "", false
		}
		return id, true

	case "youtube.com", "www.youtube.com", "m.youtube.com":
		if id := u.Query().Get("v"); id != "" {
			return id, true
		}

		// /embed/<id>, /shorts/<id>, /v/<id>
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) == 2 && (parts[0] == "embed" || parts[0] == "shorts" || parts[0] == "v") {
			return parts[1], true
		}
	}

	return "", false
}
