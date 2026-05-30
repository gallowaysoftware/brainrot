package pipeline

import "github.com/gallowaysoftware/vibe/vamp"

// BuildStub declares brainrot's profile + services with no real work, so the
// `activate` / `doctor` commands can bring up / check the stack.
func BuildStub() (*vamp.Pipeline, error) {
	p := vamp.New("brainrot-stub").
		Describe("Declares brainrot's profile + services for activate/doctor.")
	p.RequireProfile("long_form")
	p.RequireService("comfyui", "http://127.0.0.1:8188",
		"ComfyUI — Qwen-Image stills + Wan2.2 image-to-video.",
		"vibe start comfyui")
	p.RequireService("kokoro-tts", "http://127.0.0.1:8880",
		"Kokoro-FastAPI TTS — character narration.",
		"vibe start tts_kokoro")
	p.Render("noop").Prompt("{}").Output("noop.json").OutputFormatJSON()
	return p.Build()
}
