# Discs Endpoint

- Summary list: [`https://api.ennead.cc/stella/discs`](https://api.ennead.cc/stella/discs)
- Detail view: [`https://api.ennead.cc/stella/disc/Crisp%20Morning`](https://api.ennead.cc/stella/disc/Crisp%20Morning)

Use the `lang` query parameter to switch localisation (defaults to `EN`).

## GET `/stella/discs`

Returns concise disc information suitable for listings.

```bash
curl https://api.ennead.cc/stella/discs?lang=EN
```

Example:

```json
[
  {
    "id": 211001,
    "name": "Crisp Morning",
    "star": 3,
    "element": "None"
  },
  {
    "id": 211002,
    "name": "Sunny Breeze",
    "star": 3,
    "element": "None"
  },
  {
    "id": 211003,
    "name": "Indigo Sunset",
    "star": 3,
    "element": "None"
  }
]
```

## GET `/stella/disc/{idOrName}`

Returns full disc details. Accepts a numeric ID or case-insensitive name.

```bash
curl https://api.ennead.cc/stella/disc/211001?lang=EN
```

Example (only representative sections shown):

```json
{
  "id": 211001,
  "name": "Crisp Morning",
  "star": 3,
  "element": "None",
  "tag": [
    "Verse"
  ],
  "mainSkill": {
    "name": "Fortissimo: Main Theme",
    "description": "Increases the main Trekker's ATK by \\u003Ccolor=#ec6d21\\u003E{1}\\u003C/color\\u003E.",
    "params": [
      [
        "7%"
      ],
      [
        "8.4%"
      ],
      [
        "9.8%"
      ],
      [
        "11.2%"
      ],
      [
        "12.6%"
      ],
      [
        "14%"
      ]
    ]
  },
  "secondarySkills": [],
  "supportNote": [
    [
      {
        "name": "Melody of Pummel",
        "quantity": 3
      }
    ]
  ],
  "stats": [
    [
      "..."
    ]
  ],
  "dupe": [
    [
      {
        "id": "atk",
        "label": "ATK",
        "value": 46
      }
    ]
  ],
  "upgrades": [
    {
      "items": [
        {
          "id": 21091,
          "name": "Faint Light Breath",
          "quantity": 1
        }
      ],
      "currency": {
        "dorra": 2700
      }
    }
  ]
}
```

## Errors

- `404`: `{ "error": "disc not found" }`
- `405`: `method not allowed`
- `503`: MongoDB unavailable
