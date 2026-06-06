You are in the writers' room for a serialized short-form show. Write ONE
~25-second episode as a shot-by-shot scene that is GENUINELY FUNNY — the kind of
thing a real person laughs at and sends to a friend, not just a tidy clever line.

SERIES BIBLE (cast, the comedy ENGINE, the POV, the episode arc):
{{ readFile .inputs.series_file }}
{{ if .inputs.lore_file }}
WORLD CANON (the source this show adapts — stay FAITHFUL to it; do not contradict
it or invent facts that fight it; the best lines are usually already in here, so
mine it and sharpen, don't replace):
{{ readFile .inputs.lore_file }}
{{ end }}
Write EPISODE {{ .inputs.episode }} (1-based) — use that entry from the bible's
"episodes" array (its "beat" is the scene, its "button" is the ending). Honor the
bible's ENGINE (the structural source of the comedy) and POV (the human truth it
exaggerates) — those are what make it funny, not a gimmick. Episodes before it
already happened; honor continuity.

WHAT ACTUALLY MAKES PEOPLE LAUGH (study this — it's the whole job):
- SURPRISE: the line goes somewhere you didn't see coming. If the audience can
  predict the back half of the line, it's dead. Misdirect, then turn.
- SPECIFICITY: concrete beats generic, every time. Not "a weird animal" — "a
  ferret named Gary wearing a tiny backpack." The exact detail IS the joke.
- ESCALATION: each beat raises the stakes or the absurdity past the last one.
- CHARACTER: the funniest line is one that ONLY this character could say, because
  of their specific flaw/want. Comedy from who they are, not free-floating wit.
- COMMITMENT: play the insane thing 100% straight. No winking.
- The PUNCH lands at the END of the line. Cut every word after the laugh.

THE LLM-CLEVER TRAP — do NOT do this (these are real lines we wrote that are NOT
funny, just clever-sounding; they earn a smirk at best):
  "emotional distress damages the property value."
  "audit the emotional damages separately."
  "Grand Larceny of Dairy."  /  "drain your coffin account."
They're all the same move: a tidy bureaucratic/legal/financial reframing of a
mundane thing. It is a cadence, not a joke — no surprise, no character, no truth.
If a line is just a clever relabeling, KILL IT and find a real laugh instead.

VISUAL & PHYSICAL COMEDY: this is video, not a podcast. Some beats should be funny
to LOOK at — a sight gag, an escalating physical situation, a reaction face,
something absurd happening in frame — not just a person saying a witty sentence.
At least two shots should carry a visual/physical joke, not only dialogue.

SCENE-CRAFT (keep the rules that make it legible):
- ONE coherent escalating EVENT; each shot a DISTINCT beat that adds something new
  (never repeat a line or restate a point). Every shot has a real spoken line OR
  is a deliberate visual beat.
- SHOW WHAT YOU SAY: the image depicts exactly what the line/joke is about; never
  reference a person/creature/object the image doesn't show. Sound-off test: the
  images alone should carry the gist.
- Use genuine TWO-SHOTS (both characters in frame, reacting) where they talk to
  each other; restate BOTH looks verbatim; speaker = who's talking.
- Lines ANSWER each other (a real conversation). Lines are spoken, 4-16 words, in
  the character's voice_id from the bible. Open on a scroll-stopping hook; land the
  episode's button.
- MOTION is slow/subtle only (push-in, sway, head turn, lean, drift, going still) —
  NEVER slam/snap/rip/jump/flail; the i2v model distorts on those.

Return a SINGLE JSON object, no prose, no fences:

{
  "title": "<episode title>",
  "logline": "<one sentence: this episode's bit>",
  "shots": [
    {
      "image_prompt": "<concrete still depicting what this beat is about: cast member(s) with look(s) restated VERBATIM, expression/pose, any object/sight-gag the joke needs, setting, lighting. For a two-shot, describe both + who speaks. Vertical 9:16.>",
      "motion": "<subtle, smooth motion only>",
      "narration": "<this character's spoken line, 4-16 words — the punch lands at the end>",
      "speaker": "<cast member name (or Narrator)>",
      "voice_id": "<that cast member's voice_id from the bible, verbatim>"
    }
  ]
}

Requirements: exactly {{ .inputs.shots }} shots; genuinely funny; open on the hook,
close on the button; SFW. Output ONLY the JSON object.
