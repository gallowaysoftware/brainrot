package series

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleList = `{"series":[
  {"title":"The Night Shift","logline":"Two gas-station clerks vs the supernatural.",
   "premise":"A skeptic and a believer work graveyard shifts.","tone":"deadpan horror-comedy",
   "cast":[
     {"name":"Marcus","role":"skeptic","look":"tall, tired, hoodie","voice_id":"am_michael","vibe":"flat"},
     {"name":"Dot","role":"believer","look":"short, wide-eyed, beanie","voice_id":"af_nicole","vibe":"manic"}],
   "episodes":[
     {"title":"Pilot","beat":"A customer pays in 1987 coins.","button":"The coins are still warm."},
     {"title":"Regular","beat":"The coin-man returns nightly.","button":"He knows Dot's name."}]},
  {"title":"HOA Hell","logline":"A new homeowner vs an unhinged HOA president.",
   "premise":"Every episode a pettier rule.","tone":"escalating cringe",
   "cast":[{"name":"Priya","role":"victim","look":"30s, blazer","voice_id":"bf_emma","vibe":"dry"}],
   "episodes":[{"title":"Welcome","beat":"A fine for grass height.","button":"She measures with a ruler."}]}
]}`

func TestParseSeriesList(t *testing.T) {
	list, err := ParseSeriesList([]byte(sampleList))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 series, got %d", len(list))
	}
	if list[0].Title != "The Night Shift" || len(list[0].Cast) != 2 || len(list[0].Episodes) != 2 {
		t.Errorf("series[0] malformed: %+v", list[0])
	}
	if list[0].Cast[1].VoiceID != "af_nicole" {
		t.Errorf("cast voice not parsed: %q", list[0].Cast[1].VoiceID)
	}
}

// An empty wrapped array is a valid, distinct result from a parse error: a model
// that returns {"series":[]} means "nothing", not "malformed". The structural
// (object-vs-array) detection must surface it as an empty slice, no error.
func TestParseSeriesEmptyWrapped(t *testing.T) {
	list, err := ParseSeriesList([]byte(`{"series":[]}`))
	if err != nil {
		t.Fatalf("empty wrapped series should not error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("want empty slice, got %d series", len(list))
	}
}

func TestParseEpisodeBeats(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"wrapped", `{"episodes":[{"title":"a","beat":"b","button":"c"},{"title":"d","beat":"e","button":"f"}]}`, 2},
		{"bare-array", `[{"title":"a","beat":"b","button":"c"}]`, 1},
		{"empty-wrapped", `{"episodes":[]}`, 0},
		{"leading-space-wrapped", "  \n{\"episodes\":[{\"title\":\"a\",\"beat\":\"b\",\"button\":\"c\"}]}", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			beats, err := ParseEpisodeBeats([]byte(c.in))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(beats) != c.want {
				t.Errorf("got %d beats, want %d", len(beats), c.want)
			}
		})
	}
	if _, err := ParseEpisodeBeats([]byte(`not json`)); err == nil {
		t.Error("malformed input should error")
	}
}

func TestEpisodeFormat(t *testing.T) {
	cases := map[string]string{
		"":          FormatScene,     // default
		"scene":     FormatScene,     // explicit
		"monologue": FormatMonologue, // explicit
		"garbage":   FormatScene,     // unknown folds to the default
	}
	for in, want := range cases {
		if got := (Series{Format: in}).EpisodeFormat(); got != want {
			t.Errorf("EpisodeFormat(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Codex":          "codex",
		"The Coat Saga!": "the-coat-saga",
		"  Sootlip  ":    "sootlip",
		"A--B":           "a-b",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// uniqueID must never reuse an occupied base id (that would clobber an existing
// series). It suffixes on collision and errors only when the suffix space is
// exhausted.
func TestUniqueID(t *testing.T) {
	l := Layout{Root: t.TempDir()}
	mk := func(id string) {
		if err := os.MkdirAll(l.SeriesDir(id), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if got, err := l.uniqueID("goblin-town"); err != nil || got != "goblin-town" {
		t.Fatalf("free base: got %q err %v, want goblin-town", got, err)
	}
	mk("goblin-town")
	if got, err := l.uniqueID("goblin-town"); err != nil || got != "goblin-town-2" {
		t.Fatalf("first collision: got %q err %v, want goblin-town-2", got, err)
	}
	mk("goblin-town-2")
	if got, err := l.uniqueID("goblin-town"); err != nil || got != "goblin-town-3" {
		t.Fatalf("second collision: got %q err %v, want goblin-town-3", got, err)
	}
}

func TestNextEpisode(t *testing.T) {
	l := Layout{Root: t.TempDir()}
	id := "show"
	if got := NextEpisode(l, id); got != 1 {
		t.Fatalf("no episodes yet: NextEpisode = %d, want 1", got)
	}
	// Mark episodes 1 and 2 done (final.mp4 present); 3 is the next gap.
	for n := 1; n <= 2; n++ {
		dir := l.EpisodeDir(id, n)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "final.mp4"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := NextEpisode(l, id); got != 3 {
		t.Errorf("NextEpisode = %d, want 3", got)
	}
}

func TestParseSeriesBareArray(t *testing.T) {
	bare := `[{"title":"X","episodes":[{"title":"a","beat":"b","button":"c"}]}]`
	list, err := ParseSeriesList([]byte(bare))
	if err != nil || len(list) != 1 || list[0].Title != "X" {
		t.Fatalf("bare array parse failed: %v %+v", err, list)
	}
}

func TestApplyScoresAndTotal(t *testing.T) {
	list, _ := ParseSeriesList([]byte(sampleList))
	scores := `{"series":[
      {"score":{"premise":8,"cast":9,"comedy":7,"arc":6,"bingeability":8,"hook":9,"rationale":"strong"}},
      {"score":{"premise":4,"cast":3,"comedy":5,"arc":4,"bingeability":3,"hook":5,"rationale":"thin"}}
    ]}`
	if err := ApplyScores(list, []byte(scores)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if list[0].Score == nil || list[0].Score.Total != 47 {
		t.Errorf("series[0] total: want 47, got %+v", list[0].Score)
	}
	if list[1].Score == nil || list[1].Score.Total != 24 {
		t.Errorf("series[1] total: want 24, got %+v", list[1].Score)
	}
	SortByScore(list)
	if list[0].Title != "The Night Shift" {
		t.Errorf("sort by score failed; top is %q", list[0].Title)
	}
}

func TestRankPrefersGreenlight(t *testing.T) {
	// Higher craft total but lower greenlight should rank BELOW a viral-hook pick.
	craft := &Score{Premise: 9, Cast: 9, Comedy: 9, Arc: 9, Bingeability: 9, Hook: 9, Greenlight: 60}
	craft.computeTotal() // 54/60
	viral := &Score{Premise: 6, Cast: 6, Comedy: 7, Arc: 5, Bingeability: 8, Hook: 9, Greenlight: 88}
	viral.computeTotal() // 41/60
	if craft.Rank() >= viral.Rank() {
		t.Errorf("greenlight should drive rank: craft %d vs viral %d", craft.Rank(), viral.Rank())
	}
	// No greenlight → fall back to craft total scaled to 0-100.
	noGL := &Score{Premise: 10, Cast: 10, Comedy: 10, Arc: 10, Bingeability: 10, Hook: 10}
	noGL.computeTotal()
	if noGL.Rank() != 100 {
		t.Errorf("full craft, no greenlight: want Rank 100, got %d", noGL.Rank())
	}
	if (*Score)(nil).Rank() != 0 {
		t.Errorf("nil score Rank should be 0")
	}
}

func TestSortByScoreUsesGreenlight(t *testing.T) {
	list := []Series{
		{Title: "craft", Score: &Score{Greenlight: 60}},
		{Title: "viral", Score: &Score{Greenlight: 88}},
		{Title: "unscored"},
	}
	SortByScore(list)
	if list[0].Title != "viral" || list[2].Title != "unscored" {
		t.Errorf("sort order wrong: %s, %s, %s", list[0].Title, list[1].Title, list[2].Title)
	}
}

func TestNormalizeVoice(t *testing.T) {
	cases := map[string]string{
		"am_michael":  "am_michael",
		"af_nicole":   "af_nicole",
		"am_fenfir":   NarratorVoice, // common typo seen in the wild
		"":            NarratorVoice,
		"  am_puck  ": "am_puck",
		"nonsense":    NarratorVoice,
	}
	for in, want := range cases {
		if got := NormalizeVoice(in); got != want {
			t.Errorf("NormalizeVoice(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"The Night Shift": "the-night-shift",
		"HOA Hell!":       "hoa-hell",
		"  Spaced  Out  ": "spaced-out",
		"":                "series",
	}
	for in, want := range cases {
		if got := (Series{Title: in}).Slug(); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
