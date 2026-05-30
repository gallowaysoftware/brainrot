package idea

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `{"ideas":[
  {"character":{"name":"Gary","look":"40s sweaty man in a hot-dog suit","voice_id":"am_adam","vibe":"unbothered"},
   "situation":"narrates his own kidnapping like a cooking show","hook":"Step one: stay calm.","format":"fake tutorial","why_it_works":"absurd calm vs chaos"},
  {"character":{"name":"Mothra","look":"giant moth in a tiny hat","voice_id":"zz_bogus","vibe":"diva"},
   "situation":"rates the city's porch lights","hook":"This one? Two stars.","format":"ranked list","why_it_works":"silly authority"}
]}`

const scores = `{"ideas":[
  {"score":{"hook":8,"shootability":7,"punch":9,"legibility":9,"loop":7,"format_fit":8,"rationale":"strong"}},
  {"score":{"hook":5,"shootability":4,"punch":5,"legibility":6,"loop":4,"format_fit":6,"rationale":"mid"}}
]}`

func TestParseApplyScoreSort(t *testing.T) {
	ideas, err := ParseIdeas([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(ideas) != 2 || ideas[0].Character.Name != "Gary" {
		t.Fatalf("parse wrong: %+v", ideas)
	}
	if err := ApplyScores(ideas, []byte(scores)); err != nil {
		t.Fatal(err)
	}
	if ideas[0].Score == nil || ideas[0].Score.Total != 48 {
		t.Errorf("idea0 total = %v, want 48", ideas[0].Score)
	}
	if ideas[1].Score.Total != 30 {
		t.Errorf("idea1 total = %v, want 30", ideas[1].Score)
	}
	// Bad voice normalizes.
	for i := range ideas {
		ideas[i].Character.VoiceID = NormalizeVoice(ideas[i].Character.VoiceID)
	}
	if ideas[1].Character.VoiceID != "am_fenrir" {
		t.Errorf("bad voice not normalized: %q", ideas[1].Character.VoiceID)
	}
	SortByScore(ideas)
	if ideas[0].Character.Name != "Gary" {
		t.Errorf("sort wrong: top is %q (want Gary, the higher score)", ideas[0].Character.Name)
	}
}

func TestSaveLoadIdea(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	l, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	id, err := l.SaveIdea(Idea{Character: Character{Name: "Gary", VoiceID: "am_adam"}, Hook: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "gary" {
		t.Errorf("id = %q, want gary", id)
	}
	got, err := LoadIdea(l, id)
	if err != nil || got.Hook != "hi" {
		t.Fatalf("load: %v %+v", err, got)
	}
	// Second save of same name gets a unique id.
	id2, _ := l.SaveIdea(Idea{Character: Character{Name: "Gary"}})
	if id2 == id {
		t.Errorf("duplicate id not uniquified: %q", id2)
	}
	if _, err := os.Stat(filepath.Join(l.IdeaDir(id), "idea.json")); err != nil {
		t.Errorf("idea.json missing: %v", err)
	}
}
