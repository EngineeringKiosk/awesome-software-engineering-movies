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

		{"netflix title", "https://www.netflix.com/title/80117542", Netflix, true},
		{"netflix bare host", "https://netflix.com/title/12345", Netflix, true},
		{"netflix browse not detected", "https://www.netflix.com/browse", "", false},
		{"netflix root not detected", "https://www.netflix.com/", "", false},

		{"amazon prime gp/video", "https://www.amazon.de/gp/video/detail/B0FVCKCM81/", AmazonPrimeVideo, true},
		{"amazon prime /video", "https://www.amazon.com/video/detail/abc", AmazonPrimeVideo, true},
		{"amazon co.uk video", "https://www.amazon.co.uk/gp/video/detail/abc", AmazonPrimeVideo, true},
		{"amazon product page not detected", "https://www.amazon.de/dp/B08XXX", "", false},
		{"amazon home not detected", "https://www.amazon.com/", "", false},

		{"bpb mediathek", "https://www.bpb.de/mediathek/video/273199/the-cleaners/", BPB, true},
		{"bpb bare host", "https://bpb.de/mediathek/video/123/foo/", BPB, true},
		{"bpb non-mediathek not detected", "https://www.bpb.de/themen/medien/", "", false},

		{"unknown host", "https://www.example.com/title/12345", "", false},
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

func TestIsKnown(t *testing.T) {
	cases := map[string]bool{
		YouTube:          true,
		Netflix:          true,
		AmazonPrimeVideo: true,
		BPB:              true,
		"vimeo":          false,
		"":               false,
	}
	for slug, want := range cases {
		if got := IsKnown(slug); got != want {
			t.Errorf("IsKnown(%q) = %v; want %v", slug, got, want)
		}
	}
}

func TestDisplay(t *testing.T) {
	cases := map[string]string{
		YouTube:          "YouTube",
		Netflix:          "Netflix",
		AmazonPrimeVideo: "Amazon Prime Video",
		BPB:              "Bundeszentrale für politische Bildung",
		"":               "",
		"vimeo":          "vimeo", // unknown slug → returned verbatim
	}
	for in, want := range cases {
		if got := Display(in); got != want {
			t.Errorf("Display(%q) = %q; want %q", in, got, want)
		}
	}
}
