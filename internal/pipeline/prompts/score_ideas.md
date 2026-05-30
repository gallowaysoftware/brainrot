You are a brutally honest short-form video producer. For each IDEA below, imagine
it actually produced as a ~25-second vertical TikTok by an AI pipeline (AI still
images animated into short clips via image-to-video, an AI voiceover, burned
captions) and score how well it would perform.

IDEAS:
{{ .stages.generate_ideas.output }}

Score EACH idea on these axes, 1-10 (1 = bad, 5 = mid, 8+ = genuinely strong;
be stingy — most ideas are mid, reserve 9-10 for the rare banger):
- hook: does shot 1 freeze a scrolling thumb in the first second?
- shootability: does it actually work as AI stills animated with SMALL motion?
  Penalize complex action, fast movement, many characters interacting, anything
  the image-to-video can't fake convincingly. Reward a strong subject + setting +
  subtle motion.
- punch: is there a real comedic or emotional payoff / escalation / twist?
- legibility: would a stranger with zero context fully get it in 25 seconds?
- loop: does it pull a rewatch, a share, or a comment?
- format_fit: does the format genuinely suit this bit?

Imagine the real result, not the pitch. An idea that sounds clever but would render
as a boring talking head, or needs lore to land, or describes action the i2v can't
show, should score LOW even if the concept is fun.

Return a SINGLE JSON object {"ideas": [ ... ]} with EXACTLY one entry per input
idea, IN THE SAME ORDER, no prose, no fences. Each entry:

{
  "score": {
    "hook": <1-10>,
    "shootability": <1-10>,
    "punch": <1-10>,
    "legibility": <1-10>,
    "loop": <1-10>,
    "format_fit": <1-10>,
    "rationale": "<one sentence: the deciding factor, good or bad>"
  }
}

Output ONLY the JSON object, exactly {{ .inputs.count }} entries, same order as input.
