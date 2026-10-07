package space

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/pkg/cmdutil"
)

// NewCmdSpace returns the space command group.
func NewCmdSpace(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "space",
		Short: "Work with Confluence spaces",
		Long:  `List, view, create, and delete Confluence spaces.`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newDeleteCmd(f))

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var limit int
	var spaceType string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Confluence spaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			resp, err := cl.ListSpaces(cmd.Context(), limit, 0, spaceType)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, resp.Results, func() error {
				if len(resp.Results) == 0 {
					fmt.Fprintln(ios.Out, "No spaces found.")
					return nil
				}
				fmt.Fprintf(ios.Out, "%-10s %-30s %-10s\n", "KEY", "NAME", "TYPE")
				fmt.Fprintf(ios.Out, "%-10s %-30s %-10s\n", "---", "----", "----")
				for _, s := range resp.Results {
					fmt.Fprintf(ios.Out, "%-10s %-30s %-10s\n", s.Key, s.Name, s.Type)
				}
				return nil
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 25, "Maximum number of spaces to return")
	cmd.Flags().StringVarP(&spaceType, "type", "T", "", "Filter by space type (global, personal)")

	return cmd
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool

	cmd := &cobra.Command{
		Use:   "view <key>",
		Short: "View details of a space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			space, err := cl.GetSpace(cmd.Context(), key)
			if err != nil {
				return err
			}

			if web {
				webui := space.Links["webui"]
				if webui != "" {
					url := cl.BaseURL() + webui
					return f.BrowserOpener().Open(url)
				}
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, space, func() error {
				fmt.Fprintf(ios.Out, "Key:         %s\n", space.Key)
				fmt.Fprintf(ios.Out, "Name:        %s\n", space.Name)
				fmt.Fprintf(ios.Out, "Type:        %s\n", space.Type)
				if space.Homepage != nil {
					fmt.Fprintf(ios.Out, "Homepage ID: %s (%s)\n", space.Homepage.ID, space.Homepage.Title)
				}
				if space.Description != nil && space.Description.Plain != nil && space.Description.Plain.Value != "" {
					fmt.Fprintf(ios.Out, "Description: %s\n", space.Description.Plain.Value)
				}
				if webui, ok := space.Links["webui"]; ok {
					fmt.Fprintf(ios.Out, "URL:         %s%s\n", cl.BaseURL(), webui)
				}
				return nil
			})
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open space in the default web browser")
	return cmd
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var key, name, desc string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Confluence space",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			space, err := cl.CreateSpace(cmd.Context(), key, name, desc)
			if err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, space, func() error {
				fmt.Fprintf(ios.Out, "Created space %s (%s)\n", space.Name, space.Key)
				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Space key (required)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Space name (required)")
	cmd.Flags().StringVarP(&desc, "description", "d", "", "Space description")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if !confirm {
				return fmt.Errorf("deleting a space is irreversible: pass --confirm to proceed")
			}

			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return err
			}

			if err := cl.DeleteSpace(cmd.Context(), key); err != nil {
				return err
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			fmt.Fprintf(ios.Out, "Deleted space %s\n", key)
			return nil
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm space deletion")
	return cmd
}
