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

- Freely viewable on YouTube (no paywall).
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
2. Fill in the four required fields:

   ```yaml
   name: "Inside Envoy: The Proxy for the Future"
   link: https://www.youtube.com/watch?v=uaksVVHDhYU
   language:
     - en
   tags:
     - Networking
     - Service Mesh
     - Open Source
   ```

3. Open a pull request. CI will:
   - Lint your YAML (`yaml-lint` workflow).
   - Re-render the README so you can preview the entry
     (`render-readme` workflow).
   - Build and test the Go tooling (`testing` workflow).

After merge, the next monthly run (or a manual dispatch) of
`movie-data` enriches the JSON with title, description, duration,
channel, view count and thumbnail from the YouTube Data API.

## Field reference

| Field      | Type           | Required | Source of truth | Notes |
|------------|----------------|----------|-----------------|-------|
| `name`     | string         | yes      | YAML            | Drives the slug, filename and README anchor. Keep it close to the YouTube title but cleaned up if needed. |
| `link`     | string (URL)   | yes      | YAML            | YouTube URL — `youtube.com/watch?v=…`, `youtu.be/…`, `/embed/…`, `/shorts/…` are all accepted. |
| `language` | list[string]   | yes      | YAML            | ISO 639-1 codes (`en`, `de`, `fr`, …). Multiple entries when audio mixes languages. |
| `tags`     | list[string]   | yes      | YAML            | Subject-matter tags. Be coarse — better to have 3–5 broad tags than 15 narrow ones. |

The remaining JSON fields (`title`, `description`, `duration`,
`publishedAt`, `channel`, `viewCount`, `image`, `slug`, `videoID`)
are produced by the tooling. **Do not edit `README.md` directly** —
it is overwritten on every CI run. To change rendering, update
`assets/README.template`.
