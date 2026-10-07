package comment

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/pkg/cmdutil"
	"github.com/atlassian/confluence-cli/pkg/markdown"
)

// NewCmdComment returns the comment command group.
func NewCmdComment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage comments on Confluence pages",
		Long:  `List, add, edit, and delete comments on Confluence pages.`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newAddCmd(f))
	cmd.AddCommand(newEditCmd(f))
	cmd.AddCommand(newDeleteCmd(f))

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "list <page-id>",
		Aliases: []string{"ls"},
		Short:   "List comments on a page",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			resp, err := cl.ListComments(cmd.Context(), pageID, limit, 0)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, resp.Results, func() error {
				if len(resp.Results) == 0 {
					fmt.Fprintln(ios.Out, "No comments found on this page.")
					return nil
				}
				for _, c := range resp.Results {
					author := "unknown"
					if c.Version != nil && c.Version.By != nil && c.Version.By.DisplayName != "" {
						author = c.Version.By.DisplayName
					}
					bodyText := ""
					if c.Body != nil && c.Body.Storage != nil {
						bodyText = markdown.StorageToMarkdown(c.Body.Storage.Value)
					} else if c.Body != nil && c.Body.View != nil {
						bodyText = markdown.StorageToMarkdown(c.Body.View.Value)
					}
					fmt.Fprintf(ios.Out, "[ID: %s] by %s:\n%s\n\n", c.ID, author, bodyText)
				}
				return nil
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 25, "Maximum comments to return")
	return cmd
}

func newAddCmd(f *cmdutil.Factory) *cobra.Command {
	var body, filePath, parentCommentID string

	cmd := &cobra.Command{
		Use:   "add <page-id>",
		Short: "Add a comment to a page",
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
					return fmt.Errorf("read file: %w", err)
				}
				content = string(data)
			}

			if content == "" {
				return errors.New("comment body is required (use --body or --file)")
			}

			storageBody := markdown.MarkdownToStorage(content)
			created, err := cl.AddComment(cmd.Context(), pageID, storageBody, parentCommentID)
			if err != nil {
				return err
			}

			ios, _ := f.Streams()
			return cmdutil.PrintData(cmd, ios.Out, created, func() error {
				fmt.Fprintf(ios.Out, "Added comment %s to page %s\n", created.ID, pageID)
				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body text (Markdown supported)")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Read comment body from file")
	cmd.Flags().StringVar(&parentCommentID, "parent", "", "Parent comment ID for threaded replies")
	return cmd
}

func newEditCmd(f *cmdutil.Factory) *cobra.Command {
	var body, filePath string

	cmd := &cobra.Command{
		Use:   "edit <comment-id>",
		Short: "Edit a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			commentID := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			content := body
			if filePath != "" {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("read file: %w", err)
				}
				content = string(data)
			}

			if content == "" {
				return errors.New("comment body is required (use --body or --file)")
			}

			storageBody := markdown.MarkdownToStorage(content)
			updated, err := cl.UpdateComment(cmd.Context(), commentID, storageBody, 0)
			if err != nil {
				return err
			}

			ios, _ := f.Streams()
			return cmdutil.PrintData(cmd, ios.Out, updated, func() error {
				fmt.Fprintf(ios.Out, "Updated comment %s\n", commentID)
				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "New comment body")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Read new comment body from file")
	return cmd
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <comment-id>",
		Short: "Delete a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			commentID := args[0]
			if !confirm {
				return errors.New("confirm deletion with --confirm")
			}

			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			if err := cl.DeleteComment(cmd.Context(), commentID); err != nil {
				return err
			}

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Deleted comment %s\n", commentID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm deletion")
	return cmd
}
