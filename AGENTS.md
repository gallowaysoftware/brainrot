# AGENTS.md

Operating notes for agents (Claude Code, Aider, Codex, Cursor, …) working in
this repo. The user-facing model lives in `README.md`; this file captures the
conventions and invariants needed to make changes that fit.

## Repo at a glance

A single binary (`github.com/gallowaysoftware/brainrot`) — a serialized
short-form content mill. `ideate`/`plan`/`make` each build a `vamp` pipeline and
run it; `vibe` supervises the backends. brainrot is the pipeline, not the
inference stack.

- `cmd/brainrot/main.go` — the Cobra entrypoint and all the command glue
  (ideate, list, plan, make, render-judge [hidden], activate, doctor), plus the
  mechanical shot-list post-processing (`normalizeShots`, `cleanNarration`,
  `tagNarratorAnchors`).
- `internal/pipeline/` — the pipeline builders (`ideate.go`, `plan.go`,
  `episode.go`, `panel.go`, `scene.go`, `render_judge.go`, `stub.go`) plus the
  `go:embed`-ed prompts (`prompts/`) and ComfyUI graphs (`workflows/`).
- `internal/series/` — the series-library layout and data model (`series.json`
  bible, scores, voices, episode beats; library under `$XDG_STATE_HOME/brainrot/`).

Built against an **in-tree vibe checkout** (`replace => ../vibe` in `go.mod`):
brainrot tracks vibe's `vamp` (orchestration) and `contentkit` (studio
primitives: `Tournament`, `CritiqueRevise`, `Scorer`, `ReviewLoop`). Changes here
often pair with changes there.

## Inner loop

```
go build ./...
go vet ./...
go test ./...
gofmt -l .          # must print nothing
```

(Requires `../vibe` on disk, since the module `replace`s vibe to it.)

## Conventions

- **Mechanical guarantees go in code, not prompt rules.** Models treat prompt
  rules as advisory. Anything that must hold — voice normalization, stripping
  stage directions before TTS, anti-repeat episode dedup, narrator-anchor
  tagging by shot position — is enforced in Go after the model returns, not by
  asking the model nicely. Follow that pattern for new guarantees.
- **Goal is engagement, not faithfulness.** brainrot optimizes for
  bingeable/entertaining short video. (That's the opposite of a faithfulness-first
  prose pipeline — don't import "stay true to the source" instincts here.)
- **Reuse `contentkit` primitives** rather than re-implementing tournaments /
  critique-revise / scorecards / review loops; prompts stay brainrot's.
- **Stdlib first**, modern Go (`log/slog`, `errors.Join/Is/As`, `any`,
  `embed.FS`). Justify new dependencies.
- **Comments explain WHY, not WHAT.** No task-narration comments.
- **No emojis** in code, comments, or commit messages unless asked.
- **No new docs files** unless requested.

## Two-phase render invariant

`make` must unload the LLM before phase 2 loads the image model — they can't
co-reside on a 32GB GPU. `make` calls `vibe stop` between phases; preserve that
ordering when editing the render flow.
