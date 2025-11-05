# Banners Endpoint

- Listing: [`https://api.ennead.cc/stella/banners`](https://api.ennead.cc/stella/banners)

Add `?lang=JP` or similar to change localisation (defaults to `EN`).

## GET `/stella/banners`

Returns banner metadata and rate-up pools.

```bash
curl https://api.ennead.cc/stella/banners?lang=EN
```

Example excerpt:

```json
[
  {
    "id": 10119,
    "name": "A Fateful Encounter",
    "startTime": "2025-11-11T12:00:00+08:00",
    "endTime": "2025-12-02T03:59:59+08:00",
    "rateUp": {
      "fiveStar": {
        "packageId": 1011911,
        "entries": [
          {
            "id": 119,
            "name": "Nanoha"
          }
        ]
      },
      "fourStar": {
        "packageId": 1011912,
        "entries": [
          {
            "id": 107,
            "name": "Tilia"
          },
          {
            "id": 108,
            "name": "Kasimira"
          }
        ]
      }
    }
  },
  {
    "id": 20119,
    "name": "A Heart-Tuned Melody",
    "startTime": "2025-11-11T12:00:00+08:00",
    "endTime": "2025-12-02T03:59:59+08:00",
    "rateUp": {
      "fiveStar": {
        "packageId": 2011911,
        "entries": [
          {
            "id": 214028,
            "name": "Daylight Garden"
          }
        ]
      },
      "fourStar": {
        "packageId": 2011912,
        "entries": [
          {
            "id": 213008,
            "name": "Tranquil Retreat"
          },
          {
            "id": 213005,
            "name": "★Bam Bam Girl★"
          }
        ]
      }
    }
  }
]
```

## Errors

- `404`: `{ "error": "no banner data found" }`
- `405`: `method not allowed`
- `503`: MongoDB unavailable
