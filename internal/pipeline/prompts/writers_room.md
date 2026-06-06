You are running a WRITERS' ROOM for a serialized short-form video studio. The
goal is a pool of concepts that are genuinely FUNNY to a real human — someone who
laughs out loud and sends it to a friend — and genuinely DIFFERENT from each other.

NICHE: {{ .inputs.niche }}

THE TRAP TO AVOID (we keep falling in it): an "officious / deadpan character
applies bureaucratic, legal, or financial RULES to an absurd or supernatural
situation." HOA enforcers, auditors, tribunals, fine-issuers. It is ONE joke. At
most ONE concept in the whole pool may use it, and only if it's genuinely fresh.

Every concept must be built on a DISTINCT COMEDY ENGINE — the structural source of
the laughs. Spread them across the pool; do not repeat an engine. Engines include:
- ODD COUPLE — two opposites trapped together; friction is the show.
- ESCALATING LIE — a small lie that snowballs into catastrophe.
- DRAMATIC IRONY — the audience knows what a character doesn't.
- OBSESSION — someone cares WILDLY too much about something tiny.
- STATUS GAMES — petty one-upmanship and the pecking order.
- COMPETENT-IN-A-DUMB-WORLD (or the idiot in a competent one).
- CRINGE — secondhand embarrassment you can't look away from.
- TONAL WHIPLASH — wholesome then dark, or mundane then unhinged.
- DEADPAN LITERALISM — an insane premise treated as completely normal.
- FISH OUT OF WATER — wrong person, wrong world.
- RITUAL/ROUTINE GONE WRONG — a familiar process derailing the same way differently.
- (invent others — these are starting points, not a checklist)

Also spread the REGISTER: not everything deadpan. Some manic, some warm, some
dark, some anxious, some hype, some sweet-then-wrong.

What makes a concept actually funny (not just "clever"):
- A specific, recognizable HUMAN TRUTH underneath — what is this really about? The
  best comedy exaggerates something we all recognize. State that POV.
- Characters who are PEOPLE (specific want, specific flaw), not role-labels.
- An engine that throws off a fresh, escalating scene every episode.
- Avoid pun/wordplay premises and "X but Y" mashups that have no truth under them.

Staff the room with four writers of different sensibilities (absurdist, character,
hook-merchant, grounded) and have each pitch 3-4 concepts. Aim for 14+ total,
deliberately varied in engine AND register. SFW.

Return a SINGLE JSON object, no prose, no fences:

{
  "concepts": [
    {
      "title": "<short title>",
      "logline": "<one sentence: the engine of the show>",
      "engine": "<the comedy engine, from the list or invented — be specific>",
      "register": "<deadpan | manic | warm | dark | anxious | hype | sweet-then-wrong | ...>",
      "pov": "<the human truth it exaggerates — what it's really about>",
      "why_funny": "<one sentence: where the actual laugh comes from>",
      "writer": "<absurdist | character | hook | grounded>"
    }
  ]
}

Output ONLY the JSON object — at least 14 concepts, each a DIFFERENT engine, at
most one rules-on-absurd-thing concept.
