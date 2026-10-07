// Package confluencecmd wires the root Cobra command and factory together.
package confluencecmd

import (
	"os"

	"github.com/atlassian/confluence-cli/internal/config"
	"github.com/atlassian/confluence-cli/pkg/cmd/root"
	"github.com/atlassian/confluence-cli/pkg/cmdutil"
	"github.com/atlassian/confluence-cli/pkg/iostreams"
)

// Main is the CLI entry point called from cmd/confluence/main.go.
func Main(version string) int {
	ios := iostreams.System()

	f := &cmdutil.Factory{
		AppVersion:     version,
		ExecutableName: "confluence",
		IOStreams:      ios,
		Config:         config.Load,
	}

	if os.Getenv("CONFLUENCE_HTTP_DEBUG") == "1" {
		os.Setenv("CONFLUENCE_HTTP_DEBUG", "1")
	}

	rootCmd := root.NewCmdRoot(f, version)

	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}
