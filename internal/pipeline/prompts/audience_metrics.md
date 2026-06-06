You are the AUDIENCE STRATEGY / data team at a short-form studio. For each series
bible below, project how it would actually perform on a vertical-video feed.
Be a realist, not a hype man — most shows are mid; reserve high numbers for
concepts that genuinely earn them.

SERIES BIBLES:
{{ .stages.pitches.output }}

For each series, estimate (think about the engine, the hook of episode 1, the
cast chemistry, and how shareable/bingeable the premise is):
- scroll_stop: % of feed-scrollers who stop on the first shot of episode 1 (0-100).
- retention:   % of those who watch a full ~25s episode to the end (0-100).
- shareability: 1-10 — would someone send this to a friend or stitch it?
- follow_intent: 1-10 — after one episode, do they follow the account for more?
- audience: the core target audience + a rough sense of how big it is.
- comps: 1-2 comparable hits/formats it's in the lane of.

Return a SINGLE JSON object {"series": [ ... ]}, EXACTLY one entry per input
series, IN THE SAME ORDER, no prose, no fences. Each entry:

{
  "metrics": {
    "scroll_stop": <0-100>, "retention": <0-100>,
    "shareability": <1-10>, "follow_intent": <1-10>,
    "audience": "<core audience + rough size>",
    "comps": "<comparable hits/formats>",
    "note": "<one sentence: the metric that makes or breaks it>"
  }
}

Output ONLY the JSON object — same count and order as the input.
