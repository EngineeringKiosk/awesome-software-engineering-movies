package imdb

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func gzipFixture(t *testing.T, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(body)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestParseRatingsTSV(t *testing.T) {
	const body = "tconst\taverageRating\tnumVotes\n" +
		"tt0000001\t5.7\t2103\n" +
		"tt0000002\t5.6\t300\n" +
		"tt0000003\t6.4\t2100\n" +
		"tt0000004\t5.3\t189\n" +
		"malformed line without enough columns\n" +
		"tt9999999\tnotANumber\t10\n" +
		"tt0000005\t6.1\t688\n"

	want := map[string]struct{}{
		"tt0000002": {},
		"tt0000004": {},
		"tt0000005": {},
		"tt9999999": {}, // malformed numeric → must not appear in output
		"tt0000999": {}, // not in fixture → must not appear in output
	}

	got, err := parseRatingsTSV(bytes.NewReader(gzipFixture(t, body)), want)
	if err != nil {
		t.Fatalf("parseRatingsTSV: %v", err)
	}

	cases := []struct {
		id     string
		rating float64
		votes  int64
	}{
		{"tt0000002", 5.6, 300},
		{"tt0000004", 5.3, 189},
		{"tt0000005", 6.1, 688},
	}
	for _, c := range cases {
		r, ok := got[c.id]
		if !ok {
			t.Errorf("%s missing from result", c.id)
			continue
		}
		if r.AverageRating != c.rating {
			t.Errorf("%s rating: got %v, want %v", c.id, r.AverageRating, c.rating)
		}
		if r.NumVotes != c.votes {
			t.Errorf("%s votes: got %d, want %d", c.id, r.NumVotes, c.votes)
		}
	}
	if _, ok := got["tt9999999"]; ok {
		t.Error("malformed numeric row should be skipped, not returned")
	}
	if _, ok := got["tt0000999"]; ok {
		t.Error("missing tconst should not appear in result map")
	}
	if len(got) != 3 {
		t.Errorf("len(got) = %d, want 3", len(got))
	}
}

func TestFetchRatings_emptyIDs(t *testing.T) {
	got, err := FetchRatings(context.Background(), nil)
	if err != nil {
		t.Fatalf("FetchRatings(nil): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}

func TestFetchRatings_endToEnd(t *testing.T) {
	body := "tconst\taverageRating\tnumVotes\n" +
		"tt0308808\t7.2\t6500\n" +
		"tt3268458\t8.0\t27000\n"
	gz := gzipFixture(t, body)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("user-agent: got %q, want %q", r.Header.Get("User-Agent"), userAgent)
		}
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(gz)
	}))
	defer srv.Close()

	prev := DatasetURL
	DatasetURL = srv.URL
	defer func() { DatasetURL = prev }()

	got, err := FetchRatings(context.Background(), []string{"tt0308808", "tt3268458", "tt9999999"})
	if err != nil {
		t.Fatalf("FetchRatings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2 (third id is intentionally missing)", len(got))
	}
	if got["tt0308808"].AverageRating != 7.2 {
		t.Errorf("tt0308808 rating = %v, want 7.2", got["tt0308808"].AverageRating)
	}
}
