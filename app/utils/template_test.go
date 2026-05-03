package utils

import (
	"strings"
	"testing"
)

func TestTruncateDescription(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{
			name: "empty",
			in:   "",
			max:  10,
			want: "",
		},
		{
			name: "shorter than max",
			in:   "hello",
			max:  10,
			want: "hello",
		},
		{
			name: "exactly max",
			in:   "hello world",
			max:  11,
			want: "hello world",
		},
		{
			name: "boundary at space",
			in:   "hello world how are you",
			max:  5,
			want: "hello ...",
		},
		{
			name: "mid-word cut advances to next space",
			in:   "hello beautiful world",
			max:  8, // mid 'beautiful'
			want: "hello beautiful ...",
		},
		{
			name: "mid-word cut advances to sentence end",
			in:   "hello beautiful. world",
			max:  8, // mid 'beautiful'; period comes before next space
			want: "hello beautiful. ...",
		},
		{
			name: "previous rune is sentence terminator",
			in:   "Wow! Another sentence here.",
			max:  4, // cut after the '!'
			want: "Wow! ...",
		},
		{
			name: "exclamation in mid-word lookahead",
			in:   "no boundary yet wait! more",
			max:  18, // mid 'wait'
			want: "no boundary yet wait! ...",
		},
		{
			name: "first boundary wins (space before next sentence end)",
			in:   "is this thing on? yes it is",
			max:  10, // mid 'thing'; next boundary is the space after 'thing'
			want: "is this thing ...",
		},
		{
			name: "trims trailing whitespace before ellipsis",
			in:   "hello   world",
			max:  5,
			want: "hello ...",
		},
		{
			name: "single giant word returns unchanged",
			in:   strings.Repeat("a", 50),
			max:  10,
			want: strings.Repeat("a", 50),
		},
		{
			name: "multibyte / emoji counted as runes",
			// 6 runes: "héllo👋"; max 5 should cut, no boundary inside
			in:   "héllo👋 world",
			max:  5, // mid; rune 5 is the emoji
			want: "héllo👋 ...",
		},
		{
			name: "max zero returns input unchanged",
			in:   "anything",
			max:  0,
			want: "anything",
		},
		{
			name: "negative max returns input unchanged",
			in:   "anything",
			max:  -1,
			want: "anything",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TruncateDescription(tc.in, tc.max)
			if got != tc.want {
				t.Fatalf("TruncateDescription(%q, %d)\n  got:  %q\n  want: %q", tc.in, tc.max, got, tc.want)
			}
		})
	}
}
