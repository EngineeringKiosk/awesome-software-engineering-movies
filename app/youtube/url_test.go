package youtube

import "testing"

func TestParseVideoID(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
		wantOk bool
	}{
		{"watch", "https://www.youtube.com/watch?v=uaksVVHDhYU", "uaksVVHDhYU", true},
		{"watch_no_www", "https://youtube.com/watch?v=HDKUEXBF3B4", "HDKUEXBF3B4", true},
		{"watch_extra_params", "https://www.youtube.com/watch?v=bmWQqAKLgT4&t=42s", "bmWQqAKLgT4", true},
		{"short", "https://youtu.be/uaksVVHDhYU", "uaksVVHDhYU", true},
		{"short_with_query", "https://youtu.be/uaksVVHDhYU?si=abc", "uaksVVHDhYU", true},
		{"embed", "https://www.youtube.com/embed/uaksVVHDhYU", "uaksVVHDhYU", true},
		{"shorts", "https://www.youtube.com/shorts/uaksVVHDhYU", "uaksVVHDhYU", true},
		{"mobile", "https://m.youtube.com/watch?v=uaksVVHDhYU", "uaksVVHDhYU", true},
		{"trim_whitespace", "  https://youtu.be/uaksVVHDhYU  ", "uaksVVHDhYU", true},
		{"empty", "", "", false},
		{"non_youtube", "https://vimeo.com/12345", "", false},
		{"youtube_root", "https://www.youtube.com/", "", false},
		{"youtu_be_no_id", "https://youtu.be/", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseVideoID(tc.input)
			if got != tc.want || ok != tc.wantOk {
				t.Fatalf("ParseVideoID(%q) = (%q, %v); want (%q, %v)", tc.input, got, ok, tc.want, tc.wantOk)
			}
		})
	}
}

func TestIsNonVideoURL(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"playlist", "https://www.youtube.com/playlist?list=PL7nj3G6Jpv2G6Gp6NvN1kUtQuW8QshBWE", true},
		{"playlist_no_www", "https://youtube.com/playlist?list=PLabc", true},
		{"show", "https://www.youtube.com/show/SC2L5QE7tgqHgriBrZQBfT4g", true},
		{"video_watch", "https://www.youtube.com/watch?v=uaksVVHDhYU", false},
		{"video_short", "https://youtu.be/uaksVVHDhYU", false},
		{"non_youtube", "https://vimeo.com/12345", false},
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNonVideoURL(tc.input); got != tc.want {
				t.Fatalf("IsNonVideoURL(%q) = %v; want %v", tc.input, got, tc.want)
			}
		})
	}
}
