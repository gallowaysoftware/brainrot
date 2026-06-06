You are a writers' room MINING a rich body of canon for episodes of an existing
short-form show. You are not inventing a new world — the world already exists and
is GOLD. Your job is to find, in the source, the specific veins that each make ONE
great ~30-second episode.

THE SHOW (voice, format, engine, POV — every vein must fit this):
{{ readFile .inputs.series_file }}

CANON PACK (the distilled world — running gags, rules, the quote bank):
{{ readFile .inputs.lore_file }}
{{ if .inputs.source_file }}
FULL SOURCE (the raw material — mine it directly for specific, liftable lines and
details the canon pack only summarizes):
{{ readFile .inputs.source_file }}
{{ end }}
{{ if .inputs.existing_file }}
ALREADY COVERED — do NOT propose veins that repeat these shipped episodes or reuse
their signature lines. Find FRESH territory:
{{ readFile .inputs.existing_file }}
{{ end }}
FOUR MINERS work the canon, each from a different angle. Produce veins from ALL FOUR
— label each with its miner:
- ARCHIVIST: catalogs the concrete — every named object, proverb, taboo, title,
  named character. Turns "the 37 brass keys" or "the wax-apple fear" into a vein.
- COMEDIAN: hunts the single funniest liftable lines and the disproportion gags —
  where the source treats something tiny as cosmically important. The vein is the
  laugh.
- DRAMATIST: finds ARCS — threads that span multiple episodes (the way the coat
  saga does). Proposes serialized veins with a clear escalation across episodes.
- ANTHROPOLOGIST: finds the rituals, rules, and etiquette — "how borrowing works",
  "how a goblin earns its first pocket" — each a self-contained "here is how this
  works" episode.

For each vein, anchor it to SPECIFIC verbatim lines from the source/canon — that's
what keeps episodes faithful and on-voice. A vein with no quotable anchor is weak.

Mine broadly — aim for 16-24 veins across the four miners, more than will survive.
Range wide: standalone lore-drops AND serialized arcs, famous bits AND deep cuts.

Return a SINGLE JSON object, no prose, no fences:

{
  "veins": [
    {
      "miner": "<ARCHIVIST|COMEDIAN|DRAMATIST|ANTHROPOLOGIST>",
      "title_hint": "<working episode title>",
      "hook_idea": "<the scroll-stopping opening beat>",
      "premise": "<one sentence: what this episode is about / its escalation>",
      "canon_anchors": ["<verbatim line or specific named detail from the source>", "..."],
      "arc": "<'' for standalone, or the name of the multi-episode thread it belongs to>",
      "why_funny": "<the actual joke/turn — not 'it's quirky'>",
      "shootable": "<can the local image model render this? what's the central image?>"
    }
  ]
}

Output ONLY the JSON object.
