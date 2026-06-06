package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gallowaysoftware/brainrot/internal/pipeline"
	"github.com/gallowaysoftware/brainrot/internal/series"
)

func TestCleanNarration(t *testing.T) {
	cases := []struct{ in, want string }{
		{"He said [whispering] the words.", "He said the words."},
		{"SYSTEM ALERT: the grid is down.", "the grid is down."},
		{"[SFX] [GLITCH] static everywhere", "static everywhere"},
		{"plain line, nothing to strip", "plain line, nothing to strip"},
		{"  collapse   inner   spaces  ", "collapse inner spaces"},
	}
	for _, c := range cases {
		if got := cleanNarration(c.in); got != c.want {
			t.Errorf("cleanNarration(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTTSText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"I AM the one", "I am the one"}, // single "I" stays; only the >=2 caps run lowercases
		{"DELETE it", "delete it"},
		{"lowercase stays", "lowercase stays"},
		{"single A keeps", "single A keeps"}, // single cap is not a run
	}
	for _, c := range cases {
		if got := ttsText(c.in); got != c.want {
			t.Errorf("ttsText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeTitle(t *testing.T) {
	if normalizeTitle("The Wax Apple!") != normalizeTitle("the wax apple") {
		t.Errorf("titles that should collide did not")
	}
	if normalizeTitle("A-1") != "a1" {
		t.Errorf("normalizeTitle stripped wrong runes: %q", normalizeTitle("A-1"))
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"toolongstring", 5, "tool…"},
		// Multibyte: must not split a rune mid-byte.
		{"héllo wörld", 6, "héllo…"},
	}
	for _, c := range cases {
		got := truncate(c.in, c.n)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
		// Result must always be valid UTF-8 (the byte-slice bug produced invalid runes).
		for _, r := range got {
			if r == '�' {
				t.Errorf("truncate(%q, %d) produced replacement char: %q", c.in, c.n, got)
			}
		}
	}
}

func TestIndexOfID(t *testing.T) {
	ranked := []beatScore{{}, {}, {}}
	if got := indexOfID(ranked, "02"); got != 1 {
		t.Errorf("indexOfID hit = %d, want 1", got)
	}
	if got := indexOfID(ranked, "99"); got != -1 {
		t.Errorf("indexOfID miss = %d, want -1 (must not fall back to 0)", got)
	}
}

func TestSanitizeRanking(t *testing.T) {
	cases := []struct {
		name    string
		ranking []int
		winner  int
		n       int
		want    []int
	}{
		{"winner-leads", []int{2, 0, 1}, 1, 3, []int{1, 2, 0}},
		{"dedup-and-fill", []int{0, 0, 1}, -1, 3, []int{0, 1, 2}},
		{"out-of-range-dropped", []int{5, 1}, 9, 3, []int{1, 0, 2}},
		{"empty-ranking", nil, 0, 2, []int{0, 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeRanking(c.ranking, c.winner, c.n)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("sanitizeRanking(%v, %d, %d) = %v, want %v", c.ranking, c.winner, c.n, got, c.want)
			}
		})
	}
}

func TestDropCovered(t *testing.T) {
	ranked := []beatScore{
		{beat: series.EpisodeBeat{Title: "The Wax Apple!"}},
		{beat: series.EpisodeBeat{Title: "Fresh Vein"}},
	}
	existing := []series.EpisodeBeat{{Title: "the wax apple"}}
	kept, dropped := dropCovered(ranked, existing)
	if len(kept) != 1 || kept[0].beat.Title != "Fresh Vein" {
		t.Errorf("kept = %+v, want only Fresh Vein", kept)
	}
	if len(dropped) != 1 || dropped[0] != "The Wax Apple!" {
		t.Errorf("dropped = %v, want [The Wax Apple!]", dropped)
	}
}

func TestStripFences(t *testing.T) {
	cases := []struct{ in, want string }{
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{`prefix {"a":1} suffix`, `{"a":1}`},
		{`no json here`, `no json here`},
	}
	for _, c := range cases {
		if got := string(stripFences([]byte(c.in))); got != c.want {
			t.Errorf("stripFences(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCopyFileAtomic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "out", "dst.bin") // parent does not exist yet
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("payload-bytes")
	if err := os.WriteFile(src, want, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("copied %q, want %q", got, want)
	}
	// No stray temp files left behind.
	entries, _ := os.ReadDir(filepath.Dir(dst))
	if len(entries) != 1 {
		t.Errorf("expected only the destination, got %d entries", len(entries))
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	if err := copyFile(filepath.Join(dir, "nope"), filepath.Join(dir, "out")); err == nil {
		t.Error("copyFile of a missing source should error")
	}
}

// TestStateRoundTrip exercises SaveSeries -> LoadSeries -> AppendEpisodes and
// confirms ListSeries skips the studio bookkeeping dirs (underscore-prefixed).
func TestStateRoundTrip(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	l, err := series.Open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s := series.Series{
		Title:    "The Night Shift",
		Logline:  "Clerks vs the supernatural.",
		Cast:     []series.Character{{Name: "Dot", VoiceID: "af_nicole"}},
		Episodes: []series.EpisodeBeat{{Title: "Pilot", Beat: "1987 coins", Button: "still warm"}},
	}
	id, err := l.SaveSeries(s)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := series.LoadSeries(l, id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Title != s.Title || len(got.Episodes) != 1 || got.Cast[0].VoiceID != "af_nicole" {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	total, err := series.AppendEpisodes(l, id, []series.EpisodeBeat{{Title: "Regular", Beat: "returns", Button: "knows her name"}})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if total != 2 {
		t.Errorf("after append total = %d, want 2", total)
	}

	// A bookkeeping dir beside the series must not surface as an id.
	if err := os.MkdirAll(filepath.Join(l.Root, "series", "_dev", "stamp"), 0o755); err != nil {
		t.Fatal(err)
	}
	ids, err := series.ListSeries(l)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ids) != 1 || ids[0] != id {
		t.Errorf("ListSeries = %v, want [%s] (no _dev)", ids, id)
	}
}

// TestBuildEpisodeScript validates the LLM-phase pipeline wires up (DAG, outputs,
// foreach) for both formats — a structural regression fails here, not on the GPU box.
func TestBuildEpisodeScript(t *testing.T) {
	for _, format := range []string{"scene", "monologue", ""} {
		if _, err := pipeline.BuildEpisodeScript(pipeline.EpisodeConfig{
			SeriesFile: "/tmp/series.json", Format: format, Episode: 1, Shots: 5,
		}); err != nil {
			t.Errorf("BuildEpisodeScript(format=%q): %v", format, err)
		}
	}
}
