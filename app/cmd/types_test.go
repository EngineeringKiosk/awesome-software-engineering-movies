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

func TestTagsAsList(t *testing.T) {
	m := &MovieInformation{Tags: []string{"Networking", "Service Mesh"}}
	if got, want := m.TagsAsList(), "Networking, Service Mesh"; got != want {
		t.Fatalf("TagsAsList() = %q; want %q", got, want)
	}
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
