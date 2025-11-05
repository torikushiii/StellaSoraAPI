# Characters Endpoint

- Summary list: [`https://api.ennead.cc/stella/characters`](https://api.ennead.cc/stella/characters)
- Detail view: [`https://api.ennead.cc/stella/character/Amber`](https://api.ennead.cc/stella/character/Amber)

Append `?lang=JP` (for example) to request another localisation.

## GET `/stella/characters`

Returns a trimmed set of fields suitable for directory views.

```bash
curl https://api.ennead.cc/stella/characters?lang=EN
```

Example:

```json
[
  {
    "id": 103,
    "name": "Amber",
    "description": "The agile dual pistols allow Amber to sashay across the battlefield. Enemies who try to get close to her will turn into ashes long before they can reach her perimeter.",
    "grade": 4,
    "element": "Ignis",
    "position": "Vanguard",
    "attackType": "ranged",
    "style": "Collector",
    "faction": "New Star Guild",
    "tags": [
      "Vanguard",
      "Collector",
      "New Star Guild"
    ]
  },
  {
    "id": 107,
    "name": "Tilia",
    "description": "Tilia will always be charging at the forefront, using her shield to protect everyone. Guess only another Imperial Knight might break through her defense.",
    "grade": 4,
    "element": "Lux",
    "position": "Support",
    "attackType": "melee",
    "style": "Steady",
    "faction": "Imperial Guard",
    "tags": [
      "Support",
      "Steady",
      "Imperial Guard"
    ]
  },
  {
    "id": 108,
    "name": "Kasimira",
    "description": "Kasimira wields a brutal shotgun. She often puts herself in the line of fire in order to get a hit on her enemies. No matter what, her enemy will fall first.",
    "grade": 4,
    "element": "Ignis",
    "position": "Versatile",
    "attackType": "ranged",
    "style": "Adventurous",
    "faction": "White Cat Troupe",
    "tags": [
      "Versatile",
      "Adventurous",
      "White Cat Troupe"
    ]
  }
]
```

## GET `/stella/character/{idOrName}`

Accepts either a numeric ID or case-insensitive name. Returns the complete character payload.

```bash
curl https://api.ennead.cc/stella/character/103?lang=EN
```

Example (long sections collapsed with `{...}`):

```json
{
  "id": 103,
  "name": "Amber",
  "description": "The agile dual pistols allow Amber to sashay across the battlefield. Enemies who try to get close to her will turn into ashes long before they can reach her perimeter.",
  "grade": 4,
  "element": "Ignis",
  "position": "Vanguard",
  "attackType": "ranged",
  "style": "Collector",
  "faction": "New Star Guild",
  "tags": [
    "Vanguard",
    "Collector",
    "New Star Guild"
  ],
  "dateEvents": [
    {
      "name": "The Kitten Encounter",
      "clue": "Visit the Port to unlock",
      "secondChoice": "How about doing some night fishing?"
    },
    {
      "name": "Cafeteria Trick",
      "clue": "Visit the Academy to unlock",
      "secondChoice": "Getting hungry. Gonna go to the cafeteria."
    }
  ],
  "giftPreferences": {
    "loves": [
      "Card Photo Capturer",
      "Reflective Photo Capturer",
      "Ultra-Precision Photo Capturer",
      "Rising Star",
      "Emerging Talent",
      "Shining Star"
    ],
    "hates": []
  },
  "normalAttack": {
    "name": "Duet",
    "description": "Fires both pistols. Each shot deals \\u003Ccolor=#fb8037\\u003E&Param1& of ATK\\u003C/color\\u003E as Ignis DMG. Magazine holds 12 ammo.",
    "shortDescription": "Fires both pistols at the target rapidly.",
    "params": [
      "9.9%/10.9%/11.8%/14.8%/15.8%/16.8%/18.8%/19.7%/20.7%/22.7%/23.7%/24.7%/25.7%"
    ]
  },
  "skill": {
    "name": "Fireworks Jam",
    "description": "Fires both pistols in a sweeping motion, dealing \\u003Ccolor=#fb8037\\u003E&Param1& of ATK x2\\u003C/color\\u003E and \\u003Ccolor=#fb8037\\u003E&Param2& of ATK x4\\u003C/color\\u003E as AoE Ignis DMG, and increasing Auto Attack DMG by &Param4& for &Param5&s.\\u000BFireworks Jam (Main Skill) can trigger ##Ignis Mark: Sacred Flame#2013#, dealing &Param3& of ATK as AoE Ignis Mark DMG.",
    "shortDescription": "Fires both pistols in a sweeping motion three times.\\u000BThe Main Skill can trigger ##Ignis Mark: Sacred Flame#2013#.",
    "params": [
      "46%/51%/55%/69%/73%/78%/87%/92%/96%/106%/110%/115%/119%",
      "3.3%/3.6%/3.9%/4.9%/5.3%/5.6%/6.2%/6.6%/6.9%/7.6%/7.9%/8.2%/8.6%",
      "54%/65%/76%/87%/98%/109%/120%/131%/142%",
      "25%",
      "10"
    ],
    "cooldown": "8s"
  },
  "supportSkill": { "...": "..." },
  "ultimate": { "...": "..." },
  "potentials": { "...": "..." },
  "talents": { "...": "..." },
  "stats": { "...": "..." },
  "upgrades": { "...": "..." },
  "skillUpgrades": { "...": "..." }
}
```

## Errors

- `404`: `{ "error": "character not found" }`
- `405`: `method not allowed`
- `503`: MongoDB unavailable
