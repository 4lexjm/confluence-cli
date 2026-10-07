package root

import (
	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/pkg/cmd/api"
	"github.com/atlassian/confluence-cli/pkg/cmd/attachment"
	"github.com/atlassian/confluence-cli/pkg/cmd/auth"
	"github.com/atlassian/confluence-cli/pkg/cmd/blog"
	"github.com/atlassian/confluence-cli/pkg/cmd/comment"
	"github.com/atlassian/confluence-cli/pkg/cmd/context"
	"github.com/atlassian/confluence-cli/pkg/cmd/mcp"
	"github.com/atlassian/confluence-cli/pkg/cmd/page"
	"github.com/atlassian/confluence-cli/pkg/cmd/search"
	"github.com/atlassian/confluence-cli/pkg/cmd/space"
	"github.com/atlassian/confluence-cli/pkg/cmdutil"
)

// NewCmdRoot builds the root cobra command for confluence CLI.
func NewCmdRoot(f *cmdutil.Factory, version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "confluence",
		Short: "Confluence CLI for AI agents and humans",
		Long: `confluence is a unified CLI for Confluence Cloud and Confluence Data Center / Server.

Optimized for AI agents — providing structured JSON/YAML output, Markdown translation,
a built-in Model Context Protocol (MCP) server, and agent skills — while being equally
productive for humans in the terminal.

Authentication:
  Cloud API token:    confluence auth login https://myorg.atlassian.net --email me@example.com --token <TOKEN>
  DC Bearer PAT:      confluence auth login https://confluence.example.com --token <PAT>
  Browser SSO (DC):   confluence auth login https://confluence.example.com --browser
  Kerberos (DC+AD):   confluence auth login https://confluence.example.com --auth-method kerberos

Headless / CI Environment Variables:
  CONFLUENCE_HOST, CONFLUENCE_TOKEN, CONFLUENCE_EMAIL, CONFLUENCE_SPACE,
  CONFLUENCE_AUTH_METHOD, CONFLUENCE_SESSION_COOKIE`,
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
	}

	// Global persistent flags
	cmd.PersistentFlags().StringP("context", "c", "", "Active Confluence context name")
	cmd.PersistentFlags().StringP("space", "s", "", "Default Confluence space key")
	cmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	cmd.PersistentFlags().Bool("yaml", false, "Output in YAML format")
	cmd.PersistentFlags().String("jq", "", "Apply a jq expression to JSON output")
	cmd.PersistentFlags().Bool("debug", false, "Show HTTP request/response details (sets CONFLUENCE_HTTP_DEBUG=1)")

	// Subcommands
	cmd.AddCommand(auth.NewCmdAuth(f))
	cmd.AddCommand(context.NewCmdContext(f))
	cmd.AddCommand(space.NewCmdSpace(f))
	cmd.AddCommand(page.NewCmdPage(f))
	cmd.AddCommand(blog.NewCmdBlog(f))
	cmd.AddCommand(comment.NewCmdComment(f))
	cmd.AddCommand(attachment.NewCmdAttachment(f))
	cmd.AddCommand(search.NewCmdSearch(f))
	cmd.AddCommand(api.NewCmdAPI(f))
	cmd.AddCommand(mcp.NewCmdMCP(f))

	return cmd
}
