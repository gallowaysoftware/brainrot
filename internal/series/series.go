// Package series models a serialized short-form video story: a small recurring
// cast and an overarching arc broken into episode beats. Breadth generates
// series concepts and scores them; depth turns one episode beat into a
// multi-character scene script + video. Content-first — the writing is the
// product; visuals are downstream.
package series

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Character is a recurring cast member, consistent across episodes.
type Character struct {
	Name    string `json:"name"`
	Role    string `json:"role"`     // their function in the story (protagonist, foil, ...)
	Look    string `json:"look"`     // concrete, castable, identical every episode
	VoiceID string `json:"voice_id"` // Kokoro voice
	Vibe    string `json:"vibe"`     // voice/energy on the page
}

// EpisodeBeat is one episode's slot in the overarching arc: a self-contained
// scene that also advances the story and hooks the next one.
type EpisodeBeat struct {
	Title  string `json:"title"`
	Beat   string `json:"beat"`   // what happens in this episode's scene
	Button string `json:"button"` // the closing hook / cliffhanger into the next
	// Lore is the canon this specific episode must surface: source material, the
	// verbatim lines, named objects/proverbs/taboos it should feature. It's how a
	// lore-heavy source reaches the screen episode-by-episode instead of being
	// distilled away — the episode writer riffs on THIS, in voice. Optional.
	Lore string `json:"lore,omitempty"`
}

// Series is a serialized story concept: the bible the episodes are written
// against.
type Series struct {
	Title   string `json:"title"`
	Logline string `json:"logline"`
	Premise string `json:"premise"`
	Tone    string `json:"tone"`
	// Engine is the comedic structure that generates every scene (e.g. odd-couple,
	// escalating-lie, dramatic-irony, obsession, status-games). Kept distinct
	// across series so the slate isn't one joke wearing many hats.
	Engine string `json:"engine,omitempty"`
	// POV is the human truth the show exaggerates — what it's actually about under
	// the premise. The thing that makes it land beyond a gimmick.
	POV string `json:"pov,omitempty"`
	// Format selects how episodes are written. "scene" (default) is a
	// multi-character scene where lines answer each other; "monologue" is a single
	// narrating voice (the first cast member) delivering an escalating lore-drop
	// over illustrated tableaux — for shows whose comedy is one voice, not a
	// conversation (e.g. an AI rambling about goblins).
	Format string `json:"format,omitempty"`
	// Shots is this series' preferred shot count per episode (its target length).
	// When set, it overrides the make default; an explicit --shots still wins.
	// Lets a tight TikTok monologue run at 5 while a denser scene show runs at 7.
	Shots    int           `json:"shots,omitempty"`
	Cast     []Character   `json:"cast"`
	Episodes []EpisodeBeat `json:"episodes"`
	Score    *Score        `json:"score,omitempty"`
}

// Episode formats.
const (
	FormatScene     = "scene"
	FormatMonologue = "monologue"
)

// EpisodeFormat returns the series' episode format, defaulting to "scene".
func (s Series) EpisodeFormat() string {
	if s.Format == FormatMonologue {
		return FormatMonologue
	}
	return FormatScene
}

// Score is the studio's verdict on a series. Craft axes are 1-10 (Total/60);
// the audience block is the metrics-driven estimate; Greenlight is the final
// blended studio number (0-100) the ranking sorts on.
type Score struct {
	// Craft axes (1-10).
	Premise      int `json:"premise"`      // fresh, not a worn template
	Cast         int `json:"cast"`         // distinct voices + real chemistry/conflict
	Comedy       int `json:"comedy"`       // actually funny / compelling, with payoffs
	Arc          int `json:"arc"`          // the overarching story escalates to a real payoff
	Bingeability int `json:"bingeability"` // do you want the next episode?
	Hook         int `json:"hook"`         // episode-1 scroll-stop
	Total        int `json:"total"`        // sum of the six craft axes

	// Audience estimation (metrics-driven studio projection).
	ScrollStop   int    `json:"scroll_stop,omitempty"`   // predicted % who stop on shot 1 (0-100)
	Retention    int    `json:"retention,omitempty"`     // predicted % who finish (0-100)
	Shareability int    `json:"shareability,omitempty"`  // 1-10
	FollowIntent int    `json:"follow_intent,omitempty"` // 1-10 (follow for more episodes)
	Audience     string `json:"audience,omitempty"`      // target audience + rough size
	Comps        string `json:"comps,omitempty"`         // comparable hits

	// Greenlight is the final studio verdict, 0-100, blending craft + audience.
	Greenlight int    `json:"greenlight,omitempty"`
	Rationale  string `json:"rationale,omitempty"`
}

func (s *Score) computeTotal() {
	s.Total = s.Premise + s.Cast + s.Comedy + s.Arc + s.Bingeability + s.Hook
}

// Rank is the value the ranking sorts on: the studio greenlight if present, else
// the craft total scaled into the same 0-100 range.
func (s *Score) Rank() int {
	if s == nil {
		return 0
	}
	if s.Greenlight > 0 {
		return s.Greenlight
	}
	return s.Total * 100 / 60
}

// ParseSeriesList decodes a JSON array of series (or {"series":[...]}).
func ParseSeriesList(b []byte) ([]Series, error) {
	t := strings.TrimSpace(string(b))
	var wrap struct {
		Series []Series `json:"series"`
	}
	if err := json.Unmarshal([]byte(t), &wrap); err == nil && len(wrap.Series) > 0 {
		return wrap.Series, nil
	}
	var arr []Series
	if err := json.Unmarshal([]byte(t), &arr); err != nil {
		return nil, fmt.Errorf("parse series: %w", err)
	}
	return arr, nil
}

// ParseEpisodeBeats decodes a plan's beats JSON: {"episodes":[...]} or a bare
// array of episode beats.
func ParseEpisodeBeats(b []byte) ([]EpisodeBeat, error) {
	t := strings.TrimSpace(string(b))
	var wrap struct {
		Episodes []EpisodeBeat `json:"episodes"`
	}
	if err := json.Unmarshal([]byte(t), &wrap); err == nil && len(wrap.Episodes) > 0 {
		return wrap.Episodes, nil
	}
	var arr []EpisodeBeat
	if err := json.Unmarshal([]byte(t), &arr); err != nil {
		return nil, fmt.Errorf("parse episode beats: %w", err)
	}
	return arr, nil
}

// ApplyScores merges a scores JSON (index-aligned, each {"score":{...}}).
func ApplyScores(list []Series, scoredJSON []byte) error {
	t := strings.TrimSpace(string(scoredJSON))
	var scored []struct {
		Score *Score `json:"score"`
	}
	var wrap struct {
		Series []struct {
			Score *Score `json:"score"`
		} `json:"series"`
	}
	if err := json.Unmarshal([]byte(t), &wrap); err == nil && len(wrap.Series) > 0 {
		for i := range wrap.Series {
			scored = append(scored, struct {
				Score *Score `json:"score"`
			}{wrap.Series[i].Score})
		}
	} else if err := json.Unmarshal([]byte(t), &scored); err != nil {
		return fmt.Errorf("parse scores: %w", err)
	}
	for i := range list {
		if i < len(scored) && scored[i].Score != nil {
			scored[i].Score.computeTotal()
			list[i].Score = scored[i].Score
		}
	}
	return nil
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slug derives a filesystem id from the series title.
func (s Series) Slug() string {
	out := slugRE.ReplaceAllString(strings.ToLower(s.Title), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		out = "series"
	}
	return out
}

// Slugify converts an arbitrary name to the same filesystem-safe slug Slug()
// uses for titles — lowercase, non-alphanumeric runs collapsed to single
// hyphens, edges trimmed. Used to derive a cast member's anchor-image filename
// (e.g. "Codex" -> "codex" -> anchors/codex.png).
func Slugify(name string) string {
	out := slugRE.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(out, "-")
}

// validVoices is the Kokoro voice set; unknown/typo'd voices 400 the TTS stage.
var validVoices = map[string]bool{
	"am_fenrir": true, "am_michael": true, "am_puck": true,
	"am_adam": true, "am_eric": true,
	"af_bella": true, "af_nicole": true, "bf_emma": true,
}

// NarratorVoice is the default + fallback for an unknown voice_id.
const NarratorVoice = "am_fenrir"

// NormalizeVoice returns v if known, else NarratorVoice.
func NormalizeVoice(v string) string {
	v = strings.TrimSpace(v)
	if validVoices[v] {
		return v
	}
	return NarratorVoice
}
