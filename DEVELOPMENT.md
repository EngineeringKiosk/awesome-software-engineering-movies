# Development

This repo is a sibling of
[`awesome-software-engineering-games`](https://github.com/EngineeringKiosk/awesome-software-engineering-games)
and
[`GermanTechPodcasts`](https://github.com/EngineeringKiosk/GermanTechPodcasts).
The shape and tooling are intentionally similar so improvements
cross-pollinate easily.

## Local setup

```sh
cd app
make build              # produces ./awesome-software-engineering-movies
make test               # go test -v -race ./...
make staticcheck        # honnef.co/go/tools/cmd/staticcheck@2026.1
```

## Running the pipeline locally

```sh
cd app
./awesome-software-engineering-movies convertYamlToJson \
    --yaml-directory ../movies \
    --json-directory ../generated

# Needs a YouTube Data API v3 key:
YOUTUBE_API_KEY=… ./awesome-software-engineering-movies collectMovieData \
    --json-directory ../generated

./awesome-software-engineering-movies convertJsonToReadme \
    --json-directory ../generated \
    --readme-template ../assets/README.template \
    --readme-output ../README.md
```

The first step needs no API access. Step two does — get a key from a
Google Cloud project with the YouTube Data API enabled.

`collectMovieData` also pulls IMDb ratings for any entry whose
`imdbID` is set in YAML, but the IMDb dataset is only fetched when
something actually needs refreshing: an entry without IMDb data yet,
or one whose `ratings.imdb.refreshedAt` is older than 30 days. Quiet
runs skip the download entirely. Pass `--force-imdb-refresh` to
override the cache window and refetch every IMDb-tagged entry.

## IMDb dataset

IMDb publishes a daily-refreshed gzipped TSV of all title ratings at
`https://datasets.imdbws.com/title.ratings.tsv.gz`. Documentation:
[IMDb Non-Commercial Datasets](https://developer.imdb.com/non-commercial-datasets/).
The tooling streams that file directly — no API key, no IMDb
Developer API on AWS, no third-party service. The dataset is
licensed for personal and non-commercial use; this curated
open-source list fits that boundary.

## Editing the README

The committed `README.md` is generated. Do not edit it. To change
the layout, edit `assets/README.template` (Go `html/template`
syntax). The render step is exercised on every PR via the
`render-readme` workflow, so template changes get reviewed alongside
the data they touch.

## Extending the YAML schema

Adding a new field is a four-touch change:

1. Update every file under `movies/*.yml` with the new field (or
   make it optional).
2. Add the field to `MovieInformation` in
   `app/cmd/types.go` with the right `yaml` and `json` tags.
3. If the field is YAML-sourced, add it to `mergeMovieInformation`
   in `app/cmd/convertYamlToJson.go` — that function is the
   authoritative list of fields the YAML overrides on every run.
4. Reference it in `assets/README.template` if it should appear in
   the rendered output.

Update this file and `CONTRIBUTING.md` to document the new field.

## Testing

- Go unit tests live next to the code they cover
  (`*_test.go`). `make test` runs the whole suite with the race
  detector.
- YAML files are linted by `yamllint` using `.yamllint.yml`. The
  default ruleset is in effect minus `line-length` and
  `document-start`.
- `golangci-lint v2.12.1` runs in CI with the config at
  `app/.golangci.yml`. `staticcheck` is intentionally disabled there
  and run separately via `make staticcheck`.

## Dependencies

Updated by [Renovate](https://docs.renovatebot.com/) on a monthly
schedule. The `renovate.json` file pins regex managers for the
`staticcheck` version in the Makefile and the `golangci-lint`
version in workflow YAML. Go modules carry a 7-day stability period
to avoid pulling yanked releases.
