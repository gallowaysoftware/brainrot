// Command brainrot is a TikTok short-form content mill: breadth (ideate)
// brainstorms character+situation content seeds and scores them for short-video
// potential; depth (make) turns one seed into a vertical AI video.
//
//	brainrot ideate --niche "..."   Generate + score ideas, keep the best.
//	brainrot list                   Show kept ideas with scores.
//	brainrot make <idea-id>         Render a vertical video from one idea.
//	brainrot activate / doctor      Bring up / check the vibe stack.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/gallowaysoftware/vibe/vamp"

	"github.com/gallowaysoftware/brainrot/internal/idea"
	"github.com/gallowaysoftware/brainrot/internal/pipeline"
)

func main() {
	root := &cobra.Command{
		Use:           "brainrot",
		Short:         "TikTok short-form content mill (ideate -> score -> make).",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(ideateCommand(), listCommand(), makeCommand(), activateCommand(), doctorCommand())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "brainrot:", err)
		os.Exit(1)
	}
}

// ---- ideate (breadth + evaluate) ----

func ideateCommand() *cobra.Command {
	var (
		niche string
		count int
		keep  int
	)
	cmd := &cobra.Command{
		Use:   "ideate --niche <niche>",
		Short: "Generate content ideas for a niche, score them, keep the best.",
		Long: `ideate brainstorms <count> character+situation video ideas for a niche,
scores each for short-form potential (hook, shootability, punch, legibility,
loop, format-fit), keeps the top <keep>, and saves the full ranking. Each kept
idea is ready for ` + "`brainrot make`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if niche == "" {
				return fmt.Errorf("--niche is required")
			}
			return runIdeate(cmd, niche, count, keep)
		},
	}
	cmd.Flags().StringVar(&niche, "niche", "", "Content niche to brainstorm within.")
	cmd.Flags().IntVar(&count, "count", 20, "How many ideas to generate + score.")
	cmd.Flags().IntVar(&keep, "keep", 5, "How many top-scored ideas to keep.")
	return cmd
}

func runIdeate(cmd *cobra.Command, niche string, count, keep int) error {
	l, err := idea.Open()
	if err != nil {
		return err
	}
	genDir, err := os.MkdirTemp(l.Root, ".ideate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(genDir)

	fmt.Fprintf(cmd.OutOrStdout(), "niche: %s\ngenerating + scoring %d ideas...\n\n", niche, count)
	r, err := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
		return pipeline.BuildIdeate(pipeline.IdeateConfig{Niche: niche, Count: count})
	})
	if err != nil {
		return err
	}
	r.SetArgs([]string{"run", "--run-dir", genDir, "--no-cache"})
	if err := r.Execute(); err != nil {
		return fmt.Errorf("ideate: %w", err)
	}

	ideasRaw, err := os.ReadFile(filepath.Join(genDir, "ideas.json"))
	if err != nil {
		return fmt.Errorf("read ideas: %w", err)
	}
	ideas, err := idea.ParseIdeas(ideasRaw)
	if err != nil {
		return err
	}
	if scoresRaw, err := os.ReadFile(filepath.Join(genDir, "scores.json")); err == nil {
		if err := idea.ApplyScores(ideas, scoresRaw); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "note: scores not applied (%v); keeping unscored order\n", err)
		}
	}
	// Normalize voices on the way in so a bad voice can't bite at make time.
	for i := range ideas {
		ideas[i].Character.VoiceID = idea.NormalizeVoice(ideas[i].Character.VoiceID)
	}
	idea.SortByScore(ideas)

	stamp := time.Now().Format("2006-01-02T15-04-05")
	if err := l.SaveRanking(stamp, ideas); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "note: save ranking: %v\n", err)
	}

	if keep > len(ideas) {
		keep = len(ideas)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "kept top %d of %d:\n", keep, len(ideas))
	for i := 0; i < keep; i++ {
		id, err := l.SaveIdea(ideas[i])
		if err != nil {
			return err
		}
		total := 0
		if ideas[i].Score != nil {
			total = ideas[i].Score.Total
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  [%2d/60] %-28s %s\n", total, id, ideas[i].Hook)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nfull ranking: %s\nmake one with: brainrot make <id>\n",
		filepath.Join(l.IdeasDir(), "_rankings", stamp+".json"))
	return nil
}

// ---- list ----

func listCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List kept ideas with their scores.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := idea.Open()
			if err != nil {
				return err
			}
			ids, err := idea.ListIdeas(l)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no ideas yet — `brainrot ideate --niche \"...\"`")
				return nil
			}
			for _, id := range ids {
				it, err := idea.LoadIdea(l, id)
				if err != nil {
					continue
				}
				total := 0
				if it.Score != nil {
					total = it.Score.Total
				}
				vids := idea.NextVideo(l, id) - 1
				fmt.Fprintf(cmd.OutOrStdout(), "  [%2d/60] %-28s %d video(s)  %s\n", total, id, vids, it.Hook)
			}
			return nil
		},
	}
}

// ---- make (depth) ----

func makeCommand() *cobra.Command {
	var (
		shots     int
		narrator  string
		publishTo string
	)
	cmd := &cobra.Command{
		Use:   "make <idea-id>",
		Short: "Render a vertical TikTok video from a kept idea.",
		Long: `make turns one content idea into a vertical 1080x1920 video. Two phases so
the LLM is unloaded before the image model loads: phase 1 (LLM) writes the shot
list; phase 2 (ComfyUI + Kokoro) renders stills -> image-to-video -> voiceover ->
captioned MP4.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMake(cmd, args[0], shots, narrator, publishTo)
		},
	}
	cmd.Flags().IntVar(&shots, "shots", 7, "Number of shots in the video.")
	cmd.Flags().StringVar(&narrator, "narrator", "am_fenrir", "Default Kokoro narrator voice.")
	cmd.Flags().StringVar(&publishTo, "publish-to", "", "Directory to copy the finished final.mp4 into.")
	return cmd
}

func runMake(cmd *cobra.Command, id string, shots int, narrator, publishTo string) error {
	l, err := idea.Open()
	if err != nil {
		return err
	}
	if _, err := os.Stat(l.IdeaFile(id)); err != nil {
		return fmt.Errorf("idea %q not found at %s — run `brainrot list`", id, l.IdeaFile(id))
	}

	n := idea.NextVideo(l, id)
	videoDir := l.VideoDir(id, n)
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		return err
	}
	cfg := pipeline.SceneConfig{
		IdeaFile:      l.IdeaFile(id),
		Shots:         shots,
		NarratorVoice: narrator,
		ShotsFile:     filepath.Join(videoDir, "shots.json"),
	}

	fmt.Fprintf(cmd.OutOrStdout(), "idea: %s\nvideo: %03d\n", id, n)

	fmt.Fprintln(cmd.OutOrStdout(), "phase 1/2: writing shot list (LLM)...")
	scriptRoot, err := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
		return pipeline.BuildSceneScript(cfg)
	})
	if err != nil {
		return err
	}
	scriptRoot.SetArgs([]string{"run", "--run-dir", videoDir, "--no-cache"})
	if err := scriptRoot.Execute(); err != nil {
		return fmt.Errorf("video %d phase 1: %w", n, err)
	}
	if err := normalizeShotVoices(cfg.ShotsFile); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "  note: normalize shot voices: %v\n", err)
	}

	// Free the LLM before the image model loads — they can't co-reside on 32GB.
	freeActiveProfile(cmd)

	fmt.Fprintln(cmd.OutOrStdout(), "phase 2/2: rendering (image -> video -> voice -> assemble)...")
	renderRoot, err := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
		return pipeline.BuildSceneRender(cfg)
	})
	if err != nil {
		return err
	}
	renderRoot.SetArgs([]string{"run", "--run-dir", videoDir, "--no-cache"})
	if err := renderRoot.Execute(); err != nil {
		return fmt.Errorf("video %d phase 2: %w", n, err)
	}

	final := filepath.Join(videoDir, "final.mp4")
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", final)
	if publishTo != "" {
		if err := os.MkdirAll(publishTo, 0o755); err != nil {
			return err
		}
		dst := filepath.Join(publishTo, fmt.Sprintf("%s-%03d.mp4", id, n))
		if err := copyFile(final, dst); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "published: %s\n", dst)
	}
	return nil
}

// normalizeShotVoices rewrites shots.json in place, replacing any shot's
// voice_id that isn't a known Kokoro voice with the narrator fallback.
func normalizeShotVoices(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	changed := false
	for _, it := range doc.Items {
		if v, ok := it["voice_id"].(string); ok {
			if nv := idea.NormalizeVoice(v); nv != v {
				it["voice_id"] = nv
				changed = true
			}
		}
	}
	if !changed {
		return nil
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// freeActiveProfile shells `vibe stop` to unload the active LLM. Non-fatal.
func freeActiveProfile(cmd *cobra.Command) {
	if err := exec.Command("vibe", "stop").Run(); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "  note: `vibe stop` to free LLM VRAM failed (%v); continuing\n", err)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ---- activate / doctor ----

func activateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "activate",
		Short: "Bring up the vibe profile + services brainrot needs.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := vamp.BuildRoot(pipeline.BuildStub)
			if err != nil {
				return err
			}
			r.SetArgs([]string{"activate"})
			return r.Execute()
		},
	}
}

func doctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Read-only: what's running, what's missing.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := vamp.BuildRoot(pipeline.BuildStub)
			if err != nil {
				return err
			}
			r.SetArgs([]string{"doctor"})
			return r.Execute()
		},
	}
}
