package pipeline

import (
	"fmt"

	"github.com/gallowaysoftware/brainrot/internal/series"
	"github.com/gallowaysoftware/vibe/contentkit"
	"github.com/gallowaysoftware/vibe/vamp"
)

// EpisodeConfig drives the depth SCRIPT phase (LLM only): turn one episode beat
// from a series bible into a multi-character shot list, via a two-pass writers'
// room (draft -> punch-up). The render phases (BuildSceneRender/Preview) consume
// the resulting shots.json, so the ~26GB LLM unloads before the image model.
type EpisodeConfig struct {
	// SeriesFile is the path to the series bible (series.json).
	SeriesFile string
	// LoreFile is an optional path to the series' canon pack (lore.md). When set,
	// the writers' room reads it so a lore-heavy source stays faithful and reaches
	// the screen rather than being distilled away.
	LoreFile string
	// Format selects the writing prompt: series.FormatScene (default,
	// multi-character scene) or series.FormatMonologue (single narrating voice).
	Format string
	// Episode is the 1-based episode number to write.
	Episode int
	Shots   int
}

// BuildEpisodeScript is phase 1 (LLM only): write one episode of the series as a
// multi-character scene via a diagnose-then-fix writers' room, then enumerate the
// shots into the shots.json shape phase 2 renders.
//
//	draft_a..e    (text)   -> five independent comedic takes (varied temp)
//	bakeoff       (text)   -> bakeoff.json  — funniest version, best beats grafted
//	critique      (text)   -> critique.json — rule audit + LAUGH AUDIT on the winner
//	punchup       (text)   -> punched.json  — rewrite: SMIRK/FLAT shots -> real laughs
//	recheck       (text)   -> recheck.json  — QC the REWRITE (catches regressions)
//	polish        (text)   -> polished.json — surgical fix of recheck's notes
//	tighten       (text)   -> tightened.json — compression pass for TikTok pacing
//	enumerate_shots (render) -> shots.json    — {"items":[{idx,...}]} for phase 2
//
// A funniest-take tournament feeds two diagnose-then-fix cycles. Five drafts give
// more shots on goal; the bake-off picks/merges the funniest (comparison beats
// single-shot for humor). The critique steps DIAGNOSE violations (incl. a harsh
// laugh audit) so the fix passes are concrete, not "make it better"; the second
// cycle catches problems the rewrite itself introduced; tighten then cuts to length.
func BuildEpisodeScript(cfg EpisodeConfig) (*vamp.Pipeline, error) {
	if cfg.Shots <= 0 {
		cfg.Shots = 7
	}
	if cfg.Episode <= 0 {
		cfg.Episode = 1
	}
	// Pick the draft prompt by format: a multi-character scene (lines answer each
	// other) or a single-voice monologue (one narrator delivers an escalating
	// lore-drop over illustrated tableaux). Both share the punch-up chain.
	draftPrompt := "episode_script.md"
	if cfg.Format == series.FormatMonologue {
		draftPrompt = "episode_monologue.md"
	}

	p := vamp.New("brainrot-episode-script").
		Describe("Write one serialized episode as a shot list (draft + punch-up).")

	format := cfg.Format
	if format == "" {
		format = series.FormatScene
	}

	p.Input("series_file", vamp.Required(), vamp.WithDefault(cfg.SeriesFile),
		vamp.Describe("Path to the series bible (series.json)."))
	p.Input("lore_file", vamp.WithDefault(cfg.LoreFile),
		vamp.Describe("Optional path to the series' canon pack (lore.md)."))
	p.Input("format", vamp.WithDefault(format),
		vamp.Describe(`Episode format: "scene" (multi-character) or "monologue" (single voice). Gates the conversation/two-shot rules in the shared punch-up prompts.`))
	p.Input("episode", vamp.WithDefault(fmt.Sprintf("%d", cfg.Episode)),
		vamp.Describe("1-based episode number to write."))
	p.Input("shots", vamp.WithDefault(fmt.Sprintf("%d", cfg.Shots)),
		vamp.Describe("Number of shots in the episode."))

	p.RequireProfile("long_form")
	p.RequireGPUMemory("~30GB during generation")
	p.CapabilityModel("long_form", vamp.ModelHint{
		MinParams: "27B", MinContext: 131072,
		SuggestedModel: "qwen3.6-27b-mtp-q6_k",
	})

	// Five independent drafts at varied temperatures — different comedic swings at
	// the same beat. Time is cheap; more shots on goal means a funnier winner.
	mkDraft := func(name string, temp float64) *vamp.TextStage {
		return contentkit.LongFormText(p, name, temp, 12288).
			PromptFS(PromptsFS, draftPrompt).
			OutputFormatJSON().
			Output(name + ".json")
	}
	// Five independent swings — comedy lives in the variance, so cast a wide net
	// and let the bake-off pick the rare gold rather than polishing one draft.
	draftA := mkDraft("draft_a", 0.85)
	draftB := mkDraft("draft_b", 1.05)
	draftC := mkDraft("draft_c", 0.95)
	draftD := mkDraft("draft_d", 1.15)
	draftE := mkDraft("draft_e", 0.9)

	// Bake-off: head writer ships the funniest version, using the strongest draft
	// as the spine and grafting in funnier beats from the others.
	bakeoff := contentkit.LongFormText(p, "bakeoff", 0.6, 12288).
		After(draftA, draftB, draftC, draftD, draftE).
		PromptFS(PromptsFS, "bakeoff.md").
		OutputFormatJSON().
		Output("bakeoff.json")

	// Critique: a script doctor names every rule violation + runs a LAUGH AUDIT
	// (LAUGH/SMIRK/FLAT per shot). Low temp, analytical — honest diagnosis.
	critique := contentkit.LongFormText(p, "critique", 0.2, 8192).
		After(bakeoff).
		PromptFS(PromptsFS, "critique.md").
		OutputFormatJSON().
		Output("critique.json")

	// Punch-up: rewrites the bake-off winner, turning SMIRK/FLAT shots into real
	// laughs and resolving every critique note.
	punch := contentkit.LongFormText(p, "punchup", 0.7, 12288).
		After(bakeoff, critique).
		PromptFS(PromptsFS, "punchup.md").
		OutputFormatJSON().
		Output("punched.json")

	// Recheck: a second diagnosis, this time on the REWRITE — to catch problems
	// the punch-up introduced (a button that no longer follows, a new off-screen
	// reference, a line/image mismatch) that the first critique never saw.
	recheck := contentkit.LongFormText(p, "recheck", 0.2, 8192).
		After(punch).
		PromptFS(PromptsFS, "recheck.md").
		OutputFormatJSON().
		Output("recheck.json")

	// Polish: a surgical final pass — fix only what recheck flagged, minimally,
	// so a strong script isn't churned (and no new problems get introduced).
	polish := contentkit.LongFormText(p, "polish", 0.5, 12288).
		After(punch, recheck).
		PromptFS(PromptsFS, "polish.md").
		OutputFormatJSON().
		Output("polished.json")

	// Tighten: a single-mandate compression pass for TikTok pacing. The writers'
	// room reliably overshoots length (it ignores word caps in a prompt that also
	// asks it to be funny), so a separate pass whose ONLY job is to cut — keep the
	// funniest punch per shot, drop all setup, touch nothing but the narration —
	// gets the runtime down where the combined prompt can't. Low temp; surgical.
	tighten := contentkit.LongFormText(p, "tighten", 0.3, 8192).
		After(polish).
		PromptFS(PromptsFS, "tighten.md").
		OutputFormatJSON().
		Output("tightened.json")

	contentkit.EnumerateItems(p, contentkit.EnumerateConfig{
		From:      tighten,
		StageName: "enumerate_shots",
		Output:    "shots.json",
		ArrayKey:  "shots",
		IndexKey:  "idx",
		Fields:    []string{"image_prompt", "motion", "narration", "speaker", "voice_id"},
	})

	return p.Build()
}
