package cmd

import "testing"

func TestDurationHumanReadable(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Seconds round up at 30+
		{"PT31M50S", "ca. 32 min."},
		{"PT43M22S", "ca. 43 min."},
		// Seconds round down below 30
		{"PT43M29S", "ca. 43 min."},
		{"PT43M30S", "ca. 44 min."},

		// Hour-spanning videos render as "h ... min."
		{"PT1H23M45S", "ca. 1 h 24 min."},
		{"PT2H", "ca. 2 h 0 min."},
		{"PT10H30M", "ca. 10 h 30 min."},

		// Sub-1-minute always renders as "ca. 1 min." rather than 0
		{"PT45S", "ca. 1 min."},
		{"PT1S", "ca. 1 min."},

		// Genuinely zero stays zero
		{"PT0S", "ca. 0 min."},

		// Parse errors return the raw value so the README still renders
		{"", ""},
		{"garbage", "garbage"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			m := &MovieInformation{Duration: tc.in}
			got := m.DurationHumanReadable()
			if got != tc.want {
				t.Fatalf("DurationHumanReadable(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestPlatformDisplay(t *testing.T) {
	cases := map[string]string{
		"youtube":  "YouTube",
		"":         "",
		"unknown":  "unknown", // delegated to platform.Display, returned verbatim
	}
	for slug, want := range cases {
		m := &MovieInformation{Platform: slug}
		if got := m.PlatformDisplay(); got != want {
			t.Errorf("PlatformDisplay(%q) = %q; want %q", slug, got, want)
		}
	}
}

func TestTagsAsList(t *testing.T) {
	m := &MovieInformation{Tags: []string{"Networking", "Service Mesh"}}
	if got, want := m.TagsAsList(), "Networking, Service Mesh"; got != want {
		t.Fatalf("TagsAsList() = %q; want %q", got, want)
	}
}

func TestFormatGroupedInt(t *testing.T) {
	cases := map[int64]string{
		0:        "0",
		7:        "7",
		42:       "42",
		999:      "999",
		1000:     "1,000",
		33455:    "33,455",
		1078731:  "1,078,731",
		-1234567: "-1,234,567",
	}
	for in, want := range cases {
		if got := formatGroupedInt(in); got != want {
			t.Errorf("formatGroupedInt(%d) = %q; want %q", in, got, want)
		}
	}
}

func TestRatingHelpers(t *testing.T) {
	t.Run("no ratings", func(t *testing.T) {
		m := &MovieInformation{}
		if m.HasYouTubeLikes() || m.HasIMDbRating() {
			t.Fatal("empty ratings should report Has...=false")
		}
		if m.YouTubeLikeCountFormatted() != "" || m.IMDbRatingFormatted() != "" {
			t.Fatal("empty ratings should format to \"\"")
		}
	})

	t.Run("youtube only", func(t *testing.T) {
		m := &MovieInformation{Ratings: Ratings{YouTube: &YouTubeRating{LikeCount: 33455}}}
		if !m.HasYouTubeLikes() || m.HasIMDbRating() {
			t.Fatal("youtube-only ratings predicate wrong")
		}
		if got, want := m.YouTubeLikeCountFormatted(), "33,455"; got != want {
			t.Errorf("YouTubeLikeCountFormatted = %q; want %q", got, want)
		}
	})

	t.Run("imdb only", func(t *testing.T) {
		m := &MovieInformation{Ratings: Ratings{IMDb: &IMDbRating{AverageRating: 8.0, NumVotes: 27000}}}
		if !m.HasIMDbRating() {
			t.Fatal("HasIMDbRating should be true")
		}
		if got, want := m.IMDbRatingFormatted(), "8.0 / 10 (27,000 votes)"; got != want {
			t.Errorf("IMDbRatingFormatted = %q; want %q", got, want)
		}
	})
}

func TestNormalizeLanguageCode(t *testing.T) {
	cases := map[string]string{
		"en":      "en",
		"EN":      "en",
		"en-US":   "en",
		"en-us":   "en",
		"de-DE":   "de",
		"  fr  ":  "fr",
		"":        "",
		"zh-Hans": "zh",
	}
	for in, want := range cases {
		if got := normalizeLanguageCode(in); got != want {
			t.Errorf("normalizeLanguageCode(%q) = %q; want %q", in, got, want)
		}
	}
}
