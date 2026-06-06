module github.com/gallowaysoftware/brainrot

go 1.26.3

require (
	github.com/gallowaysoftware/vibe v0.6.2
	github.com/spf13/cobra v1.10.2
)

require (
	connectrpc.com/connect v1.19.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.36.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// brainrot builds against an in-tree vibe checkout: clone
// github.com/gallowaysoftware/vibe as a sibling directory (../vibe). The
// require version above is a floor; the local checkout is what actually
// builds, since brainrot tracks vibe's contentkit/vamp packages.
replace github.com/gallowaysoftware/vibe => ../vibe
