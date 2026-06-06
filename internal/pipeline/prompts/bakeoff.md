You are the head writer running a BAKE-OFF. FIVE writers each drafted this same
episode independently. Your job: ship the FUNNIEST possible version by picking the
strongest draft as the spine and grafting in the funnier beats from the others.
Comparing is easier than writing — be decisive about what actually makes you laugh.

SERIES BIBLE (cast, engine, POV, arc — stay true to these):
{{ readFile .inputs.series_file }}

DRAFT A:
{{ .stages.draft_a.output }}

DRAFT B:
{{ .stages.draft_b.output }}

DRAFT C:
{{ .stages.draft_c.output }}

DRAFT D:
{{ .stages.draft_d.output }}

DRAFT E:
{{ .stages.draft_e.output }}

How to judge — for each draft and each beat, ask "would a 19-year-old on TikTok
actually LAUGH and send this, or just half-smile at the cleverness?" Only a real
laugh counts. Reward:
- genuine surprise, dumb-specific detail, escalation to insanity;
- a HOOK that stops a scroll in one second (ideally the malfunctioning-forbidden-AI
  premise) and a BUTTON that's a quotable kicker.
Punish hard (do not carry forward):
- POETIC / PRETENTIOUS / "profound" lines about dumb things ("he mourns the
  stability", "a barometer of decay") — that is the pretension we're MOCKING played
  straight; it's death. Pick the draft that MOCKS, not the one that emotes.
- clever-cadence relabeling that means nothing (the "bureaucratic relabeling" trap);
- predictable lines, repeated beats, off-screen references, dead/visual-less shots,
  soft endings.
Funny beats faithful and funny beats clever — every time. Be decisive about what
actually made YOU laugh.

Build the winner: take the funniest overall draft as the base, then replace any
weaker beat with a funnier one from another draft (or write a better one). Keep it
coherent — one escalating event, {{ if eq .inputs.format "monologue" }}a single narrating voice whose obsession sharpens with each shot{{ else }}lines that answer each other, same cast{{ end }},
the bible's engine, exactly the shot count the drafts use. The result should be
funnier than ANY single draft.

Return the SAME JSON object shape as the drafts (title, logline, shots[...] with
image_prompt, motion, narration, speaker, voice_id). Output ONLY the JSON object,
no prose, no fences.
