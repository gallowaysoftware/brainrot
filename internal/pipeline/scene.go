package pipeline

import (
	"fmt"
	"time"

	"github.com/gallowaysoftware/vibe/vamp"
)

// SceneConfig drives depth: turn one content idea into a vertical TikTok. Run
// in two pipeline phases (BuildSceneScript then BuildSceneRender) so the ~26GB
// LLM is unloaded before the ~20GB image model loads.
type SceneConfig struct {
	// IdeaFile is the path to the kept idea.json (read by the script phase).
	IdeaFile      string
	Shots         int
	NarratorVoice string
	KokoroURL     string
	// ShotsFile is the path to the phase-1 shots.json, read by phase 2.
	ShotsFile string
}

const defaultKokoroURL = "http://127.0.0.1:8880"

// BuildSceneScript is phase 1 (LLM only): turn the idea into a shot list.
// Writes shots.json (a {"items":[...]} array, each shot tagged with a stable
// idx) for phase 2.
func BuildSceneScript(cfg SceneConfig) (*vamp.Pipeline, error) {
	if cfg.Shots <= 0 {
		cfg.Shots = 7
	}
	p := vamp.New("brainrot-scene-script").
		Describe("Turn a content idea into a punchy short-form shot list.")

	p.Input("idea_file", vamp.Required(), vamp.WithDefault(cfg.IdeaFile),
		vamp.Describe("Path to the idea.json content seed."))
	p.Input("shots", vamp.WithDefault(fmt.Sprintf("%d", cfg.Shots)),
		vamp.Describe("Number of shots in the video."))

	p.RequireProfile("long_form")
	p.RequireGPUMemory("~30GB during generation")
	p.CapabilityModel("long_form", vamp.ModelHint{
		MinParams: "27B", MinContext: 131072,
		SuggestedModel: "qwen3.6-27b-mtp-q6_k",
	})

	outline := p.Text("scene_outline").
		Capability("long_form").
		PromptFS(PromptsFS, "scene_outline.md").
		OutputFormatJSON().
		Output("scene.json").
		Param("temperature", 0.8).
		Param("max_tokens", 8192).
		Param("chat_template_kwargs", thinkingOff).
		Retry(&vamp.RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 5 * time.Second,
			MaxBackoff:     30 * time.Second,
			RetryOn:        []string{"transient", "invalid_output"},
		})

	p.Render("enumerate_shots").
		After(outline).
		Prompt(`{"items": [
{{- $shots := index (parseJSON .stages.scene_outline.output) "shots" -}}
{{- range $i, $s := $shots -}}
{{- if $i }},{{ end }}
{"idx": {{ $i }}, "image_prompt": {{ toJSON (index $s "image_prompt") }}, "motion": {{ toJSON (index $s "motion") }}, "narration": {{ toJSON (index $s "narration") }}, "speaker": {{ toJSON (index $s "speaker") }}, "voice_id": {{ toJSON (index $s "voice_id") }}}
{{- end }}
] }`).
		Output("shots.json").
		OutputFormatJSON()

	return p.Build()
}

// BuildScenePreview is the cheap iteration pipeline: generate only the per-shot
// stills (Qwen-Image), skipping Wan i2v / voiceover / assembly. ~30s/shot vs
// ~2min/shot, so content quality (composition, character consistency, image vs
// narration) can be judged fast before committing to a full render.
func BuildScenePreview(cfg SceneConfig) (*vamp.Pipeline, error) {
	p := vamp.New("brainrot-scene-preview").
		Describe("Stills-only preview of a scene's shots (no animation/voice).")

	p.Input("shots_file", vamp.Required(), vamp.WithDefault(cfg.ShotsFile),
		vamp.Describe("Path to phase-1 shots.json."))
	p.RequireService("comfyui", "http://127.0.0.1:8188",
		"ComfyUI — Qwen-Image stills.", "vibe start comfyui")
	p.RequireGPUMemory("~20GB (Qwen-Image)")

	shots := p.Render("load_shots").
		Prompt(`{{ readFile .inputs.shots_file }}`).
		Output("shots_loaded.json").
		OutputFormatJSON()

	p.ComfyUI("scene_images").
		Capability("image_gen").
		After(shots).
		Foreach(shots, "shot").
		WorkflowFS(WorkflowsFS, "qwen_portrait.json").
		Parameter("4.text", "{{ .shot.image_prompt }}").
		Parameter("7.seed", "{{ .shot.idx }}").
		FreeMemoryAfter().
		Output("images/shot_{{ .shot.idx }}.png")

	return p.Build()
}

// BuildSceneRender is phase 2 (ComfyUI + TTS + assembly, no LLM): per shot
// generate a still (Qwen-Image), animate it (Wan i2v), voice the narration
// (Kokoro), then assemble a vertical MP4 with burned captions.
func BuildSceneRender(cfg SceneConfig) (*vamp.Pipeline, error) {
	if cfg.NarratorVoice == "" {
		cfg.NarratorVoice = "am_fenrir"
	}
	if cfg.KokoroURL == "" {
		cfg.KokoroURL = defaultKokoroURL
	}
	p := vamp.New("brainrot-scene-render").
		Describe("Render shots into a vertical short: image -> i2v -> voice -> assemble.")

	p.Input("shots_file", vamp.Required(), vamp.WithDefault(cfg.ShotsFile),
		vamp.Describe("Path to phase-1 shots.json."))

	p.RequireService("comfyui", "http://127.0.0.1:8188",
		"ComfyUI — Qwen-Image stills + Wan2.2 image-to-video.",
		"vibe start comfyui")
	p.RequireService("kokoro-tts", cfg.KokoroURL,
		"Kokoro-FastAPI TTS — character narration.",
		"vibe start tts_kokoro")
	p.RequireGPUMemory("~20GB (Qwen-Image) then ~12GB (Wan i2v); run with the LLM unloaded")
	p.RequireDiskSpace("~100MB per video (stills + clips + final MP4)")

	shots := p.Render("load_shots").
		Prompt(`{{ readFile .inputs.shots_file }}`).
		Output("shots_loaded.json").
		OutputFormatJSON()

	images := p.ComfyUI("scene_images").
		Capability("image_gen").
		After(shots).
		Foreach(shots, "shot").
		WorkflowFS(WorkflowsFS, "qwen_portrait.json").
		Parameter("4.text", "{{ .shot.image_prompt }}").
		Parameter("7.seed", "{{ .shot.idx }}").
		Output("images/shot_{{ .shot.idx }}.png")

	clips := p.ComfyUI("animate").
		Capability("video_gen").
		After(images).
		Foreach(shots, "shot").
		WorkflowFS(WorkflowsFS, "wan_i2v.json").
		Parameter("4.text", "{{ .shot.motion }}, cinematic, smooth motion").
		Parameter("8.seed", "{{ .shot.idx }}").
		InputImage("6.image", "images/shot_{{ .shot.idx }}.png").
		FreeMemoryAfter().
		Output("clips/shot_{{ .shot.idx }}.mp4")

	voices := p.Audio("voiceover").
		Capability("tts").
		After(shots).
		Foreach(shots, "shot").
		Engine(vamp.AudioEngineKokoro).
		EngineURL(cfg.KokoroURL).
		Voice(fmt.Sprintf(`{{ or .shot.voice_id %q }}`, cfg.NarratorVoice)).
		TextTemplate(`{{ ttsNormalize .shot.narration "" }}`).
		Output("audio/shot_{{ .shot.idx }}.wav")

	assembly := p.Render("assembly_script").
		After(shots).
		Prompt(`{{- $shots := index (parseJSON .stages.load_shots.output) "items" -}}
{"width": 1080, "height": 1920, "fps": 30, "shots": [
{{- range $i, $s := $shots -}}
{{- if $i }},{{ end }}
{"video": "clips/shot_{{ index $s "idx" }}.mp4", "audio": "audio/shot_{{ index $s "idx" }}.wav", "caption": {{ toJSON (index $s "narration") }}}
{{- end }}
]}`).
		Output("assembly.json").
		OutputFormatJSON()

	p.Short("assemble").
		After(clips, voices, assembly).
		ScriptFile("assembly.json").
		Size(1080, 1920).
		FPS(30).
		Output("final.mp4")

	return p.Build()
}
