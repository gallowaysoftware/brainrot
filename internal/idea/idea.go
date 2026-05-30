// Package idea models a brainrot content seed: a castable character + a
// compelling situation, oriented entirely toward short-form video — no world
// bibles. Breadth generates many; the evaluate phase scores them for
// TikTok potential and keeps the best.
package idea

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Character is a castable, visually-distinct persona. Look drives image gen;
// VoiceID drives Kokoro casting; Vibe is the energy/personality in one line.
type Character struct {
	Name    string `json:"name"`
	Look    string `json:"look"`
	VoiceID string `json:"voice_id"`
	Vibe    string `json:"vibe"`
}

// Idea is one content seed. Situation/Hook/Format are the content; Score is
// filled by the evaluate phase.
type Idea struct {
	Character  Character `json:"character"`
	Situation  string    `json:"situation"`
	Hook       string    `json:"hook"`
	Format     string    `json:"format"`
	WhyItWorks string    `json:"why_it_works,omitempty"`
	Score      *Score    `json:"score,omitempty"`
}

// Score is the evaluate phase's judgement, imagining the finished TikTok.
// Each axis is 1-10; Total is their sum (max 60).
type Score struct {
	Hook         int    `json:"hook"`         // scroll-stop power of shot 1
	Shootability int    `json:"shootability"` // works as stills -> image-to-video
	Punch        int    `json:"punch"`        // comedic / emotional payoff
	Legibility   int    `json:"legibility"`   // a cold viewer gets it in 25s
	Loop         int    `json:"loop"`         // rewatch / loop / share pull
	FormatFit    int    `json:"format_fit"`   // suits the chosen format
	Total        int    `json:"total"`
	Rationale    string `json:"rationale,omitempty"`
}

// computeTotal sums the axes (authoritative — we don't trust the model's own
// arithmetic on Total).
func (s *Score) computeTotal() {
	s.Total = s.Hook + s.Shootability + s.Punch + s.Legibility + s.Loop + s.FormatFit
}

// ParseIdeas decodes a JSON array of ideas (or a {"ideas":[...]} wrapper).
func ParseIdeas(b []byte) ([]Idea, error) {
	b = []byte(strings.TrimSpace(string(b)))
	var wrap struct {
		Ideas []Idea `json:"ideas"`
	}
	if err := json.Unmarshal(b, &wrap); err == nil && len(wrap.Ideas) > 0 {
		return wrap.Ideas, nil
	}
	var arr []Idea
	if err := json.Unmarshal(b, &arr); err != nil {
		return nil, fmt.Errorf("parse ideas: %w", err)
	}
	return arr, nil
}

// ApplyScores merges a scored-ideas JSON (array aligned by index, each with a
// "score" object) onto ideas, recomputing Totals. Robust to length mismatch:
// only as many as line up get scored; unscored ideas keep Score nil.
func ApplyScores(ideas []Idea, scoredJSON []byte) error {
	var scored []struct {
		Score *Score `json:"score"`
	}
	s := strings.TrimSpace(string(scoredJSON))
	var wrap struct {
		Ideas []struct {
			Score *Score `json:"score"`
		} `json:"ideas"`
	}
	if err := json.Unmarshal([]byte(s), &wrap); err == nil && len(wrap.Ideas) > 0 {
		scored = wrap.Ideas
	} else if err := json.Unmarshal([]byte(s), &scored); err != nil {
		return fmt.Errorf("parse scores: %w", err)
	}
	for i := range ideas {
		if i < len(scored) && scored[i].Score != nil {
			scored[i].Score.computeTotal()
			ideas[i].Score = scored[i].Score
		}
	}
	return nil
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slug derives a filesystem-safe id from the character name + a short hint
// from the hook, so two ideas about different characters don't collide.
func (i Idea) Slug() string {
	base := i.Character.Name
	out := slugRE.ReplaceAllString(strings.ToLower(base), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		out = "idea"
	}
	return out
}

// validVoices is the set of Kokoro voice ids brainrot offers; an unknown /
// typo'd voice 400s the TTS stage, so we normalize to the narrator fallback.
var validVoices = map[string]bool{
	"am_fenrir": true, "am_michael": true, "am_puck": true,
	"am_adam": true, "am_eric": true,
	"af_bella": true, "af_nicole": true, "bf_emma": true,
}

// NarratorVoice is the default and the fallback for an unknown voice_id.
const NarratorVoice = "am_fenrir"

// NormalizeVoice returns v if it's a known voice, else NarratorVoice.
func NormalizeVoice(v string) string {
	v = strings.TrimSpace(v)
	if validVoices[v] {
		return v
	}
	return NarratorVoice
}
