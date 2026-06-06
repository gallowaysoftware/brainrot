# brainrot

A serialized short-form (TikTok-shaped) content mill: **ideate → score → plan → make**.

- **Breadth (`ideate`)** runs a studio development funnel over a niche — a writers' room fans out series concepts (a recurring cast plus an overarching arc), an adversarial producers' table argues them down, an audience team projects metrics, and a greenlight committee scores them. The best are kept.
- **Depth (`make`)** writes one *episode* of a kept series as a multi-character scene (a two-pass writers' room: draft → punch-up) and renders it into a vertical 1080×1920 AI video that stands alone and advances the larger story.

Built as a set of Go-DSL pipelines on top of [vibe](https://github.com/gallowaysoftware/vibe): `vibe` supervises the model backends (LLM, ComfyUI image/video, Kokoro TTS) under capability profiles, `vamp` orchestrates the DAGs, and `contentkit` provides the shared studio primitives (tournaments, critique/revise, scorecards, review loops).

## Prerequisites

brainrot is the *pipeline*, not the inference stack. It needs a running vibe stack:

- **A sibling `vibe` checkout.** `go.mod` has `replace github.com/gallowaysoftware/vibe => ../vibe`, and brainrot tracks vibe's `contentkit`/`vamp` packages. Clone vibe next to this repo:
  ```
  git clone https://github.com/gallowaysoftware/vibe          # ../vibe
  git clone https://github.com/gallowaysoftware/brainrot      # ../brainrot
  ```
  (Because of the `replace`, `go install …@latest` won't work — build from the checkout, below.)
- **The vibe daemon** running, with a `long_form` text profile (suggested: a 27B-class model such as Qwen3.6-27B-MTP at 128k+ context).
- **ComfyUI** on `:8188` for Qwen-Image stills + Wan2.2 image-to-video (`vibe start comfyui`). The two workflow graphs are bundled under `internal/pipeline/workflows/`.
- **Kokoro-FastAPI TTS** on `:8880` for character narration (`vibe start tts_kokoro`).
- **ffmpeg** on `$PATH` for assembly (stills → video → voice → captioned MP4).
- **~30GB VRAM** during generation. The LLM and the image model don't co-reside; `make` unloads the LLM (`vibe stop`) before phase 2 loads ComfyUI.

Run `brainrot doctor` to see what's up and what's missing; `brainrot activate` brings up the declared profile + services via vibe.

## Build

```bash
# with ../vibe checked out alongside:
cd brainrot
go build ./...
go install ./cmd/brainrot      # drops `brainrot` in $(go env GOPATH)/bin
```

## Workflow

```bash
brainrot activate                          # bring up long_form + comfyui + kokoro
brainrot ideate --niche "liminal office horror"   # develop + score series, keep the best
brainrot list                              # kept series with scores + episode progress
brainrot make <series-id>                  # write + render the next episode
```

`ideate` prints the kept series IDs and the full studio trace (concept pool, producer debate, projected metrics) so the run is inspectable. `make` with no `--episode` writes the next unrendered one; the finished `final.mp4` path is printed at the end.

### Optionally: mine a canon-backed series for episodes

A series with a `lore.md` canon pack (and optional `source.txt` seed) can be mined for a deep, fresh, ranked slate of episodes:

```bash
brainrot plan <series-id>            # stage a ranked slate (no changes)
brainrot plan <series-id> --review   # curate the slate accept/reject, one at a time
brainrot plan <series-id> --apply    # append the whole slate to the bible
```

Mining stays fresh — it won't repeat episodes already in the bible (an anti-repeat guard drops title collisions mechanically, not just by prompt).

## Commands

| Command | What it does |
|---|---|
| `ideate --niche <s>` | Develop + score series for a niche, keep the top ones. Flags: `--count` (shortlist size, default 6), `--keep` (default 4). |
| `list` | List kept series with scores and `done/total` episode progress. |
| `plan <id>` | Mine a series' canon (`lore.md` + `source.txt`) into a ranked slate of episode beats. Flags: `--count` (default 8), `--apply`, `--review`. |
| `make <id>` | Write + render one episode as a vertical video. See flags below. |
| `activate` | Bring up the vibe profile + services brainrot needs. |
| `doctor` | Read-only: what's running, what's missing. |

`make` flags:

| Flag | Effect |
|---|---|
| `--episode N` | Which episode (1-based); default = next unrendered. |
| `--shots N` | Shots in the episode (default 7; a series can set its own). |
| `--candidates N` | Write N independent candidate scripts; an editor/producer/money panel picks one to render. |
| `--finalists K` | Two-stage funnel: render the top K panel finalists and let a vision judge pick the best actual render (requires `--candidates > K`). |
| `--narrator <voice>` | Default Kokoro narrator voice. |
| `--publish-to <dir>` | Copy the finished `final.mp4` into `<dir>` as `<id>-<NNN>.mp4`. |
| `--preview` | Stills only: shot list + per-shot images, skip animation/voice/assembly (fast iteration). |
| `--script-only` | Phase 1 only: write the script, leave the LLM loaded. |
| `--render-only` | Phase 2 only: render the existing `shots.json` (use after `--script-only`). |

## How `make` renders

Two phases so the LLM is unloaded before the image model loads (they can't co-reside on 32GB):

1. **Phase 1 (LLM)** — draft the shot list, then punch it up. With `--candidates > 1`, write several and let a greenlight panel pick. Narration is mechanically cleaned (stage directions / shouty labels stripped so they don't reach the TTS or the burned captions).
2. **Phase 2 (ComfyUI + Kokoro + ffmpeg)** — Qwen-Image stills → Wan2.2 image-to-video → Kokoro voiceover → captioned, assembled MP4.

## Series library

Kept series live under `$XDG_STATE_HOME/brainrot/` (default `~/.local/state/brainrot/`):

```
series/
  <series-id>/
    series.json        the bible: cast, arc, episode beats, scores
    lore.md            optional canon pack (enables `plan` mining)
    source.txt         optional raw seed mined alongside lore
    anchors/<slug>.png  optional fixed character portrait (used verbatim for monologue narrators)
    episodes/<NNN>/      per-episode run dir (shots.json, images/, final.mp4)
  _dev/<stamp>/        full studio trace from an `ideate` run
  _rankings/<stamp>.json
  _planned/<stamp>/    staged `plan` slates
```

The bible is yours to hand-edit; brainrot appends to it (via `plan`) but doesn't clobber your edits.

## License

MIT — see [LICENSE](LICENSE). The pipeline is Galloway Software's; the content you generate with it is yours.
