package pipeline

import (
	"fmt"

	"github.com/gallowaysoftware/vibe/contentkit"
	"github.com/gallowaysoftware/vibe/vamp"
)

// IdeateConfig drives the studio development funnel: a writers' room fans out
// concepts, an adversarial producers' table narrows them, the room writes full
// bibles, an audience team projects metrics, and a greenlight committee scores.
type IdeateConfig struct {
	Niche string
	Count int // how many concepts to develop into full pitches
}

// BuildIdeate constructs the studio development pipeline:
//
//	writers_room     (text) -> a diverse pool of concepts from 4 writer voices
//	development      (text) -> adversarial producers narrow to Count shortlist
//	pitches          (text) -> series.json   — full bibles for the shortlist
//	audience_metrics (text) -> per-series scroll/retention/share/follow projection
//	greenlight       (text) -> scores.json   — craft + metrics + greenlight number
//
// The CLI reads series.json + scores.json, merges, ranks by greenlight, keeps K.
func BuildIdeate(cfg IdeateConfig) (*vamp.Pipeline, error) {
	if cfg.Count <= 0 {
		cfg.Count = 6
	}
	p := vamp.New("brainrot-ideate").
		Describe("Studio development funnel: writers' room -> producers -> pitches -> audience metrics -> greenlight.")

	p.Input("niche", vamp.Required(), vamp.WithDefault(cfg.Niche),
		vamp.Describe("The content niche to develop serialized shows within."))
	p.Input("count", vamp.WithDefault(fmt.Sprintf("%d", cfg.Count)),
		vamp.Describe("How many concepts to develop into full pitches."))

	p.RequireProfile("long_form")
	p.RequireGPUMemory("~30GB during generation")
	p.CapabilityModel("long_form", vamp.ModelHint{
		MinParams: "27B", MinContext: 131072,
		SuggestedModel: "qwen3.6-27b-mtp-q6_k",
	})

	// Writers' room: 4 distinct writer voices fan out a diverse concept pool.
	// Hot for divergence; JSON gate + retry keep it valid.
	room := contentkit.LongFormText(p, "writers_room", 0.95, 12288).
		PromptFS(PromptsFS, "writers_room.md").
		OutputFormatJSON().
		Output("pool.json")

	// Development: an adversarial producers' table (champion/skeptic/showrunner)
	// argues and narrows the pool to the Count strongest, with notes.
	dev := contentkit.LongFormText(p, "development", 0.55, 8192).
		After(room).
		PromptFS(PromptsFS, "development.md").
		OutputFormatJSON().
		Output("development.json")

	// Pitches: the room writes full bibles for the shortlist (the Series array).
	pitches := contentkit.LongFormText(p, "pitches", 0.7, 28672).
		After(dev).
		PromptFS(PromptsFS, "pitches.md").
		OutputFormatJSON().
		Output("series.json")

	// Audience metrics: a data team projects scroll-stop / retention / share /
	// follow per pitch. Cool and realist.
	metrics := contentkit.LongFormText(p, "audience_metrics", 0.3, 8192).
		After(pitches).
		PromptFS(PromptsFS, "audience_metrics.md").
		OutputFormatJSON().
		Output("metrics.json")

	// Greenlight: the committee scores craft, folds in the metrics, and issues a
	// single greenlight number per series (the ranking sorts on it).
	contentkit.LongFormText(p, "greenlight", 0.3, 10240).
		After(pitches, metrics).
		PromptFS(PromptsFS, "greenlight.md").
		OutputFormatJSON().
		Output("scores.json")

	return p.Build()
}
