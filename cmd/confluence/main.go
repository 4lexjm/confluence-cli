package main

import (
	"os"

	"github.com/atlassian/confluence-cli/internal/confluencecmd"
)

var version = "dev"

func main() {
	os.Exit(confluencecmd.Main(version))
}
