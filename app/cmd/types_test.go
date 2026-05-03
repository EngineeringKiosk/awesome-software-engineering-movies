package cmd

import "testing"

func TestDurationHumanReadable(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"PT43M22S", "43:22"},
		{"PT1H23M45S", "1:23:45"},
		{"PT2H", "2:00:00"},
		{"PT45S", "0:45"},
		{"PT0S", "0:00"},
		{"", ""},               // returns raw on parse error
		{"garbage", "garbage"}, // returns raw on parse error
		{"PT10H30M", "10:30:00"},
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
