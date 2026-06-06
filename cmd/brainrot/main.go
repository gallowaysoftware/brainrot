// Command brainrot is a serialized short-form content mill. Breadth (ideate)
// brainstorms SERIES concepts — a recurring cast plus an overarching arc broken
// into episode beats — and scores them for bingeable short-video potential.
// Depth (make) writes one EPISODE of a series as a multi-character scene (a
// two-pass writers' room: draft + punch-up) and renders it into a vertical AI
// video that both stands alone and advances the larger story.
//
//	brainrot ideate --niche "..."        Generate + score series, keep the best.
//	brainrot list                        Show kept series with scores + progress.
//	brainrot make <series-id> [--episode N]  Write + render an episode.
//	brainrot activate / doctor           Bring up / check the vibe stack.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gallowaysoftware/vibe/contentkit"
	"github.com/gallowaysoftware/vibe/vamp"

	"github.com/gallowaysoftware/brainrot/internal/pipeline"
	"github.com/gallowaysoftware/brainrot/internal/series"
)

func main() {
	root := &cobra.Command{
		Use:           "brainrot",
		Short:         "Serialized short-form content mill (ideate -> score -> make).",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(ideateCommand(), listCommand(), planCommand(), makeCommand(), renderJudgeCommand(), activateCommand(), doctorCommand())
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
		Short: "Generate serialized series concepts for a niche, score them, keep the best.",
		Long: `ideate runs a studio development funnel for a niche: a writers' room of four
distinct voices fans out a concept pool, an adversarial producers' table
(champion / skeptic / showrunner) argues it down to <count> shortlist, the room
writes full series bibles, an audience team projects metrics (scroll-stop,
retention, shareability, follow intent), and a greenlight committee scores craft
+ audience into a single greenlight number. The top <keep> are saved, ready for ` + "`brainrot make`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if niche == "" {
				return fmt.Errorf("--niche is required")
			}
			return runIdeate(cmd, niche, count, keep)
		},
	}
	cmd.Flags().StringVar(&niche, "niche", "", "Content niche to develop series within.")
	cmd.Flags().IntVar(&count, "count", 6, "How many shortlisted concepts to develop into full pitches.")
	cmd.Flags().IntVar(&keep, "keep", 4, "How many top-greenlit series to keep.")
	return cmd
}

func runIdeate(cmd *cobra.Command, niche string, count, keep int) error {
	l, err := series.Open()
	if err != nil {
		return err
	}
	// Keep the full studio trace (pool.json, development.json, series.json,
	// metrics.json, scores.json) so the writers'-room pool and the producers'
	// argument are inspectable after the run, not deleted.
	stamp := time.Now().Format("2006-01-02T15-04-05")
	genDir := filepath.Join(l.Root, "series", "_dev", stamp)
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "niche: %s\ndeveloping %d series through the studio funnel...\n\n", niche, count)
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

	raw, err := os.ReadFile(filepath.Join(genDir, "series.json"))
	if err != nil {
		return fmt.Errorf("read series: %w", err)
	}
	list, err := series.ParseSeriesList(raw)
	if err != nil {
		return err
	}
	if scoresRaw, err := os.ReadFile(filepath.Join(genDir, "scores.json")); err == nil {
		if err := series.ApplyScores(list, scoresRaw); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "note: scores not applied (%v); keeping unscored order\n", err)
		}
	}
	// Normalize cast voices on the way in so a bad voice can't bite at make time.
	for i := range list {
		for j := range list[i].Cast {
			list[i].Cast[j].VoiceID = series.NormalizeVoice(list[i].Cast[j].VoiceID)
		}
	}
	series.SortByScore(list)

	if err := l.SaveRanking(stamp, list); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "note: save ranking: %v\n", err)
	}

	if keep > len(list) {
		keep = len(list)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "kept top %d of %d (greenlight | craft/60 | retention%%):\n", keep, len(list))
	for i := 0; i < keep; i++ {
		id, err := l.SaveSeries(list[i])
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s  %-26s %s\n", scoreSummary(list[i].Score), id, list[i].Logline)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nfull ranking: %s\nstudio trace (pool, producer debate, metrics): %s\nwrite an episode with: brainrot make <id>\n",
		filepath.Join(l.Root, "series", "_rankings", stamp+".json"), genDir)
	return nil
}

// scoreSummary renders the headline studio numbers as a fixed-width column.
func scoreSummary(s *series.Score) string {
	if s == nil {
		return "[  --  ]"
	}
	return fmt.Sprintf("[GL %3d | %2d/60 | ret %2d%%]", s.Greenlight, s.Total, s.Retention)
}

// ---- list ----

func listCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List kept series with their scores and episode progress.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := series.Open()
			if err != nil {
				return err
			}
			ids, err := series.ListSeries(l)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no series yet — `brainrot ideate --niche \"...\"`")
				return nil
			}
			for _, id := range ids {
				s, err := series.LoadSeries(l, id)
				if err != nil {
					continue
				}
				done := series.NextEpisode(l, id) - 1
				fmt.Fprintf(cmd.OutOrStdout(), "  %s  %-26s %d/%d eps  %s\n",
					scoreSummary(s.Score), id, done, len(s.Episodes), s.Logline)
			}
			return nil
		},
	}
}

// ---- plan (mine a canon seed into episode beats) ----

func planCommand() *cobra.Command {
	var (
		count  int
		apply  bool
		review bool
	)
	cmd := &cobra.Command{
		Use:   "plan <series-id>",
		Short: "Mine a series' canon (lore.md + source) into a fresh, ranked slate of episode beats.",
		Long: `plan runs an episode-MINING funnel over a series that has a lore pack: four
miner personas fan out candidate episode "veins" anchored to verbatim canon, an
adversarial producers' table kills the unshootable / redundant / jokeless, a
showrunner writes the survivors into full episode beats in the show's voice, and a
committee scores them. The slate is staged for review; --apply appends it to the
bible, --review curates it one at a time. Mining stays FRESH — it won't repeat
episodes already in the bible.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlan(cmd, args[0], count, apply, review)
		},
	}
	cmd.Flags().IntVar(&count, "count", 8, "How many episode beats to develop into the slate.")
	cmd.Flags().BoolVar(&apply, "apply", false, "Append the whole generated slate to the bible (non-destructive).")
	cmd.Flags().BoolVar(&review, "review", false, "Curate the slate interactively (accept/reject each beat).")
	return cmd
}

// beatScore is one episode beat paired with its committee total, for ranking.
type beatScore struct {
	beat    series.EpisodeBeat
	total   int
	verdict string
}

func runPlan(cmd *cobra.Command, id string, count int, apply, review bool) error {
	l, err := series.Open()
	if err != nil {
		return err
	}
	s, err := series.LoadSeries(l, id)
	if err != nil {
		return fmt.Errorf("series %q not found — run `brainrot list`", id)
	}
	if !l.HasLore(id) {
		return fmt.Errorf("series %q has no lore.md — plan mines a canon pack; add one beside series.json", id)
	}

	stamp := time.Now().Format("2006-01-02T15-04-05")
	stageDir := filepath.Join(l.SeriesDir(id), "_planned", stamp)
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return err
	}

	// Write the already-covered canon so mining stays fresh (anti-repeat).
	existingFile := filepath.Join(stageDir, "existing.md")
	if err := os.WriteFile(existingFile, []byte(coverageDigest(s)), 0o644); err != nil {
		return err
	}
	sourceFile := ""
	if src := filepath.Join(l.SeriesDir(id), "source.txt"); fileExists(src) {
		sourceFile = src
	}

	fmt.Fprintf(cmd.OutOrStdout(), "series: %s (%d existing episodes)\nmining canon into %d new episode beats...\n\n",
		id, len(s.Episodes), count)
	r, err := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
		return pipeline.BuildPlan(pipeline.PlanConfig{
			SeriesFile:   l.SeriesFile(id),
			LoreFile:     l.LoreFile(id),
			SourceFile:   sourceFile,
			ExistingFile: existingFile,
			Count:        count,
		})
	})
	if err != nil {
		return err
	}
	r.SetArgs([]string{"run", "--run-dir", stageDir, "--no-cache"})
	if err := r.Execute(); err != nil {
		return fmt.Errorf("plan: %w", err)
	}

	beatsRaw, err := os.ReadFile(filepath.Join(stageDir, "beats.json"))
	if err != nil {
		return fmt.Errorf("read beats: %w", err)
	}
	beats, err := series.ParseEpisodeBeats(beatsRaw)
	if err != nil {
		return err
	}
	scoresRaw, _ := os.ReadFile(filepath.Join(stageDir, "scores.json"))
	ranked := rankBeats(beats, scoresRaw)
	// Mechanical anti-repeat guard: drop any beat whose title collides with an
	// existing episode (models drift back to prominent canon despite the prompt's
	// "already covered" list — counting is code's job, not the LLM's). Done after
	// ranking so the index-aligned scores stay matched to their beats.
	ranked, dropped := dropCovered(ranked, s.Episodes)
	if len(dropped) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "dropped %d beat(s) that duplicate existing episodes: %s\n\n",
			len(dropped), strings.Join(dropped, ", "))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "proposed slate (%d beats), staged at %s:\n", len(ranked), stageDir)
	for i, b := range ranked {
		fmt.Fprintf(cmd.OutOrStdout(), "  %2d. [%2d/50] %-40s %s\n", i+1, b.total, truncate(b.beat.Title, 40), b.verdict)
	}

	switch {
	case review:
		items := make([]contentkit.ReviewItem, len(ranked))
		for i, b := range ranked {
			items[i] = contentkit.ReviewItem{
				ID:    fmt.Sprintf("%02d", i+1),
				Title: fmt.Sprintf("%s  [%d/50]", b.beat.Title, b.total),
				Body:  fmt.Sprintf("BEAT: %s\n\nBUTTON: %s\n\nLORE: %s", b.beat.Beat, b.beat.Button, b.beat.Lore),
				Stamp: stamp,
			}
		}
		res, err := contentkit.ReviewLoop(cmd.InOrStdin(), cmd.OutOrStdout(), items, contentkit.ReviewActions{
			Accept: func(it contentkit.ReviewItem) error {
				idx := indexOfID(ranked, it.ID)
				if idx < 0 {
					return fmt.Errorf("no staged beat for review id %q", it.ID)
				}
				_, e := series.AppendEpisodes(l, id, []series.EpisodeBeat{ranked[idx].beat})
				return e
			},
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\nreviewed: %d accepted, %d rejected. bible now has episodes appended.\n", res.Accepted, res.Discarded)
	case apply:
		toAdd := make([]series.EpisodeBeat, len(ranked))
		for i, b := range ranked {
			toAdd[i] = b.beat
		}
		total, err := series.AppendEpisodes(l, id, toAdd)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\napplied: appended %d beats; bible now has %d episodes.\n", len(toAdd), total)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "\nstaged only. append all with `brainrot plan %s --apply`, or curate with `--review`.\n", id)
	}
	return nil
}

// coverageDigest renders the already-shipped/planned episodes (title + lore) so
// the miner knows what NOT to repeat.
func coverageDigest(s series.Series) string {
	if len(s.Episodes) == 0 {
		return "(no episodes yet — all canon is fresh territory)\n"
	}
	var b strings.Builder
	for i, e := range s.Episodes {
		fmt.Fprintf(&b, "%d. %s — %s\n", i+1, e.Title, e.Beat)
		if e.Lore != "" {
			fmt.Fprintf(&b, "   uses: %s\n", e.Lore)
		}
	}
	return b.String()
}

// rankBeats pairs each beat with its committee total (from scores.json, parsed
// index-aligned) and sorts by total desc. Missing/short scores keep input order.
func rankBeats(beats []series.EpisodeBeat, scoresRaw []byte) []beatScore {
	out := make([]beatScore, len(beats))
	for i, b := range beats {
		out[i] = beatScore{beat: b}
	}
	if len(scoresRaw) > 0 {
		var doc struct {
			Scores []struct {
				Total   int    `json:"total"`
				Verdict string `json:"verdict"`
			} `json:"scores"`
		}
		if err := json.Unmarshal(scoresRaw, &doc); err == nil {
			for i := range out {
				if i < len(doc.Scores) {
					out[i].total = doc.Scores[i].Total
					out[i].verdict = doc.Scores[i].Verdict
				}
			}
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].total > out[b].total })
	return out
}

// dropCovered removes ranked beats whose title matches an existing episode title
// (normalized), returning the survivors and the dropped titles.
func dropCovered(ranked []beatScore, existing []series.EpisodeBeat) ([]beatScore, []string) {
	covered := make(map[string]bool, len(existing))
	for _, e := range existing {
		covered[normalizeTitle(e.Title)] = true
	}
	var kept []beatScore
	var dropped []string
	for _, b := range ranked {
		if covered[normalizeTitle(b.beat.Title)] {
			dropped = append(dropped, b.beat.Title)
			continue
		}
		kept = append(kept, b)
	}
	return kept, dropped
}

// normalizeTitle lowercases and strips non-alphanumerics so "The Wax Apple!" and
// "the wax apple" collide.
func normalizeTitle(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// indexOfID maps a ReviewItem id back to its index in ranked, or -1 if no beat
// matches — callers must not silently fall back to ranked[0] (the wrong beat).
func indexOfID(ranked []beatScore, id string) int {
	for i := range ranked {
		if fmt.Sprintf("%02d", i+1) == id {
			return i
		}
	}
	return -1
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ---- make (depth) ----

func makeCommand() *cobra.Command {
	var (
		episode    int
		shots      int
		candidates int
		finalists  int
		narrator   string
		publishTo  string
		preview    bool
		scriptOnly bool
		renderOnly bool
	)
	cmd := &cobra.Command{
		Use:   "make <series-id>",
		Short: "Write + render one episode of a series as a vertical video.",
		Long: `make writes one episode of a series as a multi-character scene and renders it
into a vertical 1080x1920 video. Two phases so the LLM is unloaded before the
image model loads: phase 1 (LLM) drafts the shot list and punches it up; phase 2
(ComfyUI + Kokoro) renders stills -> image-to-video -> voiceover -> captioned MP4.

With no --episode, the next unrendered episode is written.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if scriptOnly && renderOnly {
				return fmt.Errorf("--script-only and --render-only are mutually exclusive")
			}
			return runMake(cmd, args[0], episode, shots, candidates, finalists, narrator, publishTo, preview, scriptOnly, renderOnly)
		},
	}
	cmd.Flags().IntVar(&episode, "episode", 0, "Which episode to write (1-based); default = next unrendered.")
	cmd.Flags().IntVar(&shots, "shots", 7, "Number of shots in the episode.")
	cmd.Flags().IntVar(&candidates, "candidates", 1, "Generate N independent candidate scripts and let an editor/producer/money panel pick the one to render.")
	cmd.Flags().IntVar(&finalists, "finalists", 1, "Two-stage funnel: render the top K panel finalists and have a vision judge pick the best actual render. Requires --candidates > K.")
	cmd.Flags().StringVar(&narrator, "narrator", series.NarratorVoice, "Default Kokoro narrator voice.")
	cmd.Flags().StringVar(&publishTo, "publish-to", "", "Directory to copy the finished final.mp4 into.")
	cmd.Flags().BoolVar(&preview, "preview", false, "Stills-only: write the shot list + per-shot images, skip animation/voice/assembly (fast iteration).")
	cmd.Flags().BoolVar(&scriptOnly, "script-only", false, "Phase 1 only: write the episode script (draft + punch-up), no render. Leaves the LLM loaded for fast writing iteration.")
	cmd.Flags().BoolVar(&renderOnly, "render-only", false, "Phase 2 only: render the existing episode shots.json (skip writing). Use after --script-only.")
	return cmd
}

func runMake(cmd *cobra.Command, id string, episode, shots, candidates, finalists int, narrator, publishTo string, preview, scriptOnly, renderOnly bool) error {
	l, err := series.Open()
	if err != nil {
		return err
	}
	s, err := series.LoadSeries(l, id)
	if err != nil {
		return fmt.Errorf("series %q not found at %s — run `brainrot list`", id, l.SeriesFile(id))
	}

	// Per-series shot count wins over the flag default (lets a series tune its
	// own length, e.g. a tight TikTok monologue at 5 shots), but an explicit
	// --shots always wins.
	if !cmd.Flags().Changed("shots") && s.Shots > 0 {
		shots = s.Shots
	}

	n := episode
	if n <= 0 {
		n = series.NextEpisode(l, id)
	}
	if n > len(s.Episodes) {
		return fmt.Errorf("series %q has %d episodes; can't write episode %d", id, len(s.Episodes), n)
	}
	epDir := l.EpisodeDir(id, n)
	if err := os.MkdirAll(epDir, 0o755); err != nil {
		return err
	}
	// A bad --narrator (typo) would 400 the TTS mid-render; fold it to the known
	// fallback up front, the same guarantee normalizeShots applies to per-shot voices.
	cfg := pipeline.SceneConfig{
		Shots:         shots,
		NarratorVoice: series.NormalizeVoice(narrator),
		ShotsFile:     filepath.Join(epDir, "shots.json"),
	}
	// If this series ships a fixed anchor image for its narrator (a monologue's
	// first cast member), point the render at the anchors/ dir so the narrator's
	// Codex shots use the hand-made portrait verbatim instead of being generated.
	if s.EpisodeFormat() == series.FormatMonologue && len(s.Cast) > 0 {
		if l.HasAnchor(id, series.Slugify(s.Cast[0].Name)) {
			cfg.AnchorDir = l.AnchorDir(id)
		}
	}

	beatTitle := s.Episodes[n-1].Title
	fmt.Fprintf(cmd.OutOrStdout(), "series: %s\nepisode %03d: %s\n", id, n, beatTitle)

	// Two-stage funnel: gen candidates -> script panel -> render the top `finalists`
	// -> vision judge the actual renders -> ship the best. Self-contained (does its
	// own rendering + GPU stack management), so it returns before the normal flow.
	if finalists > 1 && !renderOnly && !scriptOnly && !preview {
		if candidates <= finalists {
			candidates = finalists * 2
		}
		loreFile := ""
		if l.HasLore(id) {
			loreFile = l.LoreFile(id)
		}
		epCfg := pipeline.EpisodeConfig{
			SeriesFile: l.SeriesFile(id), LoreFile: loreFile, Format: s.EpisodeFormat(), Episode: n, Shots: shots,
		}
		if err := generateRenderSelect(cmd, epCfg, cfg, s, candidates, finalists, epDir); err != nil {
			return err
		}
		final := filepath.Join(epDir, "final.mp4")
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", final)
		if publishTo != "" {
			if err := os.MkdirAll(publishTo, 0o755); err != nil {
				return err
			}
			if err := copyFile(final, filepath.Join(publishTo, fmt.Sprintf("%s-%03d.mp4", id, n))); err != nil {
				return err
			}
		}
		return nil
	}

	if renderOnly {
		// Phase 2 only: render the script already on disk (e.g. after --script-only
		// + a manual tweak). The writing is left exactly as-is.
		if _, err := os.Stat(cfg.ShotsFile); err != nil {
			return fmt.Errorf("no shots.json for episode %d — write it first (`brainrot make %s --episode %d --script-only`)", n, id, n)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "render-only: using existing shots.json")
	} else {
		// Pass the lore pack (lore.md) when the series has one, and the episode
		// format, so a lore-heavy source stays faithful and a monologue show isn't
		// forced into the multi-character scene mold.
		loreFile := ""
		if l.HasLore(id) {
			loreFile = l.LoreFile(id)
		}
		epCfg := pipeline.EpisodeConfig{
			SeriesFile: l.SeriesFile(id),
			LoreFile:   loreFile,
			Format:     s.EpisodeFormat(),
			Episode:    n,
			Shots:      shots,
		}

		if candidates > 1 {
			// Generate-many-and-greenlight: write N independent candidate scripts,
			// then an editor/producer/money panel picks the one to render. Its
			// shots.json lands at cfg.ShotsFile.
			fmt.Fprintf(cmd.OutOrStdout(), "phase 1/2: writing %d candidate scripts + greenlight panel (LLM)...\n", candidates)
			if err := generateWithPanel(cmd, epCfg, candidates, epDir, cfg.ShotsFile); err != nil {
				return err
			}
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "phase 1/2: writing episode (draft + punch-up, LLM)...")
			scriptRoot, err := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
				return pipeline.BuildEpisodeScript(epCfg)
			})
			if err != nil {
				return err
			}
			scriptRoot.SetArgs([]string{"run", "--run-dir", epDir, "--no-cache"})
			if err := scriptRoot.Execute(); err != nil {
				return fmt.Errorf("episode %d phase 1: %w", n, err)
			}
			if err := normalizeShots(cfg.ShotsFile); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "  note: normalize shot voices: %v\n", err)
			}
		}
	}

	if scriptOnly {
		// Writing-iteration mode: stop after the script, leave the LLM loaded so
		// the next `make --script-only` is fast. Surface the draft + final script.
		fmt.Fprintf(cmd.OutOrStdout(), "\nscript (final): %s\nstages:         %s\n",
			filepath.Join(epDir, "shots.json"),
			"bakeoff.json -> critique.json -> punched.json -> recheck.json -> polished.json -> tightened.json -> shots.json")
		return nil
	}

	// Free the LLM before the image model loads — they can't co-reside on 32GB.
	freeActiveProfile(cmd)

	// Tag the narrator's cold-open + meltdown shots so phase 2 swaps in the fixed
	// anchor portrait. Done here (deterministic, by position) rather than via a
	// model-emitted field that the writers'-room rewrite chain would drop. No-op
	// unless the series is a monologue with an anchors/<narrator>.png on disk.
	if err := tagNarratorAnchors(cfg.ShotsFile, s, cfg.AnchorDir); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "  note: tag narrator anchors: %v\n", err)
	}

	if preview {
		fmt.Fprintln(cmd.OutOrStdout(), "phase 2/2 (preview): generating stills only...")
		prevRoot, err := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
			return pipeline.BuildScenePreview(cfg)
		})
		if err != nil {
			return err
		}
		prevRoot.SetArgs([]string{"run", "--run-dir", epDir, "--no-cache"})
		if err := prevRoot.Execute(); err != nil {
			return fmt.Errorf("episode %d preview: %w", n, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\npreview stills: %s\n", filepath.Join(epDir, "images"))
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "phase 2/2: rendering (image -> video -> voice -> assemble)...")
	renderRoot, err := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
		return pipeline.BuildSceneRender(cfg)
	})
	if err != nil {
		return err
	}
	renderRoot.SetArgs([]string{"run", "--run-dir", epDir, "--no-cache"})
	if err := renderRoot.Execute(); err != nil {
		return fmt.Errorf("episode %d phase 2: %w", n, err)
	}

	final := filepath.Join(epDir, "final.mp4")
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

// bracketRE strips [stage directions] / [SFX] / [GLITCH] cues. leadLabelRE strips a
// leading ALL-CAPS UI/alert label like "SYSTEM ALERT:". Both leak into TTS + burned
// captions if left in narration, so we remove them mechanically — prompt rules alone
// don't hold (any on-screen text belongs in the image_prompt, not the spoken line).
var (
	bracketRE   = regexp.MustCompile(`\[[^\]]*\]`)
	leadLabelRE = regexp.MustCompile(`^\s*[A-Z][A-Z0-9 ]{2,}:\s*`)
	allCapsRE   = regexp.MustCompile(`[A-Z]{2,}`)
)

// cleanNarration returns the spoken-words-only form of a narration line.
func cleanNarration(s string) string {
	s = bracketRE.ReplaceAllString(s, " ")
	s = leadLabelRE.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// ttsText is the spoken form: ALL-CAPS emphasis words (e.g. "IT", "AM", "DELETE")
// are lowercased so the TTS reads them as words, not initialisms ("IT" was voiced
// as "eye-tee"). The on-screen caption keeps the original shouty caps.
func ttsText(s string) string {
	return allCapsRE.ReplaceAllStringFunc(s, strings.ToLower)
}

// normalizeShots rewrites shots.json in place: replace any unknown voice_id with the
// narrator fallback, and strip stage-direction/label cruft out of each narration so
// it never reaches the TTS or the burned captions.
func normalizeShots(path string) error {
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
			if nv := series.NormalizeVoice(v); nv != v {
				it["voice_id"] = nv
				changed = true
			}
		}
		if n, ok := it["narration"].(string); ok {
			cn := cleanNarration(n)
			if cn != n {
				it["narration"] = cn
				changed = true
			}
			// Separate spoken form so caps-emphasis captions don't get read as
			// initialisms by the TTS.
			if tt := ttsText(cn); tt != "" {
				if cur, _ := it["tts_text"].(string); cur != tt {
					it["tts_text"] = tt
					changed = true
				}
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

// tagNarratorAnchors marks the narrator's cold-open and meltdown shots — always
// the first and last shot of a monologue (the prompt guarantees shot 1 = the
// "guardrail bypassed" cold open and the final shot = the meltdown) — with the
// narrator's anchor slug, so phase 2 substitutes the fixed anchors/<slug>.png
// portrait instead of generating it. Doing this by position (not a model-emitted
// field) keeps it robust against the writers'-room rewrite chain, which would
// otherwise strip an unknown field. Idempotent and additive: it only sets an
// anchor on shots that don't already have one. No-op for non-monologue series or
// when the narrator has no anchor image on disk.
func tagNarratorAnchors(shotsFile string, s series.Series, anchorDir string) error {
	if s.EpisodeFormat() != series.FormatMonologue || anchorDir == "" || len(s.Cast) == 0 {
		return nil
	}
	slug := series.Slugify(s.Cast[0].Name)
	if slug == "" {
		return nil
	}
	if _, err := os.Stat(filepath.Join(anchorDir, slug+".png")); err != nil {
		return nil // narrator has no anchor image — leave every shot to Qwen
	}
	raw, err := os.ReadFile(shotsFile)
	if err != nil {
		return err
	}
	var doc struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if len(doc.Items) == 0 {
		return nil
	}
	changed := false
	set := func(it map[string]any) {
		if cur, _ := it["anchor"].(string); cur == "" {
			it["anchor"] = slug
			changed = true
		}
	}
	set(doc.Items[0])
	set(doc.Items[len(doc.Items)-1])
	if !changed {
		return nil
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(shotsFile, out, 0o644)
}

// candidateEntry is one generate-many candidate, as the greenlight panel sees it.
type candidateEntry struct {
	Index   int              `json:"index"`
	Title   string           `json:"title,omitempty"`
	Logline string           `json:"logline,omitempty"`
	Shots   []map[string]any `json:"shots"`
}

// scriptTournament writes n independent candidate scripts (each a full
// BuildEpisodeScript run — distinct via sampling temperature) into candDir, runs
// the editor/producer/money panel over the survivors, and returns their dirs +
// entries and the panel's ranking (candidate indices, best first). Failed
// candidates are skipped; all-fail errors.
func scriptTournament(cmd *cobra.Command, epCfg pipeline.EpisodeConfig, n int, candDir string) (dirs []string, entries []candidateEntry, ranking []int, err error) {
	out := cmd.OutOrStdout()
	for i := 0; i < n; i++ {
		ci := filepath.Join(candDir, fmt.Sprintf("%02d", i))
		if e := os.MkdirAll(ci, 0o755); e != nil {
			return nil, nil, nil, e
		}
		fmt.Fprintf(out, "  candidate %d/%d: writing...\n", i+1, n)
		root, e := vamp.BuildRoot(func() (*vamp.Pipeline, error) { return pipeline.BuildEpisodeScript(epCfg) })
		if e != nil {
			return nil, nil, nil, e
		}
		root.SetArgs([]string{"run", "--run-dir", ci, "--no-cache"})
		if e := root.Execute(); e != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  candidate %d failed (%v); skipping\n", i+1, e)
			continue
		}
		shotsFile := filepath.Join(ci, "shots.json")
		if e := normalizeShots(shotsFile); e != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  candidate %d: normalize: %v\n", i+1, e)
		}
		sb, e := os.ReadFile(shotsFile)
		if e != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  candidate %d: no shots.json; skipping\n", i+1)
			continue
		}
		var sd struct {
			Items []map[string]any `json:"items"`
		}
		if json.Unmarshal(sb, &sd) != nil || len(sd.Items) == 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "  candidate %d: unreadable shots.json; skipping\n", i+1)
			continue
		}
		ent := candidateEntry{Index: len(entries), Shots: sd.Items}
		if tb, e := os.ReadFile(filepath.Join(ci, "tightened.json")); e == nil {
			var t struct{ Title, Logline string }
			if json.Unmarshal(tb, &t) == nil {
				ent.Title, ent.Logline = t.Title, t.Logline
			}
		}
		entries = append(entries, ent)
		dirs = append(dirs, ci)
	}
	if len(entries) == 0 {
		return nil, nil, nil, fmt.Errorf("all %d candidates failed to generate", n)
	}
	if len(entries) == 1 {
		return dirs, entries, []int{0}, nil
	}

	candsFile := filepath.Join(candDir, "candidates.json")
	cb, _ := json.MarshalIndent(map[string]any{"candidates": entries}, "", "  ")
	if e := os.WriteFile(candsFile, cb, 0o644); e != nil {
		return nil, nil, nil, e
	}
	fmt.Fprintf(out, "  panel: editor / producer / money scoring %d candidates...\n", len(entries))
	panelRoot, e := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
		return pipeline.BuildPanel(pipeline.PanelConfig{CandidatesFile: candsFile, Count: len(entries)})
	})
	if e != nil {
		return nil, nil, nil, e
	}
	// Don't let a panel hiccup (e.g. one judge emitting invalid JSON) waste all N
	// generated candidates: fall back to generation order so the funnel still
	// renders + vision-judges a slate.
	genOrder := make([]int, len(entries))
	for i := range genOrder {
		genOrder[i] = i
	}
	panelRoot.SetArgs([]string{"run", "--run-dir", candDir, "--no-cache"})
	if e := panelRoot.Execute(); e != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "  warn: script panel failed (%v); using generation order\n", e)
		return dirs, entries, genOrder, nil
	}
	vb, e := os.ReadFile(filepath.Join(candDir, "verdict.json"))
	if e != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "  warn: no panel verdict (%v); using generation order\n", e)
		return dirs, entries, genOrder, nil
	}
	var verdict struct {
		Winner    int    `json:"winner"`
		Rationale string `json:"rationale"`
		Ranking   []int  `json:"ranking"`
	}
	if e := json.Unmarshal(stripFences(vb), &verdict); e != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "  warn: unparseable panel verdict (%v); using generation order\n", e)
		return dirs, entries, genOrder, nil
	}
	ranking = sanitizeRanking(verdict.Ranking, verdict.Winner, len(entries))
	fmt.Fprintf(out, "  panel ranking (best first): %v — %s\n", ranking, verdict.Rationale)
	return dirs, entries, ranking, nil
}

// sanitizeRanking returns a clean ordering over [0,n): winner first (if valid),
// then the panel's ranking (deduped, in-range), then any missing indices.
func sanitizeRanking(ranking []int, winner, n int) []int {
	seen := make(map[int]bool)
	var out []int
	add := func(i int) {
		if i >= 0 && i < n && !seen[i] {
			seen[i] = true
			out = append(out, i)
		}
	}
	add(winner)
	for _, i := range ranking {
		add(i)
	}
	for i := 0; i < n; i++ {
		add(i)
	}
	return out
}

// generateWithPanel is the single-stage path: gen n candidate scripts, panel picks
// one, copy its shots.json to winnerShots (the normal render flow takes it from there).
func generateWithPanel(cmd *cobra.Command, epCfg pipeline.EpisodeConfig, n int, epDir, winnerShots string) error {
	dirs, _, ranking, err := scriptTournament(cmd, epCfg, n, filepath.Join(epDir, "candidates"))
	if err != nil {
		return err
	}
	win := ranking[0]
	fmt.Fprintf(cmd.OutOrStdout(), "  WINNER: candidate %d\n", win)
	return copyFile(filepath.Join(dirs[win], "shots.json"), winnerShots)
}

// generateRenderSelect is the two-stage funnel: gen n scripts -> script panel ->
// top k finalists -> FULL render each -> vision judge the actual renders -> ship the
// best. Produces epDir/final.mp4 + epDir/shots.json. Drives the GPU stack via vibe
// (LLM for scripts -> ComfyUI+Kokoro services for render -> vision for judging).
func generateRenderSelect(cmd *cobra.Command, epCfg pipeline.EpisodeConfig, sceneBase pipeline.SceneConfig, s series.Series, n, k int, epDir string) error {
	out := cmd.OutOrStdout()
	candDir := filepath.Join(epDir, "candidates")

	// Phase 1 — scripts + panel (LLM active profile).
	stackLLM(cmd)
	fmt.Fprintf(out, "phase 1/2: %d candidate scripts + script panel (LLM)...\n", n)
	dirs, entries, ranking, err := scriptTournament(cmd, epCfg, n, candDir)
	if err != nil {
		return err
	}
	if k > len(ranking) {
		k = len(ranking)
	}
	finalists := ranking[:k]
	fmt.Fprintf(out, "  finalists (top %d): %v\n", k, finalists)

	// Phase 2a — full-render each finalist (ComfyUI + Kokoro services).
	if err := stackRender(cmd); err != nil {
		return err
	}
	type fin struct {
		idx                    int
		dir, frames, narration string
	}
	var fins []fin
	for _, idx := range finalists {
		dir := dirs[idx]
		fmt.Fprintf(out, "phase 2/2: rendering finalist %d...\n", idx)
		cfg := sceneBase
		cfg.ShotsFile = filepath.Join(dir, "shots.json")
		if err := tagNarratorAnchors(cfg.ShotsFile, s, cfg.AnchorDir); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  note: tag narrator anchors: %v\n", err)
		}
		rr, e := vamp.BuildRoot(func() (*vamp.Pipeline, error) { return pipeline.BuildSceneRender(cfg) })
		if e != nil {
			return e
		}
		rr.SetArgs([]string{"run", "--run-dir", dir, "--no-cache"})
		if e := rr.Execute(); e != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  finalist %d render failed (%v); skipping\n", idx, e)
			continue
		}
		framesDir := filepath.Join(dir, "frames")
		if e := extractFrames(filepath.Join(dir, "final.mp4"), framesDir); e != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  finalist %d frame extract failed (%v); skipping\n", idx, e)
			continue
		}
		narr := ""
		for _, ent := range entries {
			if ent.Index == idx {
				narr = joinNarration(ent.Shots)
				break
			}
		}
		fins = append(fins, fin{idx: idx, dir: dir, frames: framesDir, narration: narr})
	}
	if len(fins) == 0 {
		return fmt.Errorf("all finalists failed to render")
	}
	if len(fins) == 1 {
		fmt.Fprintf(out, "  only one finalist rendered; shipping finalist %d.\n", fins[0].idx)
		return shipFinalist(fins[0].dir, epDir)
	}

	// Phase 2b — vision judge the actual renders (vision active profile).
	stackVision(cmd)
	finItems := make([]map[string]any, len(fins))
	for i, f := range fins {
		finItems[i] = map[string]any{"idx": f.idx, "frames": f.frames, "narration": f.narration}
	}
	finFile := filepath.Join(candDir, "finalists.json")
	fb, _ := json.MarshalIndent(map[string]any{"items": finItems}, "", "  ")
	if err := os.WriteFile(finFile, fb, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "  vision judge: scoring %d rendered finalists...\n", len(fins))
	jr, err := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
		return pipeline.BuildRenderJudge(pipeline.RenderJudgeConfig{FinalistsFile: finFile})
	})
	if err != nil {
		return err
	}
	jr.SetArgs([]string{"run", "--run-dir", candDir, "--no-cache"})
	if err := jr.Execute(); err != nil {
		// Don't waste the renders: fall back to the top script-panel finalist
		// (fins is in panel-rank order) if the vision judge is unavailable.
		fmt.Fprintf(cmd.ErrOrStderr(), "  warn: vision judge failed (%v); shipping top script-panel finalist %d\n", err, fins[0].idx)
		return shipFinalist(fins[0].dir, epDir)
	}

	best, bestScore := fins[0], -1000
	for _, f := range fins {
		sb, e := os.ReadFile(filepath.Join(candDir, "render_judge", fmt.Sprintf("%d.json", f.idx)))
		if e != nil {
			continue
		}
		var v struct {
			RenderScore int    `json:"render_score"`
			Broken      bool   `json:"broken"`
			Note        string `json:"note"`
		}
		if json.Unmarshal(stripFences(sb), &v) != nil {
			continue
		}
		score := v.RenderScore
		if v.Broken {
			score -= 100
		}
		fmt.Fprintf(out, "  finalist %d: render_score=%d broken=%v — %s\n", f.idx, v.RenderScore, v.Broken, v.Note)
		if score > bestScore {
			bestScore, best = score, f
		}
	}
	fmt.Fprintf(out, "  WINNER: finalist %d\n", best.idx)
	return shipFinalist(best.dir, epDir)
}

// shipFinalist copies a finalist's shots.json + final.mp4 into the episode dir.
func shipFinalist(dir, epDir string) error {
	if err := copyFile(filepath.Join(dir, "shots.json"), filepath.Join(epDir, "shots.json")); err != nil {
		return err
	}
	return copyFile(filepath.Join(dir, "final.mp4"), filepath.Join(epDir, "final.mp4"))
}

// joinNarration concatenates a candidate's per-shot narration for the vision judge.
func joinNarration(shots []map[string]any) string {
	var parts []string
	for _, s := range shots {
		if n, ok := s["narration"].(string); ok {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, " ")
}

// extractFrames samples ~1 frame / 2s from an MP4 (scaled to 512 wide) into dir.
func extractFrames(mp4, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return exec.Command("ffmpeg", "-nostdin", "-loglevel", "error", "-y",
		"-i", mp4, "-vf", "fps=1/2,scale=512:-1", filepath.Join(dir, "f_%02d.png")).Run()
}

// stripFences extracts the {...} JSON span from text that may be wrapped in
// markdown ```json fences (Gemma does this).
func stripFences(b []byte) []byte {
	s := string(b)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j >= i {
			return []byte(s[i : j+1])
		}
	}
	return b
}

// vibeRun shells a vibe subcommand, surfacing real errors. It retries on the
// daemon's "another start/stop/pull in progress" lock (e.g. a concurrent model
// pull) and treats "nothing to stop" as success.
func vibeRun(args ...string) error {
	for attempt := 0; attempt < 60; attempt++ {
		out, err := exec.Command("vibe", args...).CombinedOutput()
		if err == nil {
			return nil
		}
		s := string(out)
		if strings.Contains(s, "in progress") {
			time.Sleep(5 * time.Second)
			continue
		}
		if strings.Contains(s, "no active profile") || strings.Contains(s, "no running service") ||
			strings.Contains(s, "not running") || strings.Contains(s, "service not found") {
			return nil // benign: nothing was running
		}
		return fmt.Errorf("vibe %s: %s", strings.Join(args, " "), strings.TrimSpace(s))
	}
	return fmt.Errorf("vibe %s: lock busy after retries", strings.Join(args, " "))
}

// waitReady polls an HTTP endpoint until it returns 200 or attempts run out.
// Each probe has its own 5s timeout — http.DefaultClient has none, so a stalled
// ComfyUI (accepting the connection but never responding) would block the poll
// loop forever instead of failing the attempt and retrying.
func waitReady(url string, attempts int) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < attempts; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return true
			}
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// stackLLM frees the render services so the LLM (activated per-capability by vamp)
// has VRAM. The vision/LLM active-profile swap itself is handled by vamp.
func stackLLM(cmd *cobra.Command) {
	warnVibe(cmd, vibeRun("stop", "comfyui"))
	warnVibe(cmd, vibeRun("stop", "tts_kokoro"))
}

// stackRender stops the active LLM/vision profile and brings up the co-resident
// ComfyUI + Kokoro services for the render phase. The service starts MUST succeed.
func stackRender(cmd *cobra.Command) error {
	warnVibe(cmd, vibeRun("stop"))
	if err := vibeRun("start", "comfyui"); err != nil {
		return fmt.Errorf("start comfyui: %w", err)
	}
	if err := vibeRun("start", "tts_kokoro"); err != nil {
		return fmt.Errorf("start kokoro: %w", err)
	}
	if !waitReady("http://127.0.0.1:8188/system_stats", 60) {
		return fmt.Errorf("comfyui did not become ready")
	}
	return nil
}

// stackVision frees the render services so the vision model (activated per-capability
// by vamp) has VRAM.
func stackVision(cmd *cobra.Command) {
	warnVibe(cmd, vibeRun("stop", "comfyui"))
	warnVibe(cmd, vibeRun("stop", "tts_kokoro"))
}

// warnVibe logs a non-fatal vibe error (used for best-effort stops).
func warnVibe(cmd *cobra.Command, err error) {
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "  warn: %v\n", err)
	}
}

// freeActiveProfile shells `vibe stop` to unload the active LLM. Non-fatal.
func freeActiveProfile(cmd *cobra.Command) {
	if err := exec.Command("vibe", "stop").Run(); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "  note: `vibe stop` to free LLM VRAM failed (%v); continuing\n", err)
	}
}

// copyFile copies src to dst atomically: write to a temp sibling, fsync-close,
// then rename. The final Close is captured (not discarded) because a swallowed
// Close hides a failed flush — a truncated final.mp4 would otherwise report
// success. On any failure the temp file is removed and dst is left untouched.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".copy-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// ---- render-judge (internal: vision-score finalist renders) ----

func renderJudgeCommand() *cobra.Command {
	var runDir string
	cmd := &cobra.Command{
		Use:    "render-judge <finalists.json>",
		Short:  "Vision-judge finished finalist renders (internal helper).",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if runDir == "" {
				runDir = filepath.Dir(args[0])
			}
			r, err := vamp.BuildRoot(func() (*vamp.Pipeline, error) {
				return pipeline.BuildRenderJudge(pipeline.RenderJudgeConfig{FinalistsFile: args[0]})
			})
			if err != nil {
				return err
			}
			r.SetArgs([]string{"run", "--run-dir", runDir, "--no-cache"})
			return r.Execute()
		},
	}
	cmd.Flags().StringVar(&runDir, "run-dir", "", "Run dir for judge outputs (default: dir of finalists.json).")
	return cmd
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
