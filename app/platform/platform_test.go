package platform

import "testing"

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		link string
		slug string
		ok   bool
	}{
		{"youtube watch", "https://www.youtube.com/watch?v=abc123", YouTube, true},
		{"youtube short host", "https://youtu.be/abc123", YouTube, true},
		{"youtube embed", "https://www.youtube.com/embed/abc123", YouTube, true},
		{"youtube shorts", "https://www.youtube.com/shorts/abc123", YouTube, true},
		{"m.youtube", "https://m.youtube.com/watch?v=abc123", YouTube, true},
		{"unknown host", "https://www.netflix.com/title/12345", "", false},
		{"empty link", "", "", false},
		{"garbage", "not a url", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotSlug, gotOK := Detect(tc.link)
			if gotSlug != tc.slug || gotOK != tc.ok {
				t.Fatalf("Detect(%q) = (%q, %v); want (%q, %v)", tc.link, gotSlug, gotOK, tc.slug, tc.ok)
			}
		})
	}
}

func TestDisplay(t *testing.T) {
	cases := map[string]string{
		YouTube:        "YouTube",
		"":             "",
		"netflix":      "netflix",      // unknown slug → returned verbatim
		"amazon_prime": "amazon_prime", // ditto
	}
	for in, want := range cases {
		if got := Display(in); got != want {
			t.Errorf("Display(%q) = %q; want %q", in, got, want)
		}
	}
}
