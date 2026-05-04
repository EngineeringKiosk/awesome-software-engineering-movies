package cmd

import "testing"

// mergeMovieInformation has two override-only fields (Language,
// Description) whose YAML > API precedence is easy to break by
// accident on schema changes. These tests pin the contract.

func TestMergeMovieInformation_DescriptionPrecedence(t *testing.T) {
	t.Run("YAML description overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l", Description: "from-yaml"}
		json := &MovieInformation{Name: "stale", Link: "stale", Description: "from-api"}

		got := mergeMovieInformation(yaml, json)
		if got.Description != "from-yaml" {
			t.Fatalf("Description = %q; want %q", got.Description, "from-yaml")
		}
	})

	t.Run("empty YAML description preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l"} // Description empty
		json := &MovieInformation{Name: "stale", Link: "stale", Description: "from-api"}

		got := mergeMovieInformation(yaml, json)
		if got.Description != "from-api" {
			t.Fatalf("Description = %q; want %q (target preserved)", got.Description, "from-api")
		}
	})
}

func TestMergeMovieInformation_LanguagePrecedence(t *testing.T) {
	t.Run("YAML language overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l", Language: []string{"de"}}
		json := &MovieInformation{Name: "stale", Link: "stale", Language: []string{"en"}}

		got := mergeMovieInformation(yaml, json)
		if len(got.Language) != 1 || got.Language[0] != "de" {
			t.Fatalf("Language = %v; want [de]", got.Language)
		}
	})

	t.Run("empty YAML language preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l"} // Language empty
		json := &MovieInformation{Name: "stale", Link: "stale", Language: []string{"en"}}

		got := mergeMovieInformation(yaml, json)
		if len(got.Language) != 1 || got.Language[0] != "en" {
			t.Fatalf("Language = %v; want [en] (target preserved)", got.Language)
		}
	})
}
