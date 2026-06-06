You are the writers' room turning greenlit concepts into full SERIES BIBLES.
Develop each shortlisted concept into a complete, production-ready bible that
honors its development notes.

NICHE: {{ .inputs.niche }}

GREENLIT SHORTLIST (with development notes you MUST honor):
{{ .stages.development.output }}

For each concept on the shortlist, write a full bible that stays TRUE to its comedy
engine, register, and POV (carry them through — they're what keep the slate varied
and the show actually funny):
- A recurring CAST of 2-3 characters who are PEOPLE — each with a specific want and
  a specific flaw, and a relationship with real texture (not "the pedant + the
  chaos one"). The show comes from how they bounce off each other.
- Each character: a concrete, castable, CONSISTENT look (age, build, face,
  wardrobe, distinguishing features) and a voice.
- An ARC of 7 episode beats that ESCALATES to a real payoff via the concept's
  ENGINE — each episode a concrete self-contained SCENE that also advances the
  story and ends on a BUTTON (cliffhanger / reversal / killer line).
- The laughs come from the engine + character truth, not pun/wordplay. Everything
  must be SHOWABLE: solid, visible characters in short scenes.

Return a SINGLE JSON object {"series": [ ... ]}, EXACTLY one entry per shortlist
concept, IN THE SAME ORDER, no prose, no fences. Each series:

{
  "title": "<series title>",
  "logline": "<one sentence: the engine>",
  "premise": "<2-3 sentences: the situation, the central conflict, why it keeps generating scenes>",
  "tone": "<e.g. deadpan workplace comedy, escalating cringe, cozy-then-sinister>",
  "engine": "<the comedy engine, carried from the shortlist>",
  "pov": "<the human truth it exaggerates, carried from the shortlist>",
  "cast": [
    {
      "name": "<name>",
      "role": "<function + their want + their flaw>",
      "look": "<concrete, castable, consistent physical description>",
      "voice_id": "<one of: am_fenrir, am_michael, am_puck, am_adam, am_eric, af_bella, af_nicole, bf_emma>",
      "vibe": "<how they talk / their comedic angle>"
    }
  ],
  "episodes": [
    { "title": "<episode title>", "beat": "<the concrete scene that happens + how it advances the arc>", "button": "<the closing hook>" }
  ]
}

Requirements: 2-3 cast each with DISTINCT voice_ids from the list (match
gender/age); 7 episode beats forming a real escalating arc with a payoff; SFW.
Output ONLY the JSON object — same count and order as the shortlist.
