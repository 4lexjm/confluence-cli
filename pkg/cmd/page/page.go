package page

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/pkg/cmdutil"
	"github.com/atlassian/confluence-cli/pkg/confluence"
	"github.com/atlassian/confluence-cli/pkg/markdown"
)

// NewCmdPage returns the page command group.
func NewCmdPage(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page",
		Short: "Work with Confluence pages",
		Long:  `Create, view, edit, search, move, and manage Confluence pages and labels.`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newEditCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newSearchCmd(f))
	cmd.AddCommand(newHistoryCmd(f))
	cmd.AddCommand(newMoveCmd(f))
	cmd.AddCommand(newCopyCmd(f))
	cmd.AddCommand(newLabelCmd(f))

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var limit int
	var title string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pages in a space",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			spaceKey, _ := f.ResolveSpace(cmd, false)
			resp, err := cl.ListPages(cmd.Context(), spaceKey, limit, 0, title)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, resp.Results, func() error {
				if len(resp.Results) == 0 {
					fmt.Fprintln(ios.Out, "No pages found.")
					return nil
				}
				fmt.Fprintf(ios.Out, "%-12s %-6s %-45s %-5s\n", "ID", "SPACE", "TITLE", "VER")
				fmt.Fprintf(ios.Out, "%-12s %-6s %-45s %-5s\n", "--", "-----", "-----", "---")
				for _, p := range resp.Results {
					sp := ""
					if p.Space != nil {
						sp = p.Space.Key
					}
					v := 1
					if p.Version != nil {
						v = p.Version.Number
					}
					fmt.Fprintf(ios.Out, "%-12s %-6s %-45s %-5d\n", p.ID, sp, p.Title, v)
				}
				return nil
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 25, "Maximum number of pages to return")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Filter by page title")
	return cmd
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	var format string
	var web bool
	var showComments bool

	cmd := &cobra.Command{
		Use:   "view <id-or-title>",
		Short: "View a page content and metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrTitle := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			spaceKey, _ := f.ResolveSpace(cmd, false)
			page, err := cl.GetPage(cmd.Context(), idOrTitle, spaceKey, "")
			if err != nil {
				return err
			}

			if web {
				webui := page.WebUIURL(cl.BaseURL())
				if webui != "" {
					return f.BrowserOpener().Open(webui)
				}
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			// Format body
			bodyRaw := ""
			if page.Body != nil {
				if format == "storage" && page.Body.Storage != nil {
					bodyRaw = page.Body.Storage.Value
				} else if format == "view" && page.Body.View != nil {
					bodyRaw = page.Body.View.Value
				} else {
					// Default to markdown
					if page.Body.Storage != nil {
						bodyRaw = markdown.StorageToMarkdown(page.Body.Storage.Value)
					} else if page.Body.View != nil {
						bodyRaw = markdown.StorageToMarkdown(page.Body.View.Value)
					}
				}
			}

			viewData := map[string]any{
				"id":        page.ID,
				"title":     page.Title,
				"space":     page.Space,
				"version":   page.Version,
				"url":       page.WebUIURL(cl.BaseURL()),
				"body":      bodyRaw,
				"format":    format,
				"ancestors": page.Ancestors,
			}

			if showComments {
				comments, err := cl.ListComments(cmd.Context(), page.ID, 50, 0)
				if err == nil {
					viewData["comments"] = comments.Results
				}
			}

			return cmdutil.PrintData(cmd, ios.Out, viewData, func() error {
				fmt.Fprintf(ios.Out, "# %s\n\n", page.Title)
				fmt.Fprintf(ios.Out, "ID:      %s\n", page.ID)
				if page.Space != nil {
					fmt.Fprintf(ios.Out, "Space:   %s\n", page.Space.Key)
				}
				if page.Version != nil {
					fmt.Fprintf(ios.Out, "Version: %d (updated by %s)\n", page.Version.Number, formatUser(page.Version.By))
				}
				if u := page.WebUIURL(cl.BaseURL()); u != "" {
					fmt.Fprintf(ios.Out, "URL:     %s\n", u)
				}
				fmt.Fprintf(ios.Out, "\n---\n\n")
				fmt.Fprintln(ios.Out, bodyRaw)

				if showComments {
					comments, ok := viewData["comments"].([]*confluence.Content)
					if ok && len(comments) > 0 {
						fmt.Fprintf(ios.Out, "\n---\n### Comments (%d)\n\n", len(comments))
						for _, c := range comments {
							commentBody := ""
							if c.Body != nil && c.Body.Storage != nil {
								commentBody = markdown.StorageToMarkdown(c.Body.Storage.Value)
							}
							fmt.Fprintf(ios.Out, "- **Comment %s** (by %s):\n  %s\n\n", c.ID, formatUser(c.Version.By), strings.ReplaceAll(commentBody, "\n", "\n  "))
						}
					}
				}

				return nil
			})
		},
	}

	cmd.Flags().StringVar(&format, "format", "markdown", "Body format: markdown, storage, view")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open page in browser")
	cmd.Flags().BoolVar(&showComments, "comments", false, "Include comments in output")
	return cmd
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var title, body, filePath, parentID, format string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Confluence page",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			spaceKey, err := f.ResolveSpace(cmd, true)
			if err != nil {
				return err
			}

			if title == "" {
				return errors.New("--title is required")
			}

			content := body
			if filePath != "" {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("read file %q: %w", filePath, err)
				}
				content = string(data)
			}

			storageBody := content
			if format == "markdown" || format == "" {
				storageBody = markdown.MarkdownToStorage(content)
			}

			created, err := cl.CreatePage(cmd.Context(), spaceKey, title, storageBody, parentID)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, created, func() error {
				fmt.Fprintf(ios.Out, "Created page %q (ID: %s)\n", created.Title, created.ID)
				if u := created.WebUIURL(cl.BaseURL()); u != "" {
					fmt.Fprintf(ios.Out, "URL: %s\n", u)
				}
				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Page title (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Page body content")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Read body from file")
	cmd.Flags().StringVar(&parentID, "parent", "", "Parent page ID")
	cmd.Flags().StringVar(&format, "format", "markdown", "Body format: markdown (default) or storage")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func newEditCmd(f *cmdutil.Factory) *cobra.Command {
	var title, body, filePath, versionMsg, format string
	var versionNum int

	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit an existing page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			content := body
			if filePath != "" {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("read file %q: %w", filePath, err)
				}
				content = string(data)
			}

			storageBody := content
			if content != "" && (format == "markdown" || format == "") {
				storageBody = markdown.MarkdownToStorage(content)
			}

			// If no body provided, keep existing body
			if content == "" {
				curr, err := cl.GetPage(cmd.Context(), pageID, "", "body.storage,version,space")
				if err != nil {
					return fmt.Errorf("fetch page: %w", err)
				}
				if curr.Body != nil && curr.Body.Storage != nil {
					storageBody = curr.Body.Storage.Value
				}
			}

			updated, err := cl.UpdatePage(cmd.Context(), pageID, title, storageBody, versionNum, versionMsg)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, updated, func() error {
				fmt.Fprintf(ios.Out, "Updated page %q (ID: %s, Version: %d)\n", updated.Title, updated.ID, updated.Version.Number)
				if u := updated.WebUIURL(cl.BaseURL()); u != "" {
					fmt.Fprintf(ios.Out, "URL: %s\n", u)
				}
				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "New page title")
	cmd.Flags().StringVarP(&body, "body", "b", "", "New page body")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Read new body from file")
	cmd.Flags().StringVarP(&versionMsg, "version-comment", "m", "", "Version change comment")
	cmd.Flags().IntVar(&versionNum, "version-num", 0, "Explicit version number")
	cmd.Flags().StringVar(&format, "format", "markdown", "Body format: markdown (default) or storage")

	return cmd
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			if !confirm {
				return errors.New("deleting a page requires confirmation: pass --confirm")
			}

			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			if err := cl.DeletePage(cmd.Context(), pageID); err != nil {
				return err
			}

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Deleted page %s\n", pageID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm page deletion")
	return cmd
}

func newSearchCmd(f *cmdutil.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search pages in Confluence",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			spaceKey, _ := f.ResolveSpace(cmd, false)
			res, err := cl.Search(cmd.Context(), query, spaceKey, "page", limit)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, res.Results, func() error {
				if len(res.Results) == 0 {
					fmt.Fprintln(ios.Out, "No matching pages found.")
					return nil
				}
				fmt.Fprintf(ios.Out, "%-12s %-8s %-45s\n", "ID", "SPACE", "TITLE")
				fmt.Fprintf(ios.Out, "%-12s %-8s %-45s\n", "--", "-----", "-----")
				for _, p := range res.Results {
					sp := ""
					if p.Space != nil {
						sp = p.Space.Key
					}
					fmt.Fprintf(ios.Out, "%-12s %-8s %-45s\n", p.ID, sp, p.Title)
				}
				return nil
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 25, "Maximum results to return")
	return cmd
}

func newHistoryCmd(f *cmdutil.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "history <id>",
		Short: "View version history for a page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			hist, err := cl.GetPageHistory(cmd.Context(), pageID, limit)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, hist.Results, func() error {
				if len(hist.Results) == 0 {
					fmt.Fprintln(ios.Out, "No history available.")
					return nil
				}
				fmt.Fprintf(ios.Out, "%-6s %-25s %-20s %-30s\n", "VER", "WHEN", "BY", "MESSAGE")
				fmt.Fprintf(ios.Out, "%-6s %-25s %-20s %-30s\n", "---", "----", "--", "-------")
				for _, v := range hist.Results {
					fmt.Fprintf(ios.Out, "%-6d %-25s %-20s %-30s\n", v.Number, v.When, formatUser(v.By), v.Message)
				}
				return nil
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 20, "Maximum version history entries")
	return cmd
}

func newMoveCmd(f *cmdutil.Factory) *cobra.Command {
	var parentID, targetSpace string

	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move a page to a new parent or space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			if parentID == "" && targetSpace == "" {
				return errors.New("must specify --parent or --space")
			}

			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			moved, err := cl.MovePage(cmd.Context(), pageID, parentID, targetSpace)
			if err != nil {
				return err
			}

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Moved page %s (%s)\n", moved.ID, moved.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Target parent page ID")
	cmd.Flags().StringVarP(&targetSpace, "space", "s", "", "Target space key")
	return cmd
}

func newCopyCmd(f *cmdutil.Factory) *cobra.Command {
	var newTitle, targetSpace string

	cmd := &cobra.Command{
		Use:   "copy <id>",
		Short: "Duplicate an existing page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			copied, err := cl.CopyPage(cmd.Context(), pageID, newTitle, targetSpace)
			if err != nil {
				return err
			}

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Copied page %s to new page %s (%s)\n", pageID, copied.ID, copied.Title)
			return nil
		},
	}

	cmd.Flags().StringVarP(&newTitle, "title", "t", "", "New page title")
	cmd.Flags().StringVarP(&targetSpace, "space", "s", "", "Target space key")
	return cmd
}

func newLabelCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "Manage labels on a page",
	}

	// list
	cmd.AddCommand(&cobra.Command{
		Use:   "list <id>",
		Short: "List labels on a page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}
			labels, err := cl.ListLabels(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			ios, _ := f.Streams()
			return cmdutil.PrintData(cmd, ios.Out, labels.Results, func() error {
				if len(labels.Results) == 0 {
					fmt.Fprintln(ios.Out, "No labels found.")
					return nil
				}
				for _, l := range labels.Results {
					fmt.Fprintf(ios.Out, "- %s\n", l.Name)
				}
				return nil
			})
		},
	})

	// add
	cmd.AddCommand(&cobra.Command{
		Use:   "add <id> <label...>",
		Short: "Add one or more labels to a page",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}
			pageID := args[0]
			labels := args[1:]
			if err := cl.AddLabels(cmd.Context(), pageID, labels); err != nil {
				return err
			}
			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Added labels %v to page %s\n", labels, pageID)
			return nil
		},
	})

	// remove
	cmd.AddCommand(&cobra.Command{
		Use:   "remove <id> <label>",
		Short: "Remove a label from a page",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}
			pageID := args[0]
			label := args[1]
			if err := cl.RemoveLabel(cmd.Context(), pageID, label); err != nil {
				return err
			}
			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Removed label %q from page %s\n", label, pageID)
			return nil
		},
	})

	return cmd
}

func formatUser(u *confluence.User) string {
	if u == nil {
		return "unknown"
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.Username != "" {
		return u.Username
	}
	if u.AccountID != "" {
		return u.AccountID
	}
	return "unknown"
}
