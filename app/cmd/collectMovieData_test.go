package cmd

import (
	"reflect"
	"testing"
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
