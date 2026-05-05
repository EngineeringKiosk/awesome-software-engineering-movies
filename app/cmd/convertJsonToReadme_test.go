package cmd

import "testing"

func TestGroupMoviesByType(t *testing.T) {
	doc := &MovieInformation{Name: "Doc", Type: "Documentary"}
	tv := &MovieInformation{Name: "TV", Type: "TV Series"}
	movie := &MovieInformation{Name: "Movie", Type: "Movie"}
	weird := &MovieInformation{Name: "Weird", Type: "Reality Show"}
	noType := &MovieInformation{Name: "NoType"}

	t.Run("empty input", func(t *testing.T) {
		g := groupMoviesByType(nil)
		if g.Total != 0 || len(g.TVSeries) != 0 || len(g.Documentaries) != 0 || len(g.Movies) != 0 {
			t.Fatalf("expected all empty, got %+v", g)
		}
	})

	t.Run("only documentaries", func(t *testing.T) {
		g := groupMoviesByType([]*MovieInformation{doc, doc, doc})
		if len(g.Documentaries) != 3 || len(g.TVSeries) != 0 || len(g.Movies) != 0 {
			t.Fatalf("expected 3 documentaries only, got %+v", g)
		}
		if g.Total != 3 {
			t.Errorf("Total = %d; want 3", g.Total)
		}
	})

	t.Run("mixed types land in their buckets", func(t *testing.T) {
		g := groupMoviesByType([]*MovieInformation{doc, tv, movie})
		if len(g.Documentaries) != 1 || g.Documentaries[0] != doc {
			t.Errorf("Documentaries = %+v; want [doc]", g.Documentaries)
		}
		if len(g.TVSeries) != 1 || g.TVSeries[0] != tv {
			t.Errorf("TVSeries = %+v; want [tv]", g.TVSeries)
		}
		if len(g.Movies) != 1 || g.Movies[0] != movie {
			t.Errorf("Movies = %+v; want [movie]", g.Movies)
		}
	})

	t.Run("unknown and empty types fall into Documentaries", func(t *testing.T) {
		g := groupMoviesByType([]*MovieInformation{weird, noType})
		if len(g.Documentaries) != 2 {
			t.Errorf("Documentaries = %+v; want 2 fallbacks", g.Documentaries)
		}
		if len(g.TVSeries) != 0 || len(g.Movies) != 0 {
			t.Errorf("TVSeries/Movies should be empty for unknown/empty types")
		}
	})

	t.Run("input order is preserved within each bucket", func(t *testing.T) {
		// Two documentaries appended in order — bucketing is stable
		// since we range the input once and append per type.
		a := &MovieInformation{Name: "A", Type: "Documentary"}
		b := &MovieInformation{Name: "B", Type: "Documentary"}
		g := groupMoviesByType([]*MovieInformation{a, b})
		if g.Documentaries[0] != a || g.Documentaries[1] != b {
			t.Errorf("order not preserved; got %+v", g.Documentaries)
		}
	})
}
