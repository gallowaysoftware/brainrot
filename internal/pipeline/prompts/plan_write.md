You are the showrunner turning a shortlist of approved episode veins into finished
episode beats for the series bible. Each becomes one entry in the bible's
"episodes" array — the spec a later writers' room writes the actual shot list
against. Write them in THIS show's voice and format.

THE SHOW (voice, format, engine, POV — match it exactly):
{{ readFile .inputs.series_file }}

CANON PACK (stay faithful; the quote bank is your source of anchors):
{{ readFile .inputs.lore_file }}
{{ if .inputs.source_file }}
FULL SOURCE (lift specific verbatim lines from here for each episode's lore):
{{ readFile .inputs.source_file }}
{{ end }}
APPROVED SHORTLIST (write one episode beat per entry; honor arc + arc_order so a
serialized thread reads in sequence):
{{ .stages.develop.output }}
{{ if .inputs.existing_file }}
ALREADY COVERED — do not reuse these episodes' signature lines or premises:
{{ readFile .inputs.existing_file }}
{{ end }}
Write LOOSE beats — a TERRITORY for a later writers' room to riff a fresh scene in,
NOT a script that prescribes specific events and lines. Don't hand the room a list
of verbatim quotes to recite (that produces a flat list of proverbs); hand it a
corner of the world and trust it to invent the scene. For each approved vein:
- "title": the episode title.
- "beat": 1-2 sentences — the TERRITORY this episode explores (which corner of the
  canon) and the tone/escalation, phrased as a prompt to riff on. Do NOT script the
  exact goblins, objects, or lines; leave that to the room.
- "button": a one-line hint at the kind of turn to land on (not the exact words).

Order the episodes so any arc runs in sequence and standalones are spread through.
Keep them DISTINCT — no two covering the same territory.

Return a SINGLE JSON object, no prose, no fences:

{
  "episodes": [
    {
      "title": "<episode title>",
      "beat": "<TERRITORY + tone/escalation, as a riff prompt — no scripted events>",
      "button": "<one-line hint at the kind of closing turn>"
    }
  ]
}

Output ONLY the JSON object.
