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
   link: https://www.youtube.com/watch?v=uaksVVHDhYU
   tags:
     - Networking
     - Service Mesh
     - Open Source
   ```

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
| `link`     | string (URL)   | yes      | YAML            | YouTube URL — `youtube.com/watch?v=…`, `youtu.be/…`, `/embed/…`, `/shorts/…` are all accepted. |
| `language`    | list[string]   | no       | YAML > API      | ISO 639-1 codes (`en`, `de`, `fr`, …). If omitted, the tooling falls back to the YouTube `defaultAudioLanguage` and stores it as a single-element list. Set it manually when the API returns nothing or when the video has multiple audio languages. |
| `description` | string         | no       | YAML > API      | Free text. If omitted, the tooling uses the video's YouTube description. Set it manually when the YouTube description is empty, full of unrelated boilerplate, or otherwise unhelpful for skim-reading the README. |
| `tags`        | list[string]   | yes      | YAML            | Subject-matter tags. Be coarse — better to have 3–5 broad tags than 15 narrow ones. |
| `imdbID`      | string         | no       | YAML            | IMDb tconst (e.g. `tt3268458`). Set this only when the entry is also catalogued on IMDb so the tooling can pull the IMDb rating from the public dataset. Most YouTube documentaries are not on IMDb — leave this unset for those. |
| `platform`    | string         | no       | YAML > tooling  | Slug of the source the link lives on (`youtube` today). Auto-detected from `link` when omitted; set explicitly only if you need to override the detector or your link is from a source the tooling does not yet recognise. The tooling logs a warning when an explicit platform disagrees with what the link looks like. |
| `localized`   | map[code → object] | no   | YAML            | Per-language alternate-version overrides. Keys are ISO 639-1 codes (`de`, `es`, …). Each value supports optional `title` and `link` — provide whichever differs from the English `name`/`link` above; the English version is always the rendered default. Alternate links are not enriched (no extra YouTube/IMDb API calls); they round-trip from YAML to JSON unchanged. |

The remaining JSON fields (`title`, `duration`, `publishedAt`,
`channel`, `ratings`, `views`, `image`, `slug`, `videoID`,
`platform` when not set in YAML) are produced by the tooling.

If your entry has an alternate-language version, add a `localized`
block — for example:

```yaml
name: "Python: The Documentary"
link: https://www.youtube.com/watch?v=GfH4QL4VqJ0
tags: [Programming Languages, Open Source, History]
localized:
  de:
    title: Pythons Geschichte
    link: https://www.youtube.com/watch?v=...
  es:
    title: La historia de Python
```

**Do not edit `README.md` directly** —
it is overwritten on every CI run. To change rendering, update
`assets/README.template`.
