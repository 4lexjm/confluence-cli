package main

import (
	"os"

	"github.com/atlassian/confluence-cli/internal/build"
	"github.com/atlassian/confluence-cli/internal/confluencecmd"
)

func main() {
	os.Exit(confluencecmd.Main(build.Version))
}
