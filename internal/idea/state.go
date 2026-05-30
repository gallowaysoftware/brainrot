package idea

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Layout is the on-disk home for brainrot output:
//
//	$XDG_STATE_HOME/brainrot/
//	  ideas/<id>/idea.json        one kept content seed
//	  ideas/_rankings/<ts>.json   full scored batch from one ideate run
//	  videos/<id>/<NNN>/final.mp4 depth output per idea
type Layout struct{ Root string }

// DefaultRoot returns $XDG_STATE_HOME/brainrot or ~/.local/state/brainrot.
func DefaultRoot() string {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "brainrot")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "brainrot"
	}
	return filepath.Join(home, ".local", "state", "brainrot")
}

// Open returns the Layout, creating the directory tree.
func Open() (Layout, error) {
	l := Layout{Root: DefaultRoot()}
	for _, sub := range []string{"ideas", "videos"} {
		if err := os.MkdirAll(filepath.Join(l.Root, sub), 0o755); err != nil {
			return Layout{}, err
		}
	}
	return l, nil
}

func (l Layout) IdeasDir() string          { return filepath.Join(l.Root, "ideas") }
func (l Layout) VideosDir() string         { return filepath.Join(l.Root, "videos") }
func (l Layout) IdeaDir(id string) string  { return filepath.Join(l.IdeasDir(), id) }
func (l Layout) IdeaFile(id string) string { return filepath.Join(l.IdeaDir(id), "idea.json") }
func (l Layout) VideoDir(id string, n int) string {
	return filepath.Join(l.VideosDir(), id, fmt.Sprintf("%03d", n))
}

// uniqueID appends -2, -3, ... if an idea dir already exists, so repeated
// ideate runs don't clobber prior kept ideas.
func (l Layout) uniqueID(base string) string {
	if _, err := os.Stat(l.IdeaDir(base)); err != nil {
		return base
	}
	for i := 2; i < 10000; i++ {
		cand := fmt.Sprintf("%s-%d", base, i)
		if _, err := os.Stat(l.IdeaDir(cand)); err != nil {
			return cand
		}
	}
	return base
}

// SaveIdea writes one kept idea to ideas/<id>/idea.json and returns its id.
func (l Layout) SaveIdea(i Idea) (string, error) {
	id := l.uniqueID(i.Slug())
	if err := os.MkdirAll(l.IdeaDir(id), 0o755); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(l.IdeaFile(id), b, 0o644); err != nil {
		return "", err
	}
	return id, nil
}

// SaveRanking persists the full scored batch (newest-first by total) under a
// stamped filename so a run's complete shortlist stays inspectable.
func (l Layout) SaveRanking(stamp string, ideas []Idea) error {
	dir := filepath.Join(l.IdeasDir(), "_rankings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ideas, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, stamp+".json"), b, 0o644)
}

// LoadIdea reads a persisted idea by id.
func LoadIdea(l Layout, id string) (Idea, error) {
	b, err := os.ReadFile(l.IdeaFile(id))
	if err != nil {
		return Idea{}, err
	}
	var i Idea
	if err := json.Unmarshal(b, &i); err != nil {
		return Idea{}, fmt.Errorf("parse %s: %w", l.IdeaFile(id), err)
	}
	return i, nil
}

// ListIdeas returns the ids of every kept idea, sorted.
func ListIdeas(l Layout) ([]string, error) {
	entries, err := os.ReadDir(l.IdeasDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "_rankings" {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// NextVideo returns the smallest 1-indexed video number for an idea without a
// finished final.mp4.
func NextVideo(l Layout, id string) int {
	for n := 1; n <= 999; n++ {
		if _, err := os.Stat(filepath.Join(l.VideoDir(id, n), "final.mp4")); err != nil {
			return n
		}
	}
	return 1
}

// SortByScore orders ideas highest Total first (unscored sink to the bottom).
func SortByScore(ideas []Idea) {
	sort.SliceStable(ideas, func(a, b int) bool {
		ta, tb := 0, 0
		if ideas[a].Score != nil {
			ta = ideas[a].Score.Total
		}
		if ideas[b].Score != nil {
			tb = ideas[b].Score.Total
		}
		return ta > tb
	})
}
