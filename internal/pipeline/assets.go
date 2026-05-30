// Package pipeline holds brainrot's vamp pipelines plus the embedded prompts
// and ComfyUI workflows. The CLI (cmd/brainrot) drives them.
package pipeline

import (
	"embed"
	"io/fs"
)

//go:embed prompts/*.md workflows/*.json
var assets embed.FS

// PromptsFS / WorkflowsFS narrow the embed so callers reference files by bare
// name via PromptFS / WorkflowFS.
var (
	PromptsFS   fs.FS = mustSub(assets, "prompts")
	WorkflowsFS fs.FS = mustSub(assets, "workflows")
)

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
