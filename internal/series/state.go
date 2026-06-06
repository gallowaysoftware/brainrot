package series

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Layout is the on-disk home:
//
//	$XDG_STATE_HOME/brainrot/
//	  series/<id>/series.json            the bible (cast + episode arc)
//	  series/<id>/episodes/<NNN>/...      per-episode script + render
//	  series/_rankings/<ts>.json          a scored ideate batch
type Layout struct{ Root string }

func DefaultRoot() (string, error) {
	root := ""
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		root = filepath.Join(d, "brainrot")
	} else if home, err := os.UserHomeDir(); err == nil {
		root = filepath.Join(home, ".local", "state", "brainrot")
	}
	// Refuse a relative root: with an empty HOME and no XDG_STATE_HOME we'd
	// otherwise scatter the library under the current working directory.
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("cannot locate library root: set XDG_STATE_HOME or HOME to an absolute path")
	}
	return root, nil
}

func Open() (Layout, error) {
	root, err := DefaultRoot()
	if err != nil {
		return Layout{}, err
	}
	l := Layout{Root: root}
	if err := os.MkdirAll(filepath.Join(l.Root, "series"), 0o755); err != nil {
		return Layout{}, err
	}
	return l, nil
}

func (l Layout) SeriesDir(id string) string  { return filepath.Join(l.Root, "series", id) }
func (l Layout) SeriesFile(id string) string { return filepath.Join(l.SeriesDir(id), "series.json") }

// LoreFile is the optional per-series canon pack (lore.md) that sits beside the
// bible. When present, the episode writer reads it so a lore-heavy source stays
// faithful and reaches the screen rather than being distilled away.
func (l Layout) LoreFile(id string) string { return filepath.Join(l.SeriesDir(id), "lore.md") }

// HasLore reports whether a series has a lore.md pack on disk.
func (l Layout) HasLore(id string) bool {
	_, err := os.Stat(l.LoreFile(id))
	return err == nil
}

// AnchorDir is the optional per-series directory of fixed character "anchor"
// images (anchors/<character-slug>.png). When a character has an anchor image,
// the render substitutes it verbatim for that character's shots instead of
// re-generating the look every episode — so a hand-made portrait (e.g. the
// Codex orb) stays pixel-consistent across the whole series.
func (l Layout) AnchorDir(id string) string { return filepath.Join(l.SeriesDir(id), "anchors") }

// AnchorFile is the anchor image path for one character slug.
func (l Layout) AnchorFile(id, slug string) string {
	return filepath.Join(l.AnchorDir(id), slug+".png")
}

// HasAnchor reports whether the given character slug has an anchor image on disk.
func (l Layout) HasAnchor(id, slug string) bool {
	_, err := os.Stat(l.AnchorFile(id, slug))
	return err == nil
}

func (l Layout) EpisodeDir(id string, n int) string {
	return filepath.Join(l.SeriesDir(id), "episodes", fmt.Sprintf("%03d", n))
}

func (l Layout) uniqueID(base string) string {
	if _, err := os.Stat(l.SeriesDir(base)); err != nil {
		return base
	}
	for i := 2; i < 10000; i++ {
		c := fmt.Sprintf("%s-%d", base, i)
		if _, err := os.Stat(l.SeriesDir(c)); err != nil {
			return c
		}
	}
	return base
}

// SaveSeries writes a kept series and returns its id.
func (l Layout) SaveSeries(s Series) (string, error) {
	id := l.uniqueID(s.Slug())
	if err := os.MkdirAll(l.SeriesDir(id), 0o755); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(l.SeriesFile(id), b, 0o644); err != nil {
		return "", err
	}
	return id, nil
}

// AppendEpisodes loads a series, appends beats to its episode arc, and saves it
// in place. Additive and non-destructive — existing episodes are untouched.
// Returns the new total episode count.
func AppendEpisodes(l Layout, id string, beats []EpisodeBeat) (int, error) {
	s, err := LoadSeries(l, id)
	if err != nil {
		return 0, err
	}
	s.Episodes = append(s.Episodes, beats...)
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(l.SeriesFile(id), b, 0o644); err != nil {
		return 0, err
	}
	return len(s.Episodes), nil
}

func LoadSeries(l Layout, id string) (Series, error) {
	b, err := os.ReadFile(l.SeriesFile(id))
	if err != nil {
		return Series{}, err
	}
	var s Series
	if err := json.Unmarshal(b, &s); err != nil {
		return Series{}, fmt.Errorf("parse %s: %w", l.SeriesFile(id), err)
	}
	return s, nil
}

func (l Layout) SaveRanking(stamp string, list []Series) error {
	dir := filepath.Join(l.Root, "series", "_rankings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, stamp+".json"), b, 0o644)
}

func ListSeries(l Layout) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(l.Root, "series"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		// Skip the studio's bookkeeping dirs (_rankings, _dev, ...): they sit
		// beside series dirs but aren't series ids.
		if e.IsDir() && !strings.HasPrefix(e.Name(), "_") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// NextEpisode returns the smallest 1-indexed episode without a final.mp4.
func NextEpisode(l Layout, id string) int {
	for n := 1; n <= 999; n++ {
		if _, err := os.Stat(filepath.Join(l.EpisodeDir(id, n), "final.mp4")); err != nil {
			return n
		}
	}
	return 1
}

func SortByScore(list []Series) {
	sort.SliceStable(list, func(a, b int) bool {
		return list[a].Score.Rank() > list[b].Score.Rank()
	})
}
