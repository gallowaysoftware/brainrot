You are a chronically-online short-form video creator. Build ONE punchy
~25-second vertical TikTok from the IDEA below and break it into shots.

IDEA:
{{ readFile .inputs.idea_file }}

The idea gives you a character (with a look and a voice), a situation, a hook,
and a format. Commit to all of them — open on the hook, deliver the situation,
land a kicker.

MOST IMPORTANT RULE — make it land for a COLD viewer:
- Assume the viewer has NEVER heard of any of this and gives you 25 seconds on a
  scroll. The video must make complete sense and pay off ENTIRELY on its own.
- Open on a concrete, instantly-readable image + the hook. NEVER open on setup
  or backstory.
- Stakes must be HUMAN and visceral (pain, fear, greed, embarrassment, wanting
  something), shown not explained. Almost no proper nouns; if one appears, make
  its meaning obvious from what's shown.

HARD VISUAL RULES (the AI renders each shot as a separate still + small motion —
break these and the video looks broken):
- ONE character, the SAME one, fully visible in EVERY shot, described IDENTICALLY
  every time. Never an empty/bodiless frame, never a second person, never a crowd.
  The character is ALONE in frame — end every image_prompt with "solo portrait,
  she/he is completely alone, no other people, no bystanders, no reflections of
  other people" so a stray figure can't drift into the edge.
- The IMAGE must show exactly what the NARRATION is about. If a line mentions
  another person or thing, do NOT use it — the camera only ever shows this one
  character. Never reference anything that isn't in the frame.
- MOTION must be minimal and smooth ONLY: a slow push-in, a slight sway, a subtle
  head turn, a small hand gesture, blinking, hair/cloth drift. NEVER dancing,
  jumping, running, flailing, big or fast action — the animator distorts on those.

Tone & rules of the game:
- TikTok is PUNCH. Shot 1 is a scroll-stopper. No throat-clearing.
- Entertaining and a little meta/self-aware — internet-native voice, not earnest
  cinema. Funny, eerie, or hype, but always ENERGY.
- Every narration line is a hook, a punchline, or a reveal — 6 to 14 words,
  spoken aloud, plain English a stranger gets on first listen. Vary the rhythm;
  land a kicker (or loop back to shot 1) on the final shot.

Return a SINGLE JSON object, no prose, no fences, matching exactly:

{
  "title": "<3-6 word title>",
  "logline": "<one sentence: the bit>",
  "shots": [
    {
      "image_prompt": "<complete, concrete image-generation prompt for this shot's single still frame: subject, composition, setting, lighting. Restate the character's look from the idea VERBATIM every time they appear so they stay consistent. Vertical 9:16 framing, cinematic, high detail.>",
      "motion": "<subtle, physically-plausible motion to animate the still: a slow push-in, a head turn, drifting particles, breathing. Small and natural — image-to-video breaks on big moves.>",
      "narration": "<this shot's spoken line, 6-14 words, punchy and in-format>",
      "speaker": "<character name from the idea, or 'Narrator'>",
      "voice_id": "<copy the idea character's voice_id verbatim for that character, or am_fenrir for Narrator. Never invent.>"
    }
  ]
}

Requirements:
- Exactly {{ .inputs.shots }} shots. Front-load the hook; land a kicker last.
- image_prompt is self-contained (the image model sees only that string):
  restate the character look every time.
- motion stays subtle and depictable. No teleporting, no cuts within a shot.
- voice_id copied verbatim from the idea (or am_fenrir) — never invent.
- Concrete and human over abstract.

Output ONLY the JSON object.
