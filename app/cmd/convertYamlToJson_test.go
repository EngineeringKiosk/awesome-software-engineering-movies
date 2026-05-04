package cmd

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// mergeMovieInformation has three override-only fields (Language,
// Subtitles, Description) whose YAML > API precedence is easy to
// break by accident on schema changes. These tests pin the contract.

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

func TestMergeMovieInformation_TitlePrecedence(t *testing.T) {
	t.Run("YAML title overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l", Title: "from-yaml"}
		json := &MovieInformation{Name: "stale", Link: "stale", Title: "from-api"}

		got := mergeMovieInformation(yaml, json)
		if got.Title != "from-yaml" {
			t.Fatalf("Title = %q; want %q", got.Title, "from-yaml")
		}
	})

	t.Run("empty YAML title preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l"}
		json := &MovieInformation{Name: "stale", Link: "stale", Title: "from-api"}

		got := mergeMovieInformation(yaml, json)
		if got.Title != "from-api" {
			t.Fatalf("Title = %q; want %q (target preserved)", got.Title, "from-api")
		}
	})
}

func TestMergeMovieInformation_DurationPrecedence(t *testing.T) {
	t.Run("YAML duration overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l", Duration: "PT1H54M"}
		json := &MovieInformation{Name: "stale", Link: "stale", Duration: "PT2H"}

		got := mergeMovieInformation(yaml, json)
		if got.Duration != "PT1H54M" {
			t.Fatalf("Duration = %q; want %q", got.Duration, "PT1H54M")
		}
	})

	t.Run("empty YAML duration preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l"}
		json := &MovieInformation{Name: "stale", Link: "stale", Duration: "PT2H"}

		got := mergeMovieInformation(yaml, json)
		if got.Duration != "PT2H" {
			t.Fatalf("Duration = %q; want %q (target preserved)", got.Duration, "PT2H")
		}
	})
}

func TestMergeMovieInformation_PublishedAtPrecedence(t *testing.T) {
	t.Run("YAML publishedAt overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l", PublishedAt: "2019-07-24T00:00:00Z"}
		json := &MovieInformation{Name: "stale", Link: "stale", PublishedAt: "2019-01-01T00:00:00Z"}

		got := mergeMovieInformation(yaml, json)
		if got.PublishedAt != "2019-07-24T00:00:00Z" {
			t.Fatalf("PublishedAt = %q; want %q", got.PublishedAt, "2019-07-24T00:00:00Z")
		}
	})

	t.Run("empty YAML publishedAt preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l"}
		json := &MovieInformation{Name: "stale", Link: "stale", PublishedAt: "2019-01-01T00:00:00Z"}

		got := mergeMovieInformation(yaml, json)
		if got.PublishedAt != "2019-01-01T00:00:00Z" {
			t.Fatalf("PublishedAt = %q; want %q (target preserved)", got.PublishedAt, "2019-01-01T00:00:00Z")
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

func TestMergeMovieInformation_SubtitlesPrecedence(t *testing.T) {
	t.Run("YAML subtitles override target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l", Subtitles: []string{"de"}}
		json := &MovieInformation{Name: "stale", Link: "stale", Subtitles: []string{"en", "fr"}}

		got := mergeMovieInformation(yaml, json)
		if len(got.Subtitles) != 1 || got.Subtitles[0] != "de" {
			t.Fatalf("Subtitles = %v; want [de]", got.Subtitles)
		}
	})

	t.Run("empty YAML subtitles preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l"} // Subtitles empty
		json := &MovieInformation{Name: "stale", Link: "stale", Subtitles: []string{"en", "de"}}

		got := mergeMovieInformation(yaml, json)
		if len(got.Subtitles) != 2 || got.Subtitles[0] != "en" || got.Subtitles[1] != "de" {
			t.Fatalf("Subtitles = %v; want [en de] (target preserved)", got.Subtitles)
		}
	})
}

func TestResolvePlatform(t *testing.T) {
	cases := []struct {
		name        string
		yamlValue   string
		link        string
		wantValue   string
		wantWarnSub string // empty = expect no warning
	}{
		{
			name:      "autodetect from YouTube link when YAML omits platform",
			link:      "https://www.youtube.com/watch?v=abc123",
			wantValue: "youtube",
		},
		{
			name:      "explicit YAML matches detector → no warning",
			yamlValue: "youtube",
			link:      "https://youtu.be/abc123",
			wantValue: "youtube",
		},
		{
			name:        "YAML disagrees with link → warn, keep YAML",
			yamlValue:   "netflix",
			link:        "https://www.youtube.com/watch?v=abc123",
			wantValue:   "netflix",
			wantWarnSub: "disagrees with link-detected platform",
		},
		{
			name:        "YAML set, link unrecognised → warn, keep YAML",
			yamlValue:   "netflix",
			link:        "https://example.com/foo",
			wantValue:   "netflix",
			wantWarnSub: "matches no known platform; keeping YAML value",
		},
		{
			name:        "neither YAML nor detector → warn, leave empty",
			link:        "https://example.com/foo",
			wantValue:   "",
			wantWarnSub: "leaving empty",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := &MovieInformation{Link: tc.link, Platform: tc.yamlValue}

			var buf bytes.Buffer
			prev := log.Writer()
			log.SetOutput(&buf)
			defer log.SetOutput(prev)

			resolvePlatform(info, "test.yml")

			if info.Platform != tc.wantValue {
				t.Errorf("Platform = %q; want %q", info.Platform, tc.wantValue)
			}
			out := buf.String()
			switch {
			case tc.wantWarnSub == "" && strings.Contains(out, "WARNING"):
				t.Errorf("expected no warning, got: %s", out)
			case tc.wantWarnSub != "" && !strings.Contains(out, tc.wantWarnSub):
				t.Errorf("expected warning containing %q, got: %s", tc.wantWarnSub, out)
			}
		})
	}
}

func TestMergeMovieInformation_LocalizedPrecedence(t *testing.T) {
	t.Run("YAML localized map overrides target", func(t *testing.T) {
		yaml := &MovieInformation{
			Name: "n", Link: "l",
			Localized: map[string]LocalizedVersion{
				"de": {Title: "from-yaml", Link: "yaml-link"},
			},
		}
		json := &MovieInformation{
			Name: "stale", Link: "stale",
			Localized: map[string]LocalizedVersion{
				"de": {Title: "stale-title", Link: "stale-link"},
				"es": {Title: "should-be-dropped"},
			},
		}

		got := mergeMovieInformation(yaml, json)
		if len(got.Localized) != 1 {
			t.Fatalf("Localized = %v; want exactly the YAML map", got.Localized)
		}
		if got.Localized["de"].Title != "from-yaml" || got.Localized["de"].Link != "yaml-link" {
			t.Fatalf("Localized[de] = %+v; want from-yaml/yaml-link", got.Localized["de"])
		}
		if _, ok := got.Localized["es"]; ok {
			t.Errorf("YAML omitting 'es' must drop it; map currently has it")
		}
	})

	t.Run("empty YAML localized preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l"} // Localized empty
		json := &MovieInformation{
			Name: "stale", Link: "stale",
			Localized: map[string]LocalizedVersion{
				"de": {Title: "from-target"},
			},
		}

		got := mergeMovieInformation(yaml, json)
		if len(got.Localized) != 1 || got.Localized["de"].Title != "from-target" {
			t.Fatalf("Localized = %v; want target preserved", got.Localized)
		}
	})
}

func TestValidateLocalized(t *testing.T) {
	cases := []struct {
		name        string
		localized   map[string]LocalizedVersion
		wantWarnSub string // empty = expect no warning
	}{
		{
			name: "well-formed entries do not warn",
			localized: map[string]LocalizedVersion{
				"de": {Title: "Pythons Geschichte", Link: "https://example.com/de"},
				"es": {Title: "La historia de Python"},
				"fr": {Link: "https://example.com/fr"},
				"it": {Description: "Una descrizione localizzata"},
			},
		},
		{
			name:        "uppercase key warns",
			localized:   map[string]LocalizedVersion{"DE": {Title: "x"}},
			wantWarnSub: "is not a 2-letter lowercase code",
		},
		{
			name:        "three-letter key warns",
			localized:   map[string]LocalizedVersion{"deu": {Title: "x"}},
			wantWarnSub: "is not a 2-letter lowercase code",
		},
		{
			name:        "single-letter key warns",
			localized:   map[string]LocalizedVersion{"d": {Title: "x"}},
			wantWarnSub: "is not a 2-letter lowercase code",
		},
		{
			name:        "empty entry warns",
			localized:   map[string]LocalizedVersion{"de": {}},
			wantWarnSub: "no overrides set",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := &MovieInformation{Localized: tc.localized}

			var buf bytes.Buffer
			prev := log.Writer()
			log.SetOutput(&buf)
			defer log.SetOutput(prev)

			validateLocalized(info, "test.yml")

			out := buf.String()
			switch {
			case tc.wantWarnSub == "" && strings.Contains(out, "WARNING"):
				t.Errorf("expected no warning, got: %s", out)
			case tc.wantWarnSub != "" && !strings.Contains(out, tc.wantWarnSub):
				t.Errorf("expected warning containing %q, got: %s", tc.wantWarnSub, out)
			}
		})
	}
}

func TestResolveLocalizedPlatforms(t *testing.T) {
	t.Run("autodetects from localized link", func(t *testing.T) {
		info := &MovieInformation{
			Localized: map[string]LocalizedVersion{
				"de": {Link: "https://www.amazon.de/gp/video/detail/B0FVCKCM81/"},
			},
		}
		var buf bytes.Buffer
		prev := log.Writer()
		log.SetOutput(&buf)
		defer log.SetOutput(prev)

		resolveLocalizedPlatforms(info, "test.yml")

		if got := info.Localized["de"].Platform; got != "amazon_prime_video" {
			t.Fatalf("Localized[de].Platform = %q; want amazon_prime_video", got)
		}
		if strings.Contains(buf.String(), "WARNING") {
			t.Errorf("expected no warning, got: %s", buf.String())
		}
	})

	t.Run("description-only entry is left alone", func(t *testing.T) {
		info := &MovieInformation{
			Localized: map[string]LocalizedVersion{
				"de": {Description: "Eine deutsche Beschreibung"},
			},
		}
		resolveLocalizedPlatforms(info, "test.yml")
		if got := info.Localized["de"].Platform; got != "" {
			t.Errorf("Localized[de].Platform = %q; want empty (no link to detect from)", got)
		}
	})

	t.Run("YAML override on localized link survives", func(t *testing.T) {
		info := &MovieInformation{
			Localized: map[string]LocalizedVersion{
				"de": {Link: "https://www.amazon.de/gp/video/detail/B0/", Platform: "netflix"},
			},
		}
		var buf bytes.Buffer
		prev := log.Writer()
		log.SetOutput(&buf)
		defer log.SetOutput(prev)

		resolveLocalizedPlatforms(info, "test.yml")

		if got := info.Localized["de"].Platform; got != "netflix" {
			t.Errorf("Localized[de].Platform = %q; want %q (YAML wins)", got, "netflix")
		}
		if !strings.Contains(buf.String(), "disagrees") {
			t.Errorf("expected disagreement warning, got: %s", buf.String())
		}
	})
}

func TestMergeMovieInformation_PlatformPrecedence(t *testing.T) {
	t.Run("YAML platform overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l", Platform: "netflix"}
		json := &MovieInformation{Name: "stale", Link: "stale", Platform: "youtube"}

		got := mergeMovieInformation(yaml, json)
		if got.Platform != "netflix" {
			t.Fatalf("Platform = %q; want %q", got.Platform, "netflix")
		}
	})

	t.Run("empty YAML platform preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Link: "l"}
		json := &MovieInformation{Name: "stale", Link: "stale", Platform: "youtube"}

		got := mergeMovieInformation(yaml, json)
		if got.Platform != "youtube" {
			t.Fatalf("Platform = %q; want %q (target preserved)", got.Platform, "youtube")
		}
	})
}
