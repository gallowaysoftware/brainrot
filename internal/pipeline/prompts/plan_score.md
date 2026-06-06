You are the greenlight committee scoring a slate of proposed episode beats for a
short-form show. Score each on the axes that predict a bingeable short, and rank
them. Be honest — a flat score helps no one.

THE SHOW (the bar each beat is judged against):
{{ readFile .inputs.series_file }}

THE PROPOSED EPISODES (score in the SAME ORDER they appear):
{{ .stages.write.output }}

Score each episode on these axes (1-10 each):
- hook: does shot 1 stop a scroll in under a second?
- escalation: does it build, beat to beat, to a real payoff (not a flat list)?
- shootability: can the local image model actually render its central images?
- faithfulness: is it genuinely anchored in the canon (real verbatim lines), not
  generic?
- bingeability: after this, do you want the next one?

Return a SINGLE JSON object, no prose, no fences. The "scores" array MUST be in the
same order as the input episodes (one entry each):

{
  "scores": [
    {
      "title": "<echo the episode title>",
      "hook": <1-10>,
      "escalation": <1-10>,
      "shootability": <1-10>,
      "faithfulness": <1-10>,
      "bingeability": <1-10>,
      "total": <sum of the five axes, 5-50>,
      "verdict": "<one line: keep as-is / keep-but-fix X / cut>"
    }
  ]
}

Output ONLY the JSON object.
