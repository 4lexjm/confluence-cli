package search

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/pkg/cmdutil"
	"github.com/atlassian/confluence-cli/pkg/confluence"
)

// NewCmdSearch returns the search command.
func NewCmdSearch(f *cmdutil.Factory) *cobra.Command {
	var cqlQuery string
	var contentType string
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search Confluence content via text or CQL",
		Long: `Search pages, blog posts, and attachments across Confluence.

Supports simple free-text queries or advanced Confluence Query Language (CQL):
  confluence search "architecture"
  confluence search --cql 'space = "ENG" AND type = page AND text ~ "design"'
  confluence search --type blogpost --space ENG`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			spaceKey, _ := f.ResolveSpace(cmd, false)

			var res *confluence.SearchResult
			if cqlQuery != "" {
				res, err = cl.SearchCQL(cmd.Context(), cqlQuery, limit, 0)
			} else {
				textQuery := ""
				if len(args) > 0 {
					textQuery = strings.Join(args, " ")
				}
				res, err = cl.Search(cmd.Context(), textQuery, spaceKey, contentType, limit)
			}

			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, res.Results, func() error {
				if len(res.Results) == 0 {
					fmt.Fprintln(ios.Out, "No matching content found.")
					return nil
				}
				fmt.Fprintf(ios.Out, "%-12s %-10s %-8s %-45s\n", "ID", "TYPE", "SPACE", "TITLE")
				fmt.Fprintf(ios.Out, "%-12s %-10s %-8s %-45s\n", "--", "----", "-----", "-----")
				for _, it := range res.Results {
					sp := ""
					if it.Space != nil {
						sp = it.Space.Key
					}
					fmt.Fprintf(ios.Out, "%-12s %-10s %-8s %-45s\n", it.ID, it.Type, sp, it.Title)
				}
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&cqlQuery, "cql", "", "Confluence Query Language (CQL) expression")
	cmd.Flags().StringVarP(&contentType, "type", "T", "", "Filter by content type: page, blogpost, attachment, comment")
	cmd.Flags().IntVarP(&limit, "limit", "L", 25, "Maximum number of search results to return")

	return cmd
}
