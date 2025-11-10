# News Endpoints

- Listing: [`https://api.ennead.cc/stella/news/{category}`](https://api.ennead.cc/stella/news/updates)

Valid `category` values:

- `updates` – mirrors the "Latest" tab from the official site (mixed content).
- `notices` – service notices and dev letters.
- `news` – general announcements.
- `events` – event overviews and activity posts.

`/news/{category}` without the `/stella` prefix is available as a shorter alias.

## Query Parameters

- `index` – upstream page number (defaults to `1`).
- `size` – page size (defaults to `6`, matching the official site).

If the upstream API ignores its `type` filter (which currently happens), the handler post-filters rows locally so each category still returns the right subset. The cache key is the news `id`, so articles fetched through one category are instantly reused by the others.

## GET `/stella/news/{category}`

Returns the upstream payload with enriched thumbnails. Each article's detail page is fetched (with up to four concurrent requests) to capture the first `<img>` inside the body, which replaces the placeholder `thumbnail`. Detail responses are cached for 10 minutes to limit upstream load.

```bash
curl "https://api.ennead.cc/stella/news/notices?index=1&size=6"
```

Example excerpt:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "count": 6,
    "rows": [
      {
        "id": 1982,
        "title": "11/10 Maintenance Notice",
        "link": "https://stellasora.global/news/1982",
        "type": "notice",
        "typeLabel": "Notices",
        "publishTime": 1762771445313,
        "thumbnail": "https://webusstatic.yo-star.com/web-cms-prod/upload/content/2025/11/07/w-07FLkR.jpeg",
        "description": "Dear Tyrant, ..."
      }
    ]
  },
  "timestamp": 1762792569235
}
```

### Errors

- `400`: invalid `index`/`size` values or malformed requests.
- `404`: unknown category.
- `405`: method not allowed.
- `502`: upstream news servers unreachable or returned a non-200 status.
