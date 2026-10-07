package attachment

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/pkg/cmdutil"
)

// NewCmdAttachment returns the attachment command group.
func NewCmdAttachment(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachment",
		Short: "Manage attachments on Confluence pages",
		Long:  `List, upload, download, and delete attachments on Confluence pages.`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newUploadCmd(f))
	cmd.AddCommand(newDownloadCmd(f))
	cmd.AddCommand(newDeleteCmd(f))

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "list <page-id>",
		Aliases: []string{"ls"},
		Short:   "List attachments on a page",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			resp, err := cl.ListAttachments(cmd.Context(), pageID, limit, 0)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, resp.Results, func() error {
				if len(resp.Results) == 0 {
					fmt.Fprintln(ios.Out, "No attachments found.")
					return nil
				}
				fmt.Fprintf(ios.Out, "%-15s %-35s %-12s %-5s\n", "ID", "FILENAME", "SIZE", "VER")
				fmt.Fprintf(ios.Out, "%-15s %-35s %-12s %-5s\n", "--", "--------", "----", "---")
				for _, a := range resp.Results {
					sz := "unknown"
					if a.Extensions != nil && a.Extensions.FileSize > 0 {
						sz = formatFileSize(a.Extensions.FileSize)
					}
					v := 1
					if a.Version != nil {
						v = a.Version.Number
					}
					fmt.Fprintf(ios.Out, "%-15s %-35s %-12s %-5d\n", a.ID, a.Title, sz, v)
				}
				return nil
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 25, "Maximum attachments to return")
	return cmd
}

func newUploadCmd(f *cmdutil.Factory) *cobra.Command {
	var filePath, customName, comment string

	cmd := &cobra.Command{
		Use:   "upload <page-id>",
		Short: "Upload a file as attachment to a page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			if filePath == "" {
				return errors.New("--file is required")
			}

			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("open file %q: %w", filePath, err)
			}
			defer file.Close()

			filename := customName
			if filename == "" {
				filename = filepath.Base(filePath)
			}

			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			resp, err := cl.UploadAttachment(cmd.Context(), pageID, filename, file, comment)
			if err != nil {
				return err
			}

			ios, _ := f.Streams()
			return cmdutil.PrintData(cmd, ios.Out, resp.Results, func() error {
				fmt.Fprintf(ios.Out, "Uploaded %q to page %s\n", filename, pageID)
				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the file to upload (required)")
	cmd.Flags().StringVarP(&customName, "name", "n", "", "Custom attachment filename")
	cmd.Flags().StringVarP(&comment, "comment", "m", "", "Attachment comment or version note")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func newDownloadCmd(f *cmdutil.Factory) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "download <attachment-id>",
		Short: "Download an attachment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attID := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			// Fetch attachment details to get download link and title
			attContent, err := cl.GetPage(cmd.Context(), attID, "", "")
			if err != nil {
				return fmt.Errorf("fetch attachment metadata: %w", err)
			}

			downloadLink := ""
			if attContent.Links != nil {
				downloadLink = attContent.Links["download"]
			}
			if downloadLink == "" {
				downloadLink = fmt.Sprintf("/download/attachments/%s/%s", attID, attContent.Title)
			}

			destPath := outputPath
			if destPath == "" {
				destPath = attContent.Title
			}

			outFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("create output file %q: %w", destPath, err)
			}
			defer outFile.Close()

			if err := cl.DownloadAttachment(cmd.Context(), downloadLink, outFile); err != nil {
				return err
			}

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Downloaded %s to %s\n", attContent.Title, destPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Destination file path (defaults to attachment title)")
	return cmd
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <attachment-id>",
		Short: "Delete an attachment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attID := args[0]
			if !confirm {
				return errors.New("confirm deletion with --confirm")
			}

			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			if err := cl.DeleteAttachment(cmd.Context(), attID); err != nil {
				return err
			}

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Deleted attachment %s\n", attID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm deletion")
	return cmd
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
