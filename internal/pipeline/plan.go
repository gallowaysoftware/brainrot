package pipeline

import (
	"fmt"

	"github.com/gallowaysoftware/vibe/contentkit"
	"github.com/gallowaysoftware/vibe/vamp"
)

// PlanConfig drives the episode-MINING funnel: take ONE rich canon seed (a
// series' lore pack + raw source) and mine it into a deep, fresh, ranked slate of
// episode beats — the inverse of ideate (which invents series from a niche). This
// is what lets a 14k-word source yield 30+ episodes instead of a hand-authored 11.
type PlanConfig struct {
	// SeriesFile is the bible (series.json) — voice, format, cast, engine, POV.
	SeriesFile string
	// LoreFile is the curated canon pack (lore.md).
	LoreFile string
	// SourceFile is the full raw source the canon was distilled from (source.txt).
	SourceFile string
	// ExistingFile is an optional path to a generated list of already-covered
	// canon (existing episode titles + their lore), so mining stays FRESH and
	// doesn't repeat shipped episodes. Empty = nothing covered yet.
	ExistingFile string
	// Count is how many episode beats to develop into the final slate.
	Count int
}

// BuildPlan constructs the episode-mining funnel, mirroring ideate's proven shape
// (persona room -> adversarial table -> write -> score) but pointed at an existing
// canon seed:
//
//	mine     (text) -> pool.json        — 4 personas each mine a different angle of
//	                                      the canon into candidate episode "veins",
//	                                      each anchored to verbatim source lines
//	develop  (text) -> development.json — an adversarial table narrows the pool:
//	                                      the skeptic kills the unshootable, the
//	                                      redundant, and the thesis-without-a-joke
//	write    (text) -> beats.json       — the survivors become full EpisodeBeats in
//	                                      the series' voice/format, anchored to canon,
//	                                      not overlapping already-covered episodes
//	score    (text) -> scores.json      — rank each beat (hook / escalation /
//	                                      shootability / faithfulness / bingeability)
//
// The CLI reads beats.json + scores.json, ranks, and stages/appends the top Count
// beats to the bible (non-destructively).
func BuildPlan(cfg PlanConfig) (*vamp.Pipeline, error) {
	if cfg.Count <= 0 {
		cfg.Count = 8
	}
	p := vamp.New("brainrot-plan").
		Describe("Episode-mining funnel: mine a canon seed -> adversarial narrow -> write beats -> score.")

	p.Input("series_file", vamp.Required(), vamp.WithDefault(cfg.SeriesFile),
		vamp.Describe("Path to the series bible (series.json)."))
	p.Input("lore_file", vamp.Required(), vamp.WithDefault(cfg.LoreFile),
		vamp.Describe("Path to the curated canon pack (lore.md)."))
	p.Input("source_file", vamp.WithDefault(cfg.SourceFile),
		vamp.Describe("Optional path to the full raw source (source.txt)."))
	p.Input("existing_file", vamp.WithDefault(cfg.ExistingFile),
		vamp.Describe("Optional path to already-covered canon (keeps mining fresh)."))
	p.Input("count", vamp.WithDefault(fmt.Sprintf("%d", cfg.Count)),
		vamp.Describe("How many episode beats to develop into the slate."))

	p.RequireProfile("long_form")
	p.RequireGPUMemory("~30GB during generation")
	p.CapabilityModel("long_form", vamp.ModelHint{
		MinParams: "27B", MinContext: 131072,
		SuggestedModel: "qwen3.6-27b-mtp-q6_k",
	})

	// Mine: 4 personas fan out a diverse pool of candidate episode veins from the
	// canon. Hot for divergence; JSON gate + retry keep it valid.
	mine := contentkit.LongFormText(p, "mine", 0.9, 12288).
		PromptFS(PromptsFS, "plan_mine.md").
		OutputFormatJSON().
		Output("pool.json")

	// Develop: an adversarial producers' table argues the pool down to the Count
	// strongest, killing the unshootable / redundant / jokeless, with notes.
	develop := contentkit.LongFormText(p, "develop", 0.5, 8192).
		After(mine).
		PromptFS(PromptsFS, "plan_develop.md").
		OutputFormatJSON().
		Output("development.json")

	// Write: the survivors become full EpisodeBeats in the series' voice + format,
	// each anchored to verbatim canon and not overlapping covered episodes.
	write := contentkit.LongFormText(p, "write", 0.7, 16384).
		After(develop).
		PromptFS(PromptsFS, "plan_write.md").
		OutputFormatJSON().
		Output("beats.json")

	// Score: rank each beat on the axes that matter for a bingeable short-form arc.
	contentkit.LongFormText(p, "score", 0.3, 8192).
		After(write).
		PromptFS(PromptsFS, "plan_score.md").
		OutputFormatJSON().
		Output("scores.json")

	_ = write
	return p.Build()
}
