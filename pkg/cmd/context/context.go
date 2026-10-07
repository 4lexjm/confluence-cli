package context

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/internal/config"
	"github.com/atlassian/confluence-cli/pkg/cmdutil"
)

// NewCmdContext returns the context command group.
func NewCmdContext(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage Confluence CLI contexts",
		Long: `Contexts allow you to switch quickly between different Confluence instances,
accounts, and default spaces.`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newUseCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newDeleteCmd(f))

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.ResolveConfig()
			if err != nil {
				return err
			}

			type contextItem struct {
				Name     string `json:"name" yaml:"name"`
				Active   bool   `json:"active" yaml:"active"`
				Host     string `json:"host" yaml:"host"`
				SpaceKey string `json:"space_key,omitempty" yaml:"space_key,omitempty"`
			}

			var items []contextItem
			for name, ctx := range cfg.Contexts {
				items = append(items, contextItem{
					Name:     name,
					Active:   name == cfg.ActiveContext,
					Host:     ctx.Host,
					SpaceKey: ctx.SpaceKey,
				})
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, items, func() error {
				if len(items) == 0 {
					fmt.Fprintln(ios.Out, "No contexts configured. Run `confluence context create` or `confluence auth login`.")
					return nil
				}
				fmt.Fprintf(ios.Out, "%-3s %-20s %-35s %-10s\n", "ACT", "NAME", "HOST", "SPACE")
				fmt.Fprintf(ios.Out, "%-3s %-20s %-35s %-10s\n", "---", "----", "----", "-----")
				for _, it := range items {
					marker := " "
					if it.Active {
						marker = "*"
					}
					fmt.Fprintf(ios.Out, "%-3s %-20s %-35s %-10s\n", marker, it.Name, it.Host, it.SpaceKey)
				}
				return nil
			})
		},
	}
}

type createOptions struct {
	Host      string
	SpaceKey  string
	SetActive bool
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create or update a context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := f.ResolveConfig()
			if err != nil {
				return err
			}

			if opts.Host == "" {
				return errors.New("--host is required")
			}

			opts.Host = strings.TrimRight(opts.Host, "/")
			cfg.Contexts[name] = &config.Context{
				Host:     opts.Host,
				SpaceKey: strings.ToUpper(opts.SpaceKey),
			}

			if opts.SetActive || cfg.ActiveContext == "" {
				cfg.ActiveContext = name
			}

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Context %q created.\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Host, "host", "", "Host URL for this context (required)")
	cmd.Flags().StringVarP(&opts.SpaceKey, "space", "s", "", "Default space key")
	cmd.Flags().BoolVar(&opts.SetActive, "set-active", true, "Set as active context")
	_ = cmd.MarkFlagRequired("host")

	return cmd
}

func newUseCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the active context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := f.ResolveConfig()
			if err != nil {
				return err
			}

			if err := cfg.SetActiveContext(name); err != nil {
				return err
			}

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Now using context %q.\n", name)
			return nil
		},
	}
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "view [name]",
		Short: "View context configuration details",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.ResolveConfig()
			if err != nil {
				return err
			}

			targetName := cfg.ActiveContext
			if len(args) > 0 {
				targetName = args[0]
			}

			if targetName == "" {
				return errors.New("no context specified and no active context")
			}

			ctx, ok := cfg.Contexts[targetName]
			if !ok {
				return fmt.Errorf("context %q not found", targetName)
			}

			data := map[string]any{
				"name":      targetName,
				"active":    targetName == cfg.ActiveContext,
				"host":      ctx.Host,
				"space_key": ctx.SpaceKey,
			}

			ios, _ := f.Streams()
			return cmdutil.PrintData(cmd, ios.Out, data, func() error {
				fmt.Fprintf(ios.Out, "Context:   %s\n", targetName)
				fmt.Fprintf(ios.Out, "Host:      %s\n", ctx.Host)
				if ctx.SpaceKey != "" {
					fmt.Fprintf(ios.Out, "Space:     %s\n", ctx.SpaceKey)
				}
				fmt.Fprintf(ios.Out, "Active:    %v\n", targetName == cfg.ActiveContext)
				return nil
			})
		},
	}
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a context",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := f.ResolveConfig()
			if err != nil {
				return err
			}

			if _, ok := cfg.Contexts[name]; !ok {
				return fmt.Errorf("context %q not found", name)
			}

			delete(cfg.Contexts, name)
			if cfg.ActiveContext == name {
				cfg.ActiveContext = ""
			}

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Context %q deleted.\n", name)
			return nil
		},
	}
}
