# Banners Endpoint

- Listing: [`https://api.ennead.cc/stella/banners`](https://api.ennead.cc/stella/banners)

Add `?lang=JP` or similar to change localisation (defaults to `EN`).

## GET `/stella/banners`

Returns banner metadata grouped into `current`, `permanent`, `upcoming`, and `ended` pools. Each banner exposes an `assets` block (tab icon, banner, cover) with `/stella/assets/...` URLs. Permanent banners (no `startTime`/`endTime`) appear in the `permanent` array and include `permanent: true`.

```bash
curl https://api.ennead.cc/stella/banners?lang=EN
```

Example excerpt (empty lifecycle buckets are returned as empty arrays):

```json
{
  "current": [
    {
      "id": 10119,
      "name": "A Fateful Encounter",
      "bannerType": "Limited Trekker Banner",
      "startTime": "2025-11-11T12:00:00+08:00",
      "endTime": "2025-12-02T03:59:59+08:00",
      "assets": {
        "tabIcon": "/stella/assets/tab_gacha_10119.png",
        "banner": "/stella/assets/banner_gacha_10119.png",
        "cover": "/stella/assets/Cover_10119.png"
      },
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
  "permanent": [],
  "upcoming": [
    {
      "id": 20155,
      "name": "Tide to the Full Moon",
      "bannerType": "Limited Disc Banner",
      "startTime": "2025-10-28T12:00:00+08:00",
      "endTime": "2025-11-18T03:59:59+08:00",
      "assets": {
        "tabIcon": "/stella/assets/tab_gacha_20155.png",
        "banner": null,
        "cover": "/stella/assets/Cover_20155.png"
      },
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

Permanent banners are returned in the `permanent` array and omit both timestamps:

```json
"permanent": [
  {
    "id": 1,
    "name": "Boss's Regulars",
    "bannerType": "Permanent Trekker Banner",
    "permanent": true,
    "assets": {
      "tabIcon": "/stella/assets/tab_gacha_1.png",
      "banner": null,
      "cover": "/stella/assets/Cover_1.png"
    },
    "rateUp": {
      "fiveStar": null,
      "fourStar": null
    }
  }
]
```

## Errors

- `404`: `{ "error": "no banner data found" }`
- `405`: `method not allowed`
- `503`: MongoDB unavailable
