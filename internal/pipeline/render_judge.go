package pipeline

import (
	"time"

	"github.com/gallowaysoftware/vibe/vamp"
)

// RenderJudgeConfig drives the phase-2 vision judge: given the finished renders of
// several finalist episodes (sampled frames + narration), a multimodal model looks
// at the ACTUAL rendered output and scores each — catching i2v distortion, garbled
// text, off-model goblins, and image/line mismatch that a script-only judge can't.
type RenderJudgeConfig struct {
	// FinalistsFile is a JSON object {"items":[{idx,title,frames,narration}]} (the
	// "items" key is what vamp's foreach unwraps), where "frames" is an absolute
	// dir of sampled PNG frames for that finalist.
	FinalistsFile string
}

// BuildRenderJudge constructs the vision judge: one multimodal pass per finalist
// (its sampled frames attached via ImageDir), scoring the rendered video. The CLI
// reads the per-finalist scores and ships the highest. Uses the "vision_vl"
// capability (Qwen3-VL-32B + mmproj), which vamp activates by evicting the active profile.
func BuildRenderJudge(cfg RenderJudgeConfig) (*vamp.Pipeline, error) {
	p := vamp.New("brainrot-render-judge").
		Describe("Vision judge: look at each finalist's rendered frames, score the actual video.")

	p.Input("finalists_file", vamp.Required(), vamp.WithDefault(cfg.FinalistsFile),
		vamp.Describe("Path to the finalists JSON (idx, title, frames dir, narration)."))

	// Judge model: Qwen3-VL-32B (vision_vl) — stronger image understanding +
	// OCR/gibberish detection than Gemma 3 did, and a stable llama.cpp vision path.
	p.RequireProfile("vision_vl")
	p.RequireGPUMemory("~24GB (Qwen3-VL-32B + mmproj)")
	p.CapabilityModel("vision_vl", vamp.ModelHint{
		MinParams: "32B", MinContext: 32768,
		SuggestedModel: "qwen3-vl-32b + mmproj",
		Capabilities:   []string{"text", "vision"},
	})

	finalists := p.Render("load_finalists").
		Prompt(`{{ readFile .inputs.finalists_file }}`).
		Output("finalists_loaded.json").
		OutputFormatJSON()

	p.Text("render_judge").
		Capability("vision_vl").
		After(finalists).
		Foreach(finalists, "f").
		PromptFS(PromptsFS, "render_judge.md").
		ImageDir("{{ .f.frames }}").
		OutputFormatJSON().
		Output("render_judge/{{ .f.idx }}.json").
		Param("temperature", 0.2).
		Param("max_tokens", 1024).
		Retry(&vamp.RetryPolicy{
			MaxAttempts:    5,
			InitialBackoff: 15 * time.Second,
			MaxBackoff:     120 * time.Second,
			RetryOn:        []string{"transient", "invalid_output"},
		})

	return p.Build()
}
