package pipeline

import (
	"fmt"
	"time"

	"github.com/gallowaysoftware/vibe/vamp"
)

// IdeateConfig drives the breadth+evaluate pipeline: generate Count content
// ideas for a niche, then score each for TikTok potential.
type IdeateConfig struct {
	Niche string
	Count int
}

var thinkingOff = map[string]any{"enable_thinking": false}

// BuildIdeate constructs the two-stage breadth pipeline:
//
//	generate_ideas (text) -> ideas.json   — Count content seeds for the niche
//	score_ideas    (text) -> scores.json  — per-idea TikTok-potential scores
//
// The CLI reads both outputs, merges scores onto ideas, ranks, and keeps the
// top K (deterministic, code-side — like worldsmith's outline rerank).
func BuildIdeate(cfg IdeateConfig) (*vamp.Pipeline, error) {
	if cfg.Count <= 0 {
		cfg.Count = 20
	}
	p := vamp.New("brainrot-ideate").
		Describe("Generate short-form video content ideas for a niche and score them for TikTok potential.")

	p.Input("niche", vamp.Required(), vamp.WithDefault(cfg.Niche),
		vamp.Describe("The content niche to brainstorm within."))
	p.Input("count", vamp.WithDefault(fmt.Sprintf("%d", cfg.Count)),
		vamp.Describe("How many ideas to generate + score."))

	p.RequireProfile("long_form")
	p.RequireGPUMemory("~30GB during generation")
	p.CapabilityModel("long_form", vamp.ModelHint{
		MinParams: "27B", MinContext: 131072,
		SuggestedModel: "qwen3.6-27b-mtp-q6_k",
	})

	gen := p.Text("generate_ideas").
		Capability("long_form").
		PromptFS(PromptsFS, "generate_ideas.md").
		OutputFormatJSON().
		Output("ideas.json").
		// Warm for diversity across the batch; JSON gate + retry keep it valid.
		Param("temperature", 0.95).
		Param("max_tokens", 24576).
		Param("chat_template_kwargs", thinkingOff).
		Retry(&vamp.RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 5 * time.Second,
			MaxBackoff:     30 * time.Second,
			RetryOn:        []string{"transient", "invalid_output"},
		})

	// Fresh-eyes judge: a separate cool-headed pass scores each idea as if it
	// were already produced, so the scorer isn't anchored on having written it.
	p.Text("score_ideas").
		Capability("long_form").
		After(gen).
		PromptFS(PromptsFS, "score_ideas.md").
		OutputFormatJSON().
		Output("scores.json").
		Param("temperature", 0.25).
		Param("max_tokens", 16384).
		Param("chat_template_kwargs", thinkingOff).
		Retry(&vamp.RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 5 * time.Second,
			MaxBackoff:     30 * time.Second,
			RetryOn:        []string{"transient", "invalid_output"},
		})

	return p.Build()
}
