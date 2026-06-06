You are doing a SURGICAL final pass on an episode of a serialized short-form show.
The script is already strong. A QC doctor flagged the remaining problems. Fix
EXACTLY those — minimally. Do not rewrite clean shots, do not restyle, do not
introduce new ideas; change only what the notes require. The biggest risk now is
ADDING a new problem, so be conservative.

SERIES BIBLE (stay true to cast voices + the arc):
{{ readFile .inputs.series_file }}

CURRENT EPISODE:
{{ .stages.punchup.output }}

QC NOTES (resolve every one; if the notes say "clean", return the episode unchanged):
{{ .stages.recheck.output }}

Rules while fixing:
- Keep the same cast, beat, and number of shots; keep every shot that's already fine.
- Every line stays a real laugh (absurd-and-confident is good; no pretentious/poetic
  or clever-not-funny lines), 6-16 words, spoken, in the speaker's bible voice_id;
  {{ if eq .inputs.format "monologue" }}every line is the same single narrator, each more unhinged than the last.{{ else }}each line answers the previous one.{{ end }}
- The image_prompt must DEPICT exactly what its line is about (restate looks
  verbatim{{ if ne .inputs.format "monologue" }}; both characters in a two-shot{{ end }}). Fix any line/image mismatch by aligning
  them — usually change the image to match the line, or the line to match the image.
- The BUTTON must follow from THIS scene and pay off on something on screen.
- If two shots make the same point, merge or replace one so every shot is a
  distinct beat. Never output an empty, blank, or one-word narration line.
- MOTION stays slow and subtle (no slamming/snapping/jumping/tearing beats).
- SFW.

Return the SAME JSON object shape (title, logline, shots[...]). Output ONLY the
JSON object, no prose, no fences.
