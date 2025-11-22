# Stella Sora API

Stella Sora API is a Go-powered service that exposes game data through a clean JSON interface backed by MongoDB. It provides lightweight list endpoints for quick lookups and dedicated detail routes for heavier payloads.

See it being fully used at [StellaBase](https://stella.ennead.cc)

## Available Routes

Base path is `/stella/`. The status endpoint acts as the index and is not listed in its own payload.

| Route | Description |
| ----- | ----------- |
| `GET /stella/` | Status, uptime (Unix epoch when the server started) and enumerated endpoints. |
| `GET /stella/characters` | Lightweight character list; omits heavy fields but now includes an `icon` path (e.g. `/stella/assets/Amber.png`) for quick asset lookups. |
| `GET /stella/character/{idOrName}` | Full character document (includes stats, skills, upgrades, etc.). |
| `GET /stella/discs` | Disc summaries (id, name, star, element) plus an `icon` path for quick art lookups. |
| `GET /stella/disc/{idOrName}` | Full disc record (tags, skills, stats, upgrades, duplicates) with flattened `icon`, `background`, and `variants` asset paths. |
| `GET /stella/banners` | Banner data grouped into `current`/`permanent`/`upcoming`/`ended`, including rate-up entries, asset paths, and a `permanent` flag for timeless banners. |
| `GET /stella/events` | Event schedule with timing windows and featured rewards. |
| `GET /stella/news/{category}` | Official news proxy; `category` is one of `updates`, `notices`, `news`, or `events`. Supports `index`/`size`, deduplicates upstream rows, and swaps in the hero image from the article body with a 10-minute cache. |
| `GET /stella/assets/{friendlyName}` | Serves on-disk character textures using friendly aliases (e.g. `Amber_portrait.png`). |

Common query parameters:

- `lang`: two-letter region code (e.g. `EN`, `JP`, `KR`, `CN`, `TW`). Defaults to `EN` when omitted.

Friendly asset names are derived from the in-game character name: `Amber.png` resolves to the default icon, `Amber_portrait.png` to the `sk` variant, `Amber_background.png` to the background, and other suffixes (`_q`, `_goods`, `_xl`, etc.) mirror the variant keys returned by the character payloads. Prefix requests with `/stella/assets/`, e.g. `GET /stella/assets/Amber_q.png`.

News endpoints can also be reached without the `/stella` prefix (e.g. `GET /news/updates`). See `docs/news.md` for request samples and notes on caching/thumbnails.

The character detail payload flattens these assets into root-level `icon`, `portrait`, `background`, and `variants` fields whose values are direct `/stella/assets/...` URLs.

Error handling:

- `404` returns a JSON body `{ "error": "..." }`.
- Unsupported methods respond with `405 Method Not Allowed`.

## Project Layout

```
cmd/api/           Main entrypoint for the Go service
config.yaml        Runtime configuration (server + Mongo)
internal/app/      Shared app state, Mongo lifecycle, endpoint registry
internal/config/   YAML loader with defaults
internal/http/     HTTP server, route registration and handlers
```

Feel free to open issues or submit PRs if you encounter inconsistencies between stored data and API responses.
