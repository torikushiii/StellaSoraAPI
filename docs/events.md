# Events Endpoint

- Listing: [`https://api.ennead.cc/stella/events`](https://api.ennead.cc/stella/events)

Add `?lang=JP` or similar to change localisation (defaults to `EN`).

## GET `/stella/events`

Returns current and upcoming event windows alongside their featured rewards.

```bash
curl https://api.ennead.cc/stella/events?lang=EN
```

Example excerpt:

```json
[
  {
    "id": 10100,
    "title": "Daring Adventure! The Ghost Ship Haunts the Deep",
    "description": "[Event Notes]\nThe event: Daring Adventure! The Ghost Ship Haunts the Deep has begun!\n\nDuring the event, you can challenge limited-time stages, read event stories, and play the mini-game Lucky Treasure Shovel.\n\nCompleting these activities and event missions will reward you with event currency: Squid Rice Crackers, which can be used to redeem various rewards.",
    "startTime": "2025-10-28T12:00:00+08:00",
    "endTime": "2025-11-11T04:00:00+08:00",
    "claimEndTime": "2025-11-18T04:00:00+08:00",
    "rewards": [
      {
        "id": 116001,
        "name": "Daring Adventure! The Ghost Ship Haunts the Deep",
        "category": "honor"
      },
      {
        "id": 214039,
        "name": "Summer Sanctuary",
        "category": "item"
      },
      {
        "id": 502,
        "name": "Cerulean Ticket",
        "category": "item"
      }
    ]
  }
]
```

## Errors

- `404`: `{ "error": "no event data found" }`
- `405`: `method not allowed`
- `503`: MongoDB unavailable
