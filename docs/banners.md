# Banners Endpoint

- Listing: [`https://api.ennead.cc/stella/banners`](https://api.ennead.cc/stella/banners)

Add `?lang=JP` or similar to change localisation (defaults to `EN`).

## GET `/stella/banners`

Returns banner metadata grouped by lifecycle and rate-up pools.

```bash
curl https://api.ennead.cc/stella/banners?lang=EN
```

Example excerpt:

```json
{
  "current": [
    {
      "id": 10119,
      "name": "A Fateful Encounter",
      "bannerType": "Limited Trekker Banner",
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
    }
  ],
  "upcoming": [
    {
      "id": 20155,
      "name": "Tide to the Full Moon",
      "bannerType": "Limited Disc Banner",
      "startTime": "2025-10-28T12:00:00+08:00",
      "endTime": "2025-11-18T03:59:59+08:00",
      "rateUp": {
        "fiveStar": {
          "packageId": 2015511,
          "entries": [
            {
              "id": 214541,
              "name": "Celestial Sonata"
            }
          ]
        },
        "fourStar": {
          "packageId": 2015512,
          "entries": [
            {
              "id": 213214,
              "name": "Aurora Cascade"
            }
          ]
        }
      }
    }
  ],
  "ended": []
}
```

## Errors

- `404`: `{ "error": "no banner data found" }`
- `405`: `method not allowed`
- `503`: MongoDB unavailable
