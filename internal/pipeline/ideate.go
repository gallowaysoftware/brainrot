package pipeline

import (
	"fmt"
	"time"

	"github.com/gallowaysoftware/vibe/vamp"
)

// IdeateConfig drives the studio development funnel: a writers' room fans out
// concepts, an adversarial producers' table narrows them, the room writes full
// bibles, an audience team projects metrics, and a greenlight committee scores.
type IdeateConfig struct {
	Niche string
	Count int // how many concepts to develop into full pitches
}

var thinkingOff = map[string]any{"enable_thinking": false}

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

	retry := &vamp.RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     30 * time.Second,
		RetryOn:        []string{"transient", "invalid_output"},
	}

	// Writers' room: 4 distinct writer voices fan out a diverse concept pool.
	// Hot for divergence; JSON gate + retry keep it valid.
	room := p.Text("writers_room").
		Capability("long_form").
		PromptFS(PromptsFS, "writers_room.md").
		OutputFormatJSON().
		Output("pool.json").
		Param("temperature", 0.95).
		Param("max_tokens", 12288).
		Param("chat_template_kwargs", thinkingOff).
		Retry(retry)

	// Development: an adversarial producers' table (champion/skeptic/showrunner)
	// argues and narrows the pool to the Count strongest, with notes.
	dev := p.Text("development").
		Capability("long_form").
		After(room).
		PromptFS(PromptsFS, "development.md").
		OutputFormatJSON().
		Output("development.json").
		Param("temperature", 0.55).
		Param("max_tokens", 8192).
		Param("chat_template_kwargs", thinkingOff).
		Retry(retry)

	// Pitches: the room writes full bibles for the shortlist (the Series array).
	pitches := p.Text("pitches").
		Capability("long_form").
		After(dev).
		PromptFS(PromptsFS, "pitches.md").
		OutputFormatJSON().
		Output("series.json").
		Param("temperature", 0.7).
		Param("max_tokens", 28672).
		Param("chat_template_kwargs", thinkingOff).
		Retry(retry)

	// Audience metrics: a data team projects scroll-stop / retention / share /
	// follow per pitch. Cool and realist.
	metrics := p.Text("audience_metrics").
		Capability("long_form").
		After(pitches).
		PromptFS(PromptsFS, "audience_metrics.md").
		OutputFormatJSON().
		Output("metrics.json").
		Param("temperature", 0.3).
		Param("max_tokens", 8192).
		Param("chat_template_kwargs", thinkingOff).
		Retry(retry)

	// Greenlight: the committee scores craft, folds in the metrics, and issues a
	// single greenlight number per series (the ranking sorts on it).
	p.Text("greenlight").
		Capability("long_form").
		After(pitches, metrics).
		PromptFS(PromptsFS, "greenlight.md").
		OutputFormatJSON().
		Output("scores.json").
		Param("temperature", 0.3).
		Param("max_tokens", 10240).
		Param("chat_template_kwargs", thinkingOff).
		Retry(retry)

	return p.Build()
}
