package pipeline

import (
	"fmt"

	"github.com/gallowaysoftware/vibe/vamp"
)

// SceneConfig drives depth rendering: turn an episode's shot list into a
// vertical short. Phase 1 (the writers' room) lives in BuildEpisodeScript; this
// is phase 2, run after the ~26GB LLM is unloaded so the image model fits.
type SceneConfig struct {
	Shots         int
	NarratorVoice string
	KokoroURL     string
	// ShotsFile is the path to the phase-1 shots.json, read by phase 2.
	ShotsFile string
	// AnchorDir, when set, is the series' anchors/ directory. Shots tagged with
	// an "anchor" (the narrator's cold-open + meltdown — see cmd/brainrot
	// tagNarratorAnchors) then use the fixed image AnchorDir/<anchor>.png verbatim
	// instead of being generated, so a hand-made character portrait stays
	// pixel-identical across every episode. Empty = generate every shot (the
	// original behavior).
	AnchorDir string
}

const defaultKokoroURL = "http://127.0.0.1:8880"

// codexRobotVoice is an ffmpeg -af chain applied to the (clean, intelligible)
// Kokoro narration so Codex sounds like a malfunctioning AI rather than a smooth
// human: afftfilt phase-flatten = the classic monotone-metallic robot timbre,
// acrusher = digital/lo-fi grit, the band-pass = a comms/intercom voice, the
// phaser adds subtle synthetic movement. Kept above lowpass 6500 / bits 8 so the
// words stay clear (TikTok needs intelligible). Tunable per the voice research.
const codexRobotVoice = "afftfilt=real='hypot(re,im)*sin(0)':imag='hypot(re,im)*cos(0)':win_size=512:overlap=0.75," +
	"acrusher=bits=8:mode=log:samples=1,highpass=f=200,lowpass=f=6500," +
	"aphaser=type=t:speed=0.8:decay=0.4"

// sceneNegative is the mechanical backstop on every still: AI image models render
// text as gibberish and default human figures into "person" scenes, so we suppress
// both hard regardless of what the per-shot image_prompt says. (Goblins are not
// "human", so suppressing humans doesn't fight the cast.)
const sceneNegative = "text, words, letters, typography, caption, subtitle, sign, label, watermark, signature, logo, gibberish, UI, interface, human, person, people, man, woman, hand, hands, fingers, arm, holding, realistic human face, photograph, blurry, low quality, deformed, extra limbs, extra fingers"

// wanNegative is the Wan i2v animate negative prompt. The image model's negative
// suppresses hands/humans in the STILL, but Wan re-hallucinates motion and will
// add a finger/hand poking in unless told not to — so suppress them here too,
// alongside the anti-static/morph terms.
// wanNegative targets the BAD parts (hands/people, and the orange-debris/keys/fire
// that the meltdown kept erupting into) while leaving green energy/glitch MOTION
// alone — a too-broad "explosion/burst/particles" negative froze the orb dead. The
// still's clean negative is sceneNegative; Wan re-hallucinates on motion so it
// needs its own.
const wanNegative = "static, still, frozen, jittery, distorted, deformed, low quality, hand, hands, fingers, finger, arm, person, human, holding, body part, face, orange, amber, brown, fire, embers, smoke, dust, falling debris, keys, scattered objects"

// splitGeneratedTmpl emits {"items":[...]} containing only the shots WITHOUT an
// anchor tag — the ones phase 2 generates with Qwen. Anchored shots (the narrator
// portrait) are skipped here because the render animates their fixed anchor image
// directly (see animateInput). `index . "anchor"` is used instead of `.anchor`
// because vamp renders templates with missingkey=error, and most shots have no
// "anchor" key; `index` returns nil for a missing key rather than erroring.
// `not <nil-or-empty>` is true, so untagged shots all land here. Couples to the
// producer stage being named "load_shots" (both builders do).
const splitGeneratedTmpl = `{{- $items := index (parseJSON .stages.load_shots.output) "items" -}}
{"items":[{{ $first := true }}{{ range $items }}{{ if not (index . "anchor") }}{{ if $first }}{{ $first = false }}{{ else }},{{ end }}{{ toJSON . }}{{ end }}{{ end }}]}`

// stillStages wires phase-2 Qwen still generation and returns the scene_images
// stage. With anchorDir set, anchored shots are filtered OUT of generation (their
// look comes from the fixed anchor image, animated directly — see animateInput),
// so only the non-anchored shots cost a Qwen render. anchorDir == "" generates
// every shot (the original behavior). `shots` must be the "load_shots" render
// (splitGeneratedTmpl references it by name).
func stillStages(p *vamp.Pipeline, shots vamp.Ref, anchorDir string, freeAfter bool) vamp.Ref {
	from := shots
	if anchorDir != "" {
		from = p.Render("split_generated").
			After(shots).
			Prompt(splitGeneratedTmpl).
			Output("shots_generated.json").
			OutputFormatJSON()
	}
	gen := p.ComfyUI("scene_images").
		Capability("image_gen").
		After(from).
		Foreach(from, "shot").
		WorkflowFS(WorkflowsFS, "qwen_portrait.json").
		Parameter("4.text", "{{ .shot.image_prompt }}").
		Parameter("5.text", sceneNegative).
		Parameter("7.seed", "{{ .shot.idx }}")
	if freeAfter {
		gen = gen.FreeMemoryAfter()
	}
	return gen.Output("images/shot_{{ .shot.idx }}.png")
}

// animateInput is the Wan i2v start-image path template. With no anchors it's the
// Qwen-generated still. With anchors, a shot tagged with an "anchor" animates the
// fixed anchorDir/<anchor>.png directly (Wan's InputImage accepts absolute paths),
// so the narrator's portrait is the real hand-made image — never a re-hallucinated
// drift — while every other shot still animates its generated still. `index .shot
// "anchor"` tolerates the missing key (vamp uses missingkey=error).
func animateInput(anchorDir string) string {
	if anchorDir == "" {
		return "images/shot_{{ .shot.idx }}.png"
	}
	return `{{ $a := index .shot "anchor" }}{{ if $a }}` + anchorDir + `/{{ $a }}.png{{ else }}images/shot_{{ .shot.idx }}.png{{ end }}`
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

	// Preview the shots that actually vary: Qwen-generate the non-anchored stills.
	// Anchored shots (the fixed narrator portrait) aren't generated — they need no
	// composition iteration. freeAfter=true releases Qwen when stills are done.
	stillStages(p, shots, cfg.AnchorDir, true)

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

	// Qwen-generate stills for the non-anchored shots. Anchored shots (the narrator
	// portrait) are skipped here — animate reads their fixed anchor image directly.
	sceneImages := stillStages(p, shots, cfg.AnchorDir, false)

	clips := p.ComfyUI("animate").
		Capability("video_gen").
		After(sceneImages).
		Foreach(shots, "shot").
		WorkflowFS(WorkflowsFS, "wan_i2v.json").
		Parameter("4.text", "{{ .shot.motion }}, cinematic, smooth motion").
		Parameter("5.text", wanNegative).
		Parameter("8.seed", "{{ .shot.idx }}").
		InputImage("6.image", animateInput(cfg.AnchorDir)).
		FreeMemoryAfter().
		Output("clips/shot_{{ .shot.idx }}.mp4")

	voices := p.Audio("voiceover").
		Capability("tts").
		After(shots).
		Foreach(shots, "shot").
		Engine(vamp.AudioEngineKokoro).
		EngineURL(cfg.KokoroURL).
		Voice(fmt.Sprintf(`{{ or .shot.voice_id %q }}`, cfg.NarratorVoice)).
		TextTemplate(`{{ ttsNormalize (or .shot.tts_text .shot.narration) "" }}`).
		Effect(codexRobotVoice).
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
		StretchToAudio(true). // time-stretch each clip to its voiceover length, no last-frame stall
		Output("final.mp4")

	return p.Build()
}
