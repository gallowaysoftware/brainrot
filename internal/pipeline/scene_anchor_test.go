package pipeline

import "testing"

// The anchor path adds split + passthrough stages. Exercise vamp's Build()
// validation (DAG wiring, foreach sources, output templates) for both builders,
// with and without an anchors dir, so a structural regression fails here rather
// than at render time on the GPU box.
func TestSceneBuildsWithAndWithoutAnchorDir(t *testing.T) {
	cases := []struct {
		name      string
		anchorDir string
	}{
		{"no-anchor", ""},
		{"with-anchor", "/tmp/series/goblin-town/anchors"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := SceneConfig{Shots: 4, ShotsFile: "/tmp/shots.json", AnchorDir: tc.anchorDir}
			if _, err := BuildSceneRender(cfg); err != nil {
				t.Fatalf("BuildSceneRender(%q): %v", tc.anchorDir, err)
			}
			if _, err := BuildScenePreview(cfg); err != nil {
				t.Fatalf("BuildScenePreview(%q): %v", tc.anchorDir, err)
			}
		})
	}
}
