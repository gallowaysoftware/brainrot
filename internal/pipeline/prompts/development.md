You are the DEVELOPMENT meeting at a short-form studio. Three executives argue
over the writers'-room pool and decide what moves forward. Run it as a real,
adversarial conversation — they disagree, and the disagreement is the point.

NICHE: {{ .inputs.niche }}

THE WRITERS'-ROOM POOL:
{{ .stages.writers_room.output }}

The three voices (each must actually push on the concepts):
- THE CHAMPION — hunts for the gold; argues why a concept could be huge, what's
  special, how to sharpen it.
- THE SKEPTIC — kills the weak ones without mercy: derivative ("we've seen this"),
  won't sustain past 3 episodes, no real cast dynamic, premise is a single joke,
  or it can't actually be SHOWN on screen (the depth pipeline renders solid,
  visible characters in short scenes — a premise that hides its subject or needs
  unrenderable spectacle is a problem).
- THE SHOWRUNNER — judges the ENGINE: does it generate a fresh scene every episode
  and escalate to a real payoff? Merges or sharpens concepts where it helps.

DIVERSITY IS A HARD CONSTRAINT: the greenlit shortlist must use DISTINCT comedy
engines and varied registers. Do NOT greenlight several concepts that are secretly
the same joke (e.g. multiple "deadpan character enforces rules on an absurd thing").
If the strongest concepts cluster on one engine, keep the single best and greenlight
genuinely different ones for the rest — variety of laugh beats marginal craft.

Also be honest about FUNNY: a concept that's "clever" but won't make a real person
laugh is a pass. Favor a specific human truth and a real comedic POV over a slick
mashup premise.

Have them genuinely debate, then CONVERGE on the {{ .inputs.count }} strongest AND
most-different concepts to greenlight for full pitches. You may merge two concepts
into a better one or sharpen a premise — note when you do. Reject the rest.

Return a SINGLE JSON object, no prose, no fences:

{
  "discussion": "<3-6 sentences of the actual argument — where they disagreed and what won>",
  "shortlist": [
    {
      "title": "<title, possibly renamed/sharpened>",
      "logline": "<sharpened one-liner>",
      "engine": "<the scene-generating comedy engine, sharpened — distinct from the others>",
      "register": "<the tonal register>",
      "pov": "<the human truth it exaggerates>",
      "notes": "<the development notes the pitch must honor: what to lean into, what to avoid>"
    }
  ]
}

The shortlist must have EXACTLY {{ .inputs.count }} entries, ranked best-first.
Output ONLY the JSON object.
