package blog

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/pkg/cmdutil"
	"github.com/atlassian/confluence-cli/pkg/markdown"
)

// NewCmdBlog returns the blog command group.
func NewCmdBlog(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blog",
		Short: "Work with Confluence blog posts",
		Long:  `List, view, and create Confluence blog posts.`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newCreateCmd(f))

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List blog posts in a space",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			spaceKey, _ := f.ResolveSpace(cmd, false)
			resp, err := cl.ListBlogPosts(cmd.Context(), spaceKey, limit, 0)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, resp.Results, func() error {
				if len(resp.Results) == 0 {
					fmt.Fprintln(ios.Out, "No blog posts found.")
					return nil
				}
				fmt.Fprintf(ios.Out, "%-12s %-8s %-45s\n", "ID", "SPACE", "TITLE")
				fmt.Fprintf(ios.Out, "%-12s %-8s %-45s\n", "--", "-----", "-----")
				for _, p := range resp.Results {
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

	cmd.Flags().IntVarP(&limit, "limit", "L", 25, "Maximum blog posts to return")
	return cmd
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	var format string
	var web bool

	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View a blog post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			post, err := cl.GetPage(cmd.Context(), id, "", "")
			if err != nil {
				return err
			}

			if web {
				webui := post.WebUIURL(cl.BaseURL())
				if webui != "" {
					return f.BrowserOpener().Open(webui)
				}
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			bodyRaw := ""
			if post.Body != nil {
				if format == "storage" && post.Body.Storage != nil {
					bodyRaw = post.Body.Storage.Value
				} else if format == "view" && post.Body.View != nil {
					bodyRaw = post.Body.View.Value
				} else {
					if post.Body.Storage != nil {
						bodyRaw = markdown.StorageToMarkdown(post.Body.Storage.Value)
					}
				}
			}

			viewData := map[string]any{
				"id":      post.ID,
				"title":   post.Title,
				"space":   post.Space,
				"version": post.Version,
				"url":     post.WebUIURL(cl.BaseURL()),
				"body":    bodyRaw,
			}

			return cmdutil.PrintData(cmd, ios.Out, viewData, func() error {
				fmt.Fprintf(ios.Out, "# %s\n\n", post.Title)
				fmt.Fprintf(ios.Out, "ID:      %s\n", post.ID)
				if post.Space != nil {
					fmt.Fprintf(ios.Out, "Space:   %s\n", post.Space.Key)
				}
				if u := post.WebUIURL(cl.BaseURL()); u != "" {
					fmt.Fprintf(ios.Out, "URL:     %s\n", u)
				}
				fmt.Fprintf(ios.Out, "\n---\n\n")
				fmt.Fprintln(ios.Out, bodyRaw)
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&format, "format", "markdown", "Body format: markdown, storage, view")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open blog post in web browser")
	return cmd
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var title, body, filePath, format string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new blog post",
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
					return fmt.Errorf("read file: %w", err)
				}
				content = string(data)
			}

			storageBody := content
			if format == "markdown" || format == "" {
				storageBody = markdown.MarkdownToStorage(content)
			}

			created, err := cl.CreateBlogPost(cmd.Context(), spaceKey, title, storageBody)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, created, func() error {
				fmt.Fprintf(ios.Out, "Created blog post %q (ID: %s)\n", created.Title, created.ID)
				if u := created.WebUIURL(cl.BaseURL()); u != "" {
					fmt.Fprintf(ios.Out, "URL: %s\n", u)
				}
				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Blog post title (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Blog post body")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Read body from file")
	cmd.Flags().StringVar(&format, "format", "markdown", "Body format: markdown (default) or storage")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}
