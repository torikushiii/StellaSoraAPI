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
    ],
    "shops": [
      {
        "id": 1001,
        "name": "Shop 1",
        "currency": {
          "itemId": 73101,
          "name": "Squid Rice Cracker",
          "description": "Currency earned via active participation in the event: Daring Adventure! The Ghost Ship Haunts the Deep. Can be used to redeem items in the Event Shop.",
          "flavor": "How many of these squid rice crackers she made can you eat in one go?"
        },
        "goods": [
          {
            "id": 100101,
            "order": 1,
            "name": "Summer Sanctuary",
            "description": "A Runic Disc titled [Summer Sanctuary]. It can unleash great power.",
            "itemId": 214039,
            "quantity": 1,
            "price": 8000,
            "currency": {
              "itemId": 73101,
              "name": "Squid Rice Cracker",
              "description": "Currency earned via active participation in the event: Daring Adventure! The Ghost Ship Haunts the Deep. Can be used to redeem items in the Event Shop.",
              "flavor": "How many of these squid rice crackers she made can you eat in one go?"
            },
            "limit": 1
          },
          {
            "id": 100104,
            "order": 4,
            "name": "Cerulean Ticket",
            "description": "A certificate used for recruiting in Limited Trekker Banner.",
            "itemId": 502,
            "quantity": 1,
            "price": 500,
            "currency": {
              "itemId": 73101,
              "name": "Squid Rice Cracker",
              "description": "Currency earned via active participation in the event: Daring Adventure! The Ghost Ship Haunts the Deep. Can be used to redeem items in the Event Shop.",
              "flavor": "How many of these squid rice crackers she made can you eat in one go?"
            },
            "limit": 5
          }
        ]
      }
    ]
  }
]
```

## Errors

- `404`: `{ "error": "no event data found" }`
- `405`: `method not allowed`
- `503`: MongoDB unavailable
