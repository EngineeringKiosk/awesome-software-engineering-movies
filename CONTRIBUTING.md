# Contributing

Thanks for considering an addition. The list is curated, so the bar
is "would a software engineer find this worth their evening?"

## Acceptance criteria

An entry is a good fit if it meets **at least one** of the following:

- A documentary or longer-form film about a programming language,
  framework, library, tool or open-source project.
- A documentary or talk about software engineering culture, history,
  community or notable people in the industry.
- A multi-part series or talk that goes substantially deeper than a
  typical conference session.
- Educational content broadly relevant to software engineers (e.g.
  the inner workings of the internet, distributed systems, security,
  hardware).

In addition, the entry should be:

- Freely viewable.
- At least 15 minutes long.
- Published more than 2 weeks ago (gives the community time to react
  to clickbait/low-quality posts).

Out of scope for now:

- Short tutorials or "build X in 10 minutes" videos.
- Conference talks unless they are explicitly long-form / documentary
  in nature.
- Content not in a language any reasonable engineer can follow
  (use the `language` field to be explicit).

## How to add an entry

1. Create a new file under `movies/` named after a kebab-case slug of
   the title. Both `.yml` and `.yaml` extensions are accepted.
2. Fill in the required fields:

   ```yaml
   name: "Inside Envoy: The Proxy for the Future"
   links:
     youtube: https://www.youtube.com/watch?v=uaksVVHDhYU
   tags:
     - Networking
     - Service Mesh
     - Open Source
   ```

   `links` is a map keyed by platform slug — list every platform
   the entry is available on (`youtube`, `netflix`, `amazon_prime_video`,
   `bpb`).

   `language` is optional — see the field reference below.

3. Open a pull request. CI will:
   - Lint your YAML (`yaml-lint` workflow).
   - Re-render the README so you can preview the entry
     (`render-readme` workflow).
   - Build and test the Go tooling (`testing` workflow).

After merge, the next monthly run (or a manual dispatch) of
`movie-data` enriches the JSON with title, description, duration,
channel, views, like count and thumbnail from the YouTube Data API.
For entries that also set an `imdbID`, the same run pulls the IMDb
rating (when missing or older than 30 days) from the public IMDb
non-commercial dataset.

## Field reference

| Field      | Type           | Required | Source of truth | Notes |
|------------|----------------|----------|-----------------|-------|
| `name`     | string         | yes      | YAML            | Drives the slug, filename and README anchor. Keep it close to the YouTube title but cleaned up if needed. |
| `links`    | map[slug → URL] | yes     | YAML            | Map of every platform the entry is available on, keyed by platform slug (`youtube`, `netflix`, `amazon_prime_video`, `bpb`). At least one entry required. The tooling validates known-slug URLs against the slug they claim to be and warns on mismatch. Unknown slugs are accepted (a maintainer may pre-declare a platform before its detector lands). |
| `language`    | list[string]   | no       | YAML > API      | ISO 639-1 codes (`en`, `de`, `fr`, …). If omitted, the tooling falls back to the YouTube `defaultAudioLanguage` and stores it as a single-element list. Set it manually when the API returns nothing or when the video has multiple audio languages. |
| `description` | string         | no       | YAML > API      | Free text. If omitted, the tooling uses the video's YouTube description. Set it manually when the YouTube description is empty, full of unrelated boilerplate, or otherwise unhelpful for skim-reading the README. |
| `tags`        | list[string]   | yes      | YAML            | Subject-matter tags. Be coarse — better to have 3–5 broad tags than 15 narrow ones. |
| `imdbID`      | string         | no       | YAML            | IMDb tconst (e.g. `tt3268458`). Set this only when the entry is also catalogued on IMDb so the tooling can pull the IMDb rating from the public dataset. Most YouTube documentaries are not on IMDb — leave this unset for those. |
| `localized`   | map[code → object] | no   | YAML            | Per-language alternate-version overrides. Keys are ISO 639-1 codes (`de`, `es`, …). Each value supports optional `title`, `description`, and `links` (same shape as the top-level `links` map) — provide whichever differs from the English top-level. The `links` override is per-key: only the platform slugs you list override their top-level counterparts; every other top-level platform is inherited unchanged. Alternate links are not enriched (no extra YouTube/IMDb API calls); they round-trip from YAML to JSON unchanged. |
| `youtubeTrailerForThumbnail` | string (YouTube URL) | no | YAML | Fallback YouTube URL the tooling uses for the poster image when the entry's `links` map has no `youtube` key (or the primary YouTube thumbnail download fails). Set this for Netflix / Amazon Prime / bpb entries that have a YouTube trailer so the README still gets a poster. If neither the primary YouTube link nor this trailer yields an image, the tooling falls back to a bundled placeholder. |
| `title`       | string         | no       | YAML > API      | Optional override of the entry's title. If omitted, the tooling uses the YouTube API's `snippet.title`. Set explicitly for non-YouTube entries (Netflix, bpb, …) where there is no API title, or to override an unhelpful upload title. Note that the README's heading is driven by `name`; `title` is for the JSON and downstream consumers. |
| `duration`    | string (ISO-8601) | no    | YAML > API      | Optional override of the entry's runtime, format `PT[xH][yM][zS]` (e.g. `PT1H54M`). API-supplied for YouTube entries; set manually for non-YouTube entries so the README can render `Duration: ca. X min.` |
| `publishedAt` | string (RFC3339)  | no    | YAML > API      | Optional override of the entry's release / upload date (e.g. `2019-07-24T00:00:00Z`). API-supplied for YouTube entries; set manually for non-YouTube entries to record the release date. |

The remaining JSON fields (`channel`, `ratings`, `views`, `image`,
`slug`) are produced by the tooling.

If your entry has an alternate-language version, add a `localized`
block — for example:

```yaml
name: "Lo and Behold: Reveries of the Connected World"
links:
  youtube: https://www.youtube.com/watch?v=q3g3hqNJqpQ
tags: [Internet, History, Society]
localized:
  de:
    title: Wovon träumt das Internet?
    description: Die Dokumentation beleuchtet die Evolution des Internets ...
    links:
      amazon_prime_video: https://www.amazon.de/gp/video/detail/B0FVCKCM81/
```

The German version above adds a different platform
(`amazon_prime_video`) without touching the English top-level
`youtube` link — that's the per-key merge: the German viewer sees
both the English YouTube upload and the German Amazon Prime
upload.

**Do not edit `README.md` directly** —
it is overwritten on every CI run. To change rendering, update
`assets/README.template`.
