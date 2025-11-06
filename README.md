# Stella Sora API

Stella Sora API is a Go-powered service that exposes game data through a clean JSON interface backed by MongoDB. It provides lightweight list endpoints for quick lookups and dedicated detail routes for heavier payloads.

## Available Routes

Base path is `/stella/`. The status endpoint acts as the index and is not listed in its own payload.

| Route | Description |
| ----- | ----------- |
| `GET /stella/` | Status, uptime (Unix epoch when the server started) and enumerated endpoints. |
| `GET /stella/characters` | Lightweight character list; omits stats, skills, talents, upgrades, date events and gift preferences. |
| `GET /stella/character/{idOrName}` | Full character document (includes stats, skills, upgrades, etc.). |
| `GET /stella/discs` | Disc summaries (id, name, star, element). |
| `GET /stella/disc/{idOrName}` | Full disc record (tags, skills, stats, upgrades, duplicates). |
| `GET /stella/banners` | Banner data with rate-up entries. |
| `GET /stella/events` | Event schedule with timing windows and featured rewards. |

Common query parameters:

- `lang`: two-letter region code (e.g. `EN`, `JP`, `KR`, `CN`, `TW`). Defaults to `EN` when omitted.

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
