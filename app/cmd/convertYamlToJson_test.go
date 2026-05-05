package cmd

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// stubLinks is a reusable test fixture so each precedence test does
// not have to re-spell the map literal — and so swapping the link
// schema again later is a single-line change here.
func stubLinks() map[string]string {
	return map[string]string{"youtube": "https://www.youtube.com/watch?v=abc123"}
}

func TestMergeMovieInformation_DescriptionPrecedence(t *testing.T) {
	t.Run("YAML description overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks(), Description: "from-yaml"}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Description: "from-api"}

		got := mergeMovieInformation(yaml, json)
		if got.Description != "from-yaml" {
			t.Fatalf("Description = %q; want %q", got.Description, "from-yaml")
		}
	})

	t.Run("empty YAML description preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks()} // Description empty
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Description: "from-api"}

		got := mergeMovieInformation(yaml, json)
		if got.Description != "from-api" {
			t.Fatalf("Description = %q; want %q (target preserved)", got.Description, "from-api")
		}
	})
}

func TestMergeMovieInformation_TitlePrecedence(t *testing.T) {
	t.Run("YAML title overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks(), Title: "from-yaml"}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Title: "from-api"}

		got := mergeMovieInformation(yaml, json)
		if got.Title != "from-yaml" {
			t.Fatalf("Title = %q; want %q", got.Title, "from-yaml")
		}
	})

	t.Run("empty YAML title preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks()}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Title: "from-api"}

		got := mergeMovieInformation(yaml, json)
		if got.Title != "from-api" {
			t.Fatalf("Title = %q; want %q (target preserved)", got.Title, "from-api")
		}
	})
}

func TestMergeMovieInformation_DurationPrecedence(t *testing.T) {
	t.Run("YAML duration overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks(), Duration: "PT1H54M"}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Duration: "PT2H"}

		got := mergeMovieInformation(yaml, json)
		if got.Duration != "PT1H54M" {
			t.Fatalf("Duration = %q; want %q", got.Duration, "PT1H54M")
		}
	})

	t.Run("empty YAML duration preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks()}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Duration: "PT2H"}

		got := mergeMovieInformation(yaml, json)
		if got.Duration != "PT2H" {
			t.Fatalf("Duration = %q; want %q (target preserved)", got.Duration, "PT2H")
		}
	})
}

func TestMergeMovieInformation_PublishedAtPrecedence(t *testing.T) {
	t.Run("YAML publishedAt overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks(), PublishedAt: "2019-07-24T00:00:00Z"}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), PublishedAt: "2019-01-01T00:00:00Z"}

		got := mergeMovieInformation(yaml, json)
		if got.PublishedAt != "2019-07-24T00:00:00Z" {
			t.Fatalf("PublishedAt = %q; want %q", got.PublishedAt, "2019-07-24T00:00:00Z")
		}
	})

	t.Run("empty YAML publishedAt preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks()}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), PublishedAt: "2019-01-01T00:00:00Z"}

		got := mergeMovieInformation(yaml, json)
		if got.PublishedAt != "2019-01-01T00:00:00Z" {
			t.Fatalf("PublishedAt = %q; want %q (target preserved)", got.PublishedAt, "2019-01-01T00:00:00Z")
		}
	})
}

func TestMergeMovieInformation_LanguagePrecedence(t *testing.T) {
	t.Run("YAML language overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks(), Language: []string{"de"}}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Language: []string{"en"}}

		got := mergeMovieInformation(yaml, json)
		if len(got.Language) != 1 || got.Language[0] != "de" {
			t.Fatalf("Language = %v; want [de]", got.Language)
		}
	})

	t.Run("empty YAML language preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks()} // Language empty
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Language: []string{"en"}}

		got := mergeMovieInformation(yaml, json)
		if len(got.Language) != 1 || got.Language[0] != "en" {
			t.Fatalf("Language = %v; want [en] (target preserved)", got.Language)
		}
	})
}

func TestMergeMovieInformation_SubtitlesPrecedence(t *testing.T) {
	t.Run("YAML subtitles override target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks(), Subtitles: []string{"de"}}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Subtitles: []string{"en", "fr"}}

		got := mergeMovieInformation(yaml, json)
		if len(got.Subtitles) != 1 || got.Subtitles[0] != "de" {
			t.Fatalf("Subtitles = %v; want [de]", got.Subtitles)
		}
	})

	t.Run("empty YAML subtitles preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks()} // Subtitles empty
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Subtitles: []string{"en", "de"}}

		got := mergeMovieInformation(yaml, json)
		if len(got.Subtitles) != 2 || got.Subtitles[0] != "en" || got.Subtitles[1] != "de" {
			t.Fatalf("Subtitles = %v; want [en de] (target preserved)", got.Subtitles)
		}
	})
}

func TestMergeMovieInformation_LinksReplace(t *testing.T) {
	t.Run("YAML links replace target wholesale", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: map[string]string{"netflix": "yaml-nf"}}
		json := &MovieInformation{Name: "stale", Links: map[string]string{"youtube": "stale-yt"}}

		got := mergeMovieInformation(yaml, json)
		if len(got.Links) != 1 || got.Links["netflix"] != "yaml-nf" {
			t.Fatalf("Links = %v; want exactly the YAML map", got.Links)
		}
	})
}

func TestValidateLinks(t *testing.T) {
	cases := []struct {
		name        string
		links       map[string]string
		wantWarnSub string // empty = expect no warning
	}{
		{
			name: "matching slug + URL emits no warning",
			links: map[string]string{
				"youtube": "https://www.youtube.com/watch?v=abc",
				"netflix": "https://www.netflix.com/title/12345",
			},
		},
		{
			name:  "empty map emits no warning",
			links: map[string]string{},
		},
		{
			name:  "unknown slug passes through silently",
			links: map[string]string{"vimeo": "https://vimeo.com/12345"},
		},
		{
			name:        "known slug with mismatched URL warns",
			links:       map[string]string{"netflix": "https://www.youtube.com/watch?v=abc"},
			wantWarnSub: "looks like \"youtube\", not \"netflix\"",
		},
		{
			name:        "known slug with unrecognised URL warns",
			links:       map[string]string{"netflix": "https://example.com/foo"},
			wantWarnSub: "matches no known platform",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := log.Writer()
			log.SetOutput(&buf)
			defer log.SetOutput(prev)

			validateLinks(tc.links, "test.yml")

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
			Name: "n", Links: stubLinks(),
			Localized: map[string]LocalizedVersion{
				"de": {Title: "from-yaml", Links: map[string]string{"amazon_prime_video": "yaml-link"}},
			},
		}
		json := &MovieInformation{
			Name: "stale", Links: stubLinks(),
			Localized: map[string]LocalizedVersion{
				"de": {Title: "stale-title", Links: map[string]string{"amazon_prime_video": "stale-link"}},
				"es": {Title: "should-be-dropped"},
			},
		}

		got := mergeMovieInformation(yaml, json)
		if len(got.Localized) != 1 {
			t.Fatalf("Localized = %v; want exactly the YAML map", got.Localized)
		}
		if got.Localized["de"].Title != "from-yaml" || got.Localized["de"].Links["amazon_prime_video"] != "yaml-link" {
			t.Fatalf("Localized[de] = %+v; want from-yaml + yaml-link", got.Localized["de"])
		}
		if _, ok := got.Localized["es"]; ok {
			t.Errorf("YAML omitting 'es' must drop it; map currently has it")
		}
	})

	t.Run("empty YAML localized preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks()} // Localized empty
		json := &MovieInformation{
			Name: "stale", Links: stubLinks(),
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

func TestMergeMovieInformation_CategoryPrecedence(t *testing.T) {
	t.Run("YAML category overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks(), Category: "Programming Languages"}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Category: "Culture / Society"}

		got := mergeMovieInformation(yaml, json)
		if got.Category != "Programming Languages" {
			t.Fatalf("Category = %q; want %q", got.Category, "Programming Languages")
		}
	})

	t.Run("empty YAML category preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks()}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Category: "Culture / Society"}

		got := mergeMovieInformation(yaml, json)
		if got.Category != "Culture / Society" {
			t.Fatalf("Category = %q; want %q (target preserved)", got.Category, "Culture / Society")
		}
	})
}

func TestMergeMovieInformation_TypePrecedence(t *testing.T) {
	t.Run("YAML type overrides target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks(), Type: "Movie"}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Type: "Documentary"}

		got := mergeMovieInformation(yaml, json)
		if got.Type != "Movie" {
			t.Fatalf("Type = %q; want %q", got.Type, "Movie")
		}
	})

	t.Run("empty YAML type preserves target", func(t *testing.T) {
		yaml := &MovieInformation{Name: "n", Links: stubLinks()}
		json := &MovieInformation{Name: "stale", Links: stubLinks(), Type: "Documentary"}

		got := mergeMovieInformation(yaml, json)
		if got.Type != "Documentary" {
			t.Fatalf("Type = %q; want %q (target preserved)", got.Type, "Documentary")
		}
	})
}

func TestValidateCategoryAndType(t *testing.T) {
	cases := []struct {
		name        string
		category    string
		movieType   string
		wantWarnSub string // empty = expect no warning
	}{
		{
			name:      "valid category and type emit no warning",
			category:  "Programming Languages",
			movieType: "Documentary",
		},
		{
			name:        "empty category warns",
			movieType:   "Documentary",
			wantWarnSub: "category is required",
		},
		{
			name:        "empty type warns",
			category:    "Programming Languages",
			wantWarnSub: "type is required",
		},
		{
			name:        "unknown category warns",
			category:    "Bogus",
			movieType:   "Documentary",
			wantWarnSub: `category "Bogus" is not in the recommended set`,
		},
		{
			name:        "unknown type warns",
			category:    "Programming Languages",
			movieType:   "Reality Show",
			wantWarnSub: `type "Reality Show" is not in the recommended set`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := &MovieInformation{Category: tc.category, Type: tc.movieType}

			var buf bytes.Buffer
			prev := log.Writer()
			log.SetOutput(&buf)
			defer log.SetOutput(prev)

			validateCategoryAndType(info, "test.yml")

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

func TestValidateLocalized(t *testing.T) {
	cases := []struct {
		name        string
		localized   map[string]LocalizedVersion
		wantWarnSub string // empty = expect no warning
	}{
		{
			name: "well-formed entries do not warn",
			localized: map[string]LocalizedVersion{
				"de": {Title: "Pythons Geschichte", Links: map[string]string{"netflix": "https://www.netflix.com/title/12"}},
				"es": {Title: "La historia de Python"},
				"fr": {Links: map[string]string{"youtube": "https://www.youtube.com/watch?v=fr"}},
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
