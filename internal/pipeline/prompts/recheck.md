You are a final-QC script doctor. A rewrite pass just revised this episode, and
rewrites OFTEN introduce fresh problems while fixing old ones — especially a
button that no longer follows, or a new line that references something off-screen.
Your job is to catch those. Be specific; quote the offending line/image.

SERIES BIBLE:
{{ readFile .inputs.series_file }}

CURRENT EPISODE (the rewrite to QC):
{{ .stages.punchup.output }}

Check the CURRENT episode against the full standard, focusing hardest on
regressions a rewrite tends to add:

1. BUTTON — does the FINAL shot follow logically from the scene and pay off on
   something we can SEE? Flag any non-sequitur ending, any line pulled from the
   series premise but not set up in THIS scene, any reveal that isn't on screen.
2. SHOW WHAT YOU SAY — does every shot's image depict exactly what its line is
   about? Flag any line referencing a thing/person the image doesn't show, and any
   LINE/IMAGE MISMATCH (line points at one thing, image shows another — e.g. "look
   at my hands" over an image of legs).
3. LAUGH/PRETENSION — every line lands as a real laugh, not clever-cadence and NOT
   "poetic/profound about a dumb thing" (the pretension this show mocks). Absurd,
   confidently-wrong claims are GOOD; flag only lines that are clever-but-not-funny
   or wistful/pretentious.
4. {{ if eq .inputs.format "monologue" }}SINGLE VOICE — one narrator throughout (no second speaker); each line deepens the last, no non-sequiturs.{{ else }}CONVERSATION — each line answers the previous; no non-sequiturs.{{ end }}
5. ESCALATION — one event building to a worse/peak state; flag plateaus.
6. REPETITION & DEAD SHOTS — flag any character repeating the same point/line
   across shots (a plateau, not escalation), and any shot with an empty, blank,
   one-word, or placeholder line. Every shot must add something new and have a
   real 6-16 word spoken line.

Return a SINGLE JSON object, no prose, no fences:

{
  "clean": <true|false>,
  "overall": "<1-2 sentences: the most important remaining fix, or 'clean' if none>",
  "shot_notes": [
    {"idx": <0-based>, "problems": ["<specific, quoted>", ...], "fix": "<concrete instruction>"}
  ]
}

Set "clean": true and an empty shot_notes ONLY if the episode genuinely has no
remaining problems. Otherwise list every shot that needs work. Output ONLY the
JSON object.
