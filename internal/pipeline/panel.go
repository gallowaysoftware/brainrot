package pipeline

import (
	"fmt"

	"github.com/gallowaysoftware/vibe/contentkit"
	"github.com/gallowaysoftware/vibe/vamp"
)

// PanelConfig drives the greenlight panel: given N candidate episode scripts,
// three independent execs (editor / producer / money) score them on their own
// axis, then a greenlight stage picks the single one to ship. This is the
// automated stand-in for human curation — generate many, let the panel choose.
type PanelConfig struct {
	// CandidatesFile is a JSON object {"candidates":[{index,title,logline,shots[...]}]}.
	CandidatesFile string
	// Count is how many candidates are in the file (for the prompt).
	Count int
}

// BuildPanel constructs the selection panel:
//
//	editor    (text) -> editor.json    — craft/comedy score per candidate
//	producer  (text) -> producer.json  — shootability/pacing/works-as-video score
//	money     (text) -> money.json     — scroll-stop/shareability/sellability score
//	greenlight(text) -> verdict.json   — {"winner":<index>,"rationale":...}
//
// The three execs run independently (parallel, each sees only the candidates) so
// their judgments don't collapse onto one another; greenlight synthesizes them.
func BuildPanel(cfg PanelConfig) (*vamp.Pipeline, error) {
	p := vamp.New("brainrot-panel").
		Describe("Greenlight panel: editor/producer/money score N candidates, pick one to ship.")

	p.Input("candidates_file", vamp.Required(), vamp.WithDefault(cfg.CandidatesFile),
		vamp.Describe("Path to the candidates JSON."))
	p.Input("count", vamp.WithDefault(fmt.Sprintf("%d", cfg.Count)),
		vamp.Describe("How many candidates are in the file."))

	p.RequireProfile("long_form")
	p.RequireGPUMemory("~30GB during generation")
	p.CapabilityModel("long_form", vamp.ModelHint{
		MinParams: "27B", MinContext: 131072,
		SuggestedModel: "qwen3.6-27b-mtp-q6_k",
	})

	editor := contentkit.LongFormText(p, "editor", 0.3, 6144).
		PromptFS(PromptsFS, "panel_editor.md").
		OutputFormatJSON().
		Output("editor.json")
	producer := contentkit.LongFormText(p, "producer", 0.3, 6144).
		PromptFS(PromptsFS, "panel_producer.md").
		OutputFormatJSON().
		Output("producer.json")
	money := contentkit.LongFormText(p, "money", 0.3, 6144).
		PromptFS(PromptsFS, "panel_money.md").
		OutputFormatJSON().
		Output("money.json")

	contentkit.LongFormText(p, "greenlight", 0.2, 4096).
		After(editor, producer, money).
		PromptFS(PromptsFS, "panel_pick.md").
		OutputFormatJSON().
		Output("verdict.json")

	return p.Build()
}
