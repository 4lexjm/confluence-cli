package mcp

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/internal/mcpserver"
	"github.com/atlassian/confluence-cli/pkg/cmdutil"
)

// NewCmdMCP returns the mcp command group.
func NewCmdMCP(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol (MCP) server for AI agents",
		Long: `Start an MCP server to expose Confluence tools directly to AI coding assistants
such as Claude Code, Cursor, Gemini CLI, and GitHub Copilot.`,
	}

	cmd.AddCommand(newServeCmd(f))
	return cmd
}

func newServeCmd(f *cmdutil.Factory) *cobra.Command {
	var readOnly bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Confluence MCP stdio server",
		Long: `Start the MCP server over stdio. AI agents launch this command to interact
with Confluence tools via JSON-RPC 2.0.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return fmt.Errorf("failed to initialize Confluence client: %w", err)
			}

			spaceKey, _ := f.ResolveSpace(cmd, false)

			server := mcpserver.New(cl, readOnly, spaceKey, f.AppVersion)

			// MCP protocol communicates via standard input/output
			return server.Serve(cmd.Context(), os.Stdin, os.Stdout)
		},
	}

	cmd.Flags().BoolVar(&readOnly, "read-only", false, "Expose only read-only tools to agents")
	return cmd
}
