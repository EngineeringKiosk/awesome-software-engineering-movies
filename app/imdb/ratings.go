// Package imdb fetches title ratings from IMDb's public non-commercial
// dataset. The dataset is a single gzipped TSV file refreshed daily
// at https://datasets.imdbws.com/title.ratings.tsv.gz; using it
// avoids IMDb's gated AWS-hosted Developer API (and its associated
// price tag) while still giving us authoritative ratings.
//
// The dataset is licensed for personal and non-commercial use.
package imdb

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DatasetURL points to the TSV file we stream. Exposed as a var (not
// const) so tests can override it with a local fixture server.
var DatasetURL = "https://datasets.imdbws.com/title.ratings.tsv.gz"

// userAgent matches the value used elsewhere in the project so a
// single string appears in IMDb's request logs.
const userAgent = "EngineeringKiosk-awesome-software-engineering-movies"

// Rating is the subset of the ratings dataset we persist per movie.
// The TSV file's third column (numVotes) is signed in IMDb's schema
// only because they reuse the same loader for several files;
// in practice it is always non-negative.
type Rating struct {
	AverageRating float64
	NumVotes      int64
}

// FetchRatings downloads title.ratings.tsv.gz and returns ratings for
// the requested IMDb tconsts. Tconsts not present in the dataset
// (typo in YAML, withdrawn title, freshly-added title not yet in the
// daily dump) are simply absent from the returned map; the caller is
// expected to diff requested-vs-returned and handle the difference.
//
// Passing an empty slice is a no-op — no HTTP call is made.
func FetchRatings(ctx context.Context, ids []string) (map[string]Rating, error) {
	if len(ids) == 0 {
		return map[string]Rating{}, nil
	}

	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			want[id] = struct{}{}
		}
	}
	if len(want) == 0 {
		return map[string]Rating{}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DatasetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("imdb: build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Encoding", "gzip")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("imdb: GET %s: %w", DatasetURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("imdb: GET %s: status %s", DatasetURL, resp.Status)
	}

	return parseRatingsTSV(resp.Body, want)
}

// parseRatingsTSV streams the gzipped TSV and returns rows whose
// tconst is in want. Splitting the parse out of FetchRatings keeps
// the network and decoding concerns separable, which is what the
// tests exploit.
func parseRatingsTSV(r io.Reader, want map[string]struct{}) (map[string]Rating, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("imdb: gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	out := make(map[string]Rating, len(want))
	// IMDb's ratings file has ~1.5M rows × ~30 bytes — well within
	// bufio.Scanner's default token cap, but we raise the buffer
	// once anyway to insure against pathological future rows.
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	first := true
	for sc.Scan() {
		line := sc.Text()
		if first {
			// Header row: "tconst\taverageRating\tnumVotes". Skip it
			// without sanity-checking — IMDb has not changed this
			// schema in the lifetime of the dataset, and a hard fail
			// on column-name mismatch would just be brittle.
			first = false
			continue
		}

		tconst, rating, votes, ok := splitRatingRow(line)
		if !ok {
			continue
		}
		if _, wanted := want[tconst]; !wanted {
			continue
		}
		out[tconst] = Rating{AverageRating: rating, NumVotes: votes}

		if len(out) == len(want) {
			// Found everything; no point streaming the rest of the
			// dataset. Significant win: the file is mostly rows we
			// don't care about.
			break
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("imdb: scan: %w", err)
	}
	return out, nil
}

// splitRatingRow parses a single TSV row. Returns ok=false for rows
// that don't match the documented three-column shape, so the caller
// can skip them. We don't error on bad rows — the file is large and
// occasionally has odd entries; one weird line shouldn't fail the
// whole pipeline.
func splitRatingRow(line string) (tconst string, rating float64, votes int64, ok bool) {
	// Manual split is cheaper than strings.Split for a known column
	// count and avoids allocating a 3-element slice per row × ~1.5M
	// rows.
	t1 := strings.IndexByte(line, '\t')
	if t1 <= 0 {
		return "", 0, 0, false
	}
	t2 := strings.IndexByte(line[t1+1:], '\t')
	if t2 <= 0 {
		return "", 0, 0, false
	}
	t2 += t1 + 1

	tconst = line[:t1]
	ratingStr := line[t1+1 : t2]
	votesStr := line[t2+1:]

	r, err := strconv.ParseFloat(ratingStr, 64)
	if err != nil {
		return "", 0, 0, false
	}
	v, err := strconv.ParseInt(votesStr, 10, 64)
	if err != nil {
		return "", 0, 0, false
	}
	return tconst, r, v, true
}
