package cmd

import (
	"reflect"
	"testing"
	"time"
)

func TestMergeLanguageCodes(t *testing.T) {
	cases := []struct {
		name string
		yaml []string
		api  []string
		want []string
	}{
		{
			name: "both empty",
			want: nil,
		},
		{
			name: "yaml empty, api populated",
			api:  []string{"en"},
			want: []string{"en"},
		},
		{
			name: "yaml populated, api empty",
			yaml: []string{"de", "en"},
			want: []string{"de", "en"},
		},
		{
			name: "overlap is deduped, yaml order preserved",
			yaml: []string{"en"},
			api:  []string{"en", "de"},
			want: []string{"en", "de"},
		},
		{
			name: "BCP-47 tags normalize to ISO 639-1 before union",
			yaml: []string{"en-US"},
			api:  []string{"de-DE", "EN"},
			want: []string{"en", "de"},
		},
		{
			name: "empty/whitespace entries are dropped",
			yaml: []string{"", " "},
			api:  []string{"fr"},
			want: []string{"fr"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeLanguageCodes(tc.yaml, tc.api)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("mergeLanguageCodes(%v, %v) = %v; want %v", tc.yaml, tc.api, got, tc.want)
			}
		})
	}
}

func TestNormalizeLanguageCodes(t *testing.T) {
	got := normalizeLanguageCodes([]string{"en-US", "", "DE", "fr-FR"})
	want := []string{"en", "de", "fr"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeLanguageCodes = %v; want %v", got, want)
	}
	if normalizeLanguageCodes(nil) != nil {
		t.Fatalf("normalizeLanguageCodes(nil) should be nil")
	}
}

func TestNeedsIMDbRefresh(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	rfc := func(d time.Duration) string {
		return now.Add(-d).Format(time.RFC3339)
	}

	cases := []struct {
		name  string
		info  *MovieInformation
		force bool
		want  bool
	}{
		{
			name: "no imdbID never triggers refresh",
			info: &MovieInformation{},
			want: false,
		},
		{
			name: "no imdbID is not overridden by --force-imdb-refresh",
			info: &MovieInformation{}, force: true, want: false,
		},
		{
			name: "imdbID set but no rating block yet → refresh",
			info: &MovieInformation{IMDbID: "tt0000001"},
			want: true,
		},
		{
			name: "rating block fresh (10 days old) → no refresh",
			info: &MovieInformation{
				IMDbID:  "tt0000001",
				Ratings: Ratings{IMDb: &IMDbRating{RefreshedAt: rfc(10 * 24 * time.Hour)}},
			},
			want: false,
		},
		{
			name: "rating block older than 30 days → refresh",
			info: &MovieInformation{
				IMDbID:  "tt0000001",
				Ratings: Ratings{IMDb: &IMDbRating{RefreshedAt: rfc(40 * 24 * time.Hour)}},
			},
			want: true,
		},
		{
			name: "force flag refreshes even fresh data",
			info: &MovieInformation{
				IMDbID:  "tt0000001",
				Ratings: Ratings{IMDb: &IMDbRating{RefreshedAt: rfc(10 * 24 * time.Hour)}},
			},
			force: true,
			want:  true,
		},
		{
			name: "unparseable timestamp triggers refresh",
			info: &MovieInformation{
				IMDbID:  "tt0000001",
				Ratings: Ratings{IMDb: &IMDbRating{RefreshedAt: "not a date"}},
			},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := needsIMDbRefresh(tc.info, tc.force, now)
			if got != tc.want {
				t.Fatalf("needsIMDbRefresh = %v; want %v", got, tc.want)
			}
		})
	}
}
