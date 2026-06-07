# Lore-driven series (monologue format + lore channel) — and the pipeline roadmap

Context: brainrot was built to INVENT series from a `--niche` and write
multi-character scenes. "Goblin Town: The Insane Ramblings of Codex" inverts that:
we already have a rich **seed** (Codex's 14k-word goblin essays) and the comedy is
**one escalating voice**, not a conversation. Two changes were needed, plus a
roadmap for turning a good seed into a lot of great content.

## What shipped (done)

### 1. A lore channel — so a lore-heavy source reaches the screen, not the bin
The episode writer used to see only the bible (cast + engine + episode beats), so a
big source got distilled away. Now:
- `Series.Episodes[n].Lore` — per-episode canon: the exact verbatim lines + named
  objects that episode must surface. The bible names what to show, episode by episode.
- A sibling **`lore.md`** in the series dir (`series.LoreFile(id)` / `HasLore`): the
  curated canon pack (world rules, running gags, the quote bank, named entities). It
  is auto-injected into the writers' room when present (`EpisodeConfig.LoreFile`),
  under a `WORLD CANON` block that says: stay faithful, mine the canon, lift the
  sharp phrasing near-verbatim, don't invent facts that fight it.
- Full raw source kept as `source.txt` for reference / future mining.

Faithfulness model chosen: **riff-in-voice, grounded** — the LLM tightens canon for
the clock, lifting real lines where they already land. (Verbatim-only and
loose-inspiration are the other two dials; this is the middle.)

### 2. A `monologue` episode format — single-voice shows
`Series.Format` = `"scene"` (default) | `"monologue"`. Monologue uses
`episode_monologue.md` (one narrator, escalating lore-drop over illustrated
tableaux) instead of `episode_script.md`. The shared punch-up chain (bakeoff →
critique → punchup → recheck → polish) is now **format-aware**: a `format` input
gates every scene-specific rule (lines-answer-each-other, two-shots,
characters-play-off-each-other) and swaps in the monologue equivalent (one voice,
each shot a sharper turn, distinct tableau per shot). Render (scene.go) is
unchanged — it just renders shots.

Seed authored at `~/.local/state/brainrot/series/goblin-town/` (series.json,
lore.md, source.txt): 11 episodes mapped to the 11 essays, episodes 4–10 the
serialized "Coat Saga." Codex narrates (am_fenrir); Sootlip + Thimble are recurring
silent illustrated goblins. Ready for `brainrot make goblin-town --script-only` the
moment the GPU is free.

### 3. Character anchor images — a fixed portrait that never drifts
The narrator (Codex) is the show's face, but image models re-hallucinate it every
shot and a lens/iris/eye look mangles under Qwen+Wan — which is why the prompt had
to settle for "a green dot in a void." Anchors let a hand-made portrait be used
verbatim instead:
- Drop `series/<id>/anchors/<cast-slug>.png` (e.g. `anchors/codex.png`, a 9:16
  still). `series.Layout` gains `AnchorDir` / `AnchorFile` / `HasAnchor`.
- `cmd/brainrot` `tagNarratorAnchors` marks the monologue's first + last shot (the
  cold-open and the meltdown — always Codex) with the narrator's slug. Done
  deterministically by POSITION after the script is written, so the writers'-room
  rewrite chain can't strip it (a model-emitted field wouldn't survive bakeoff →
  punch-up → polish → tighten).
- `pipeline.SceneConfig.AnchorDir` + `stillStages` (scene.go) split phase-2 stills:
  a `split_generated` render stage filters anchored shots OUT of generation so only
  non-anchored shots cost a Qwen still; anchored shots are never generated. The Wan
  i2v start-image (`animateInput`) then animates each anchored shot's fixed
  `anchors/<slug>.png` directly (Wan's InputImage accepts the absolute path), while
  every other shot animates its generated still. Entirely additive — a series with
  no `anchors/` dir renders exactly as before.

Net: the Codex orb (a proper HAL-style green lens, made off-pipeline) is pixel-
identical in every episode, and the meltdown animates the real portrait instead of
a re-rolled green blob. `goblin-town` ships with `anchors/codex.png` staged.
Anchors generalize to any character, not just narrators.

## The roadmap — incorporating the proven idea-gen loops

The sibling content pipelines and `contentkit`/
METHODOLOGY.md already encode the patterns that turn a 27B into good long-form. We
should pull them through brainrot, in this order:

### A. Consolidate the depth chain onto `contentkit` (coordinate w/ the refactor agent)
`BuildEpisodeScript`'s `draft_a/b/c → bakeoff` IS `contentkit.Tournament`; its
`critique → punchup` and `recheck → polish` ARE two `contentkit.CritiqueRevise`
cycles. Re-wire onto those primitives (prompts stay brainrot's, per contentkit's
split). Net: less code, and the format-gating + lore channel ride along for free.
Risk: the shared-engine agent may already be doing this — sync before editing.

### B. An episode-MINING loop — the real "idea-gen loop" for a seed (NEW)
`ideate` invents *series* from a niche. A rich canon needs the opposite: mine ONE
seed into a deep, fresh, ranked **slate of episodes**. Proposed `brainrot plan
<series>`:
- writers'-room fan-out over `source.txt` + `lore.md` → many candidate episode
  beats, each anchored to specific verbatim canon lines (so lore stays on screen);
- adversarial narrowing (the skeptic kills the unshootable + the redundant);
- a greenlight/score pass → ranked beats appended to the bible.
This is what lets the goblin seed yield 30+ episodes, not 11. Built on Tournament +
the room/skeptic pattern (#4, #5).

### C. Canon = what shipped (anti-repeat) — pattern #7
Track which canon lines/bits each shipped episode actually used (re-derived against
the finished `shots.json`, not what we intended). Feed "already spent" into the
plan + episode writer so episode N+1 doesn't reuse episode N's quotes. Without this,
a finite quote bank repeats and the series rots. (Optionally a
fog-of-war "where the arc is going" layer for the Coat Saga.)

### D. Deterministic scorecard — pattern #8 (`contentkit.Scorer`)
brainrot has the LLM "laugh audit" but no *counted* anti-slop. Add a brainrot
`Scorer`: slop-word density, the "not X but Y" reflex (ironic, since it's also
Codex's signature — so for goblin-town, track-don't-punish), repeated trigrams,
narration length distribution. Surface a per-episode scorecard + trend. Measure
quality instead of guessing.

### E. Stage + curate, never clobber — pattern #10 (`contentkit.ReviewLoop`)
When `plan` proposes episodes, stage them for accept/edit/reject rather than
overwriting the bible. Same stage-and-curate loop a serialized-fiction pipeline uses for dossiers.

## Notes / constraints
- VRAM forces the two-phase shape (LLM unload → image model). Already handled by
  make's phase split; the plan loop is phase-1-only (LLM), like ideate.
- Short-form video fights the stack's weaknesses (slideshow visuals, viral comedy).
  The monologue lore-drop leans into a strength: a confident voice + specific,
  already-funny canon. Manage visual expectations accordingly.
