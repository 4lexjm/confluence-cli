package auth

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/atlassian/confluence-cli/internal/config"
	"github.com/atlassian/confluence-cli/internal/secret"
	"github.com/atlassian/confluence-cli/pkg/auth"
	"github.com/atlassian/confluence-cli/pkg/cmdutil"
)

// NewCmdAuth returns the root auth command.
func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Confluence authentication credentials",
		Long: `Manage authentication credentials for Confluence Data Center and Cloud instances.

Credentials are stored securely in the OS keychain. Supported mechanisms:
  - API Token (Cloud):      confluence auth login https://myorg.atlassian.net --email me@example.com --token <TOKEN>
  - Bearer PAT (DC >= 7.9): confluence auth login https://confluence.example.com --token <PAT>
  - Session cookie (SSO):   confluence auth login https://confluence.example.com --browser
  - Kerberos (DC + AD):     confluence auth login https://confluence.example.com --auth-method kerberos
  - Basic Auth (DC):        confluence auth login https://confluence.example.com --username admin --token <password>`,
	}

	cmd.AddCommand(newLoginCmd(f))
	cmd.AddCommand(newStatusCmd(f))
	cmd.AddCommand(newLogoutCmd(f))
	cmd.AddCommand(newDoctorCmd(f))
	cmd.AddCommand(newSessionCmd(f))

	return cmd
}

type loginOptions struct {
	Host               string
	Kind               string
	AuthMethod         string
	Username           string
	Token              string
	Email              string
	Browser            bool
	AllowInsecureStore bool
	ContextName        string
	SetActive          bool
	SpaceKey           string
}

func newLoginCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &loginOptions{Kind: "dc"}

	cmd := &cobra.Command{
		Use:   "login <host>",
		Short: "Authenticate against a Confluence instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Host = args[0]
			return runLogin(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Kind, "kind", "", "Deployment kind: cloud or dc (auto-detected if omitted)")
	cmd.Flags().StringVar(&opts.AuthMethod, "auth-method", "", "Authentication method: basic, bearer, session-cookie, kerberos")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Username for Basic authentication")
	cmd.Flags().StringVarP(&opts.Token, "token", "t", "", "API token (Cloud) or Personal Access Token (DC)")
	cmd.Flags().StringVar(&opts.Email, "email", "", "Atlassian account email (Cloud)")
	cmd.Flags().BoolVar(&opts.Browser, "browser", false, "Authenticate via browser SSO session capture")
	cmd.Flags().BoolVar(&opts.AllowInsecureStore, "allow-insecure-store", false, "Allow storing credentials in file fallback if keychain fails")
	cmd.Flags().StringVarP(&opts.ContextName, "context", "c", "", "Name of the context to create or bind")
	cmd.Flags().BoolVar(&opts.SetActive, "set-active", true, "Make this context active")
	cmd.Flags().StringVarP(&opts.SpaceKey, "space", "s", "", "Default space key for this context")

	return cmd
}

func runLogin(cmd *cobra.Command, f *cmdutil.Factory, opts *loginOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	rawHost := strings.TrimRight(opts.Host, "/")
	if !strings.HasPrefix(rawHost, "http://") && !strings.HasPrefix(rawHost, "https://") {
		rawHost = "https://" + rawHost
	}

	u, err := url.Parse(rawHost)
	if err != nil {
		return fmt.Errorf("invalid host URL: %w", err)
	}

	isCloud := strings.Contains(u.Host, ".atlassian.net")
	if opts.Kind == "" {
		if isCloud {
			opts.Kind = "cloud"
		} else {
			opts.Kind = "dc"
		}
	}

	if isCloud && !strings.HasSuffix(rawHost, "/wiki") {
		rawHost = rawHost + "/wiki"
	}

	// Browser SSO capture
	if opts.Browser {
		opts.AuthMethod = config.AuthMethodSessionCookie
		result, err := auth.CaptureSessionViaBrowser(cmd.Context(), rawHost, f.BrowserOpener())
		if err != nil {
			return fmt.Errorf("browser SSO login: %w", err)
		}
		if err := secret.Set(secret.SessionKey(rawHost), result.Cookie); err != nil {
			return fmt.Errorf("store session cookie: %w", err)
		}
		fmt.Fprintf(ios.Out, "Session captured successfully. Expires in approximately %s.\n", auth.DefaultSessionTTL)
	} else if opts.AuthMethod == config.AuthMethodKerberos {
		// Kerberos doesn't require storing a token
	} else {
		// Determine auth method
		if opts.AuthMethod == "" {
			if opts.Kind == "cloud" {
				opts.AuthMethod = config.AuthMethodBasic
			} else if opts.Username != "" {
				opts.AuthMethod = config.AuthMethodBasic
			} else {
				opts.AuthMethod = config.AuthMethodBearer
			}
		}

		// Prompt for email if cloud
		if opts.Kind == "cloud" && opts.Email == "" && ios.CanPrompt() {
			fmt.Fprintf(ios.Out, "Atlassian Account Email: ")
			reader := bufio.NewReader(ios.In)
			line, _ := reader.ReadString('\n')
			opts.Email = strings.TrimSpace(line)
		}

		// Prompt for token if empty
		if opts.Token == "" && ios.CanPrompt() {
			prompt := "API Token: "
			if opts.Kind == "dc" {
				prompt = "Personal Access Token (PAT): "
			}
			fmt.Fprint(ios.Out, prompt)
			bytePass, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(ios.Out)
			if err != nil {
				return fmt.Errorf("read password: %w", err)
			}
			opts.Token = strings.TrimSpace(string(bytePass))
		}

		if opts.Token != "" {
			if err := secret.Set(secret.TokenKey(rawHost), opts.Token); err != nil {
				return fmt.Errorf("store token: %w", err)
			}
		}
	}

	// Save configuration
	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	hostObj := &config.Host{
		Kind:               opts.Kind,
		BaseURL:            rawHost,
		Username:           opts.Username,
		Email:              opts.Email,
		AuthMethod:         opts.AuthMethod,
		DefaultSpace:       opts.SpaceKey,
		AllowInsecureStore: opts.AllowInsecureStore,
	}
	cfg.Hosts[rawHost] = hostObj

	contextName := opts.ContextName
	if contextName == "" {
		contextName = u.Hostname()
	}

	cfg.Contexts[contextName] = &config.Context{
		Host:     rawHost,
		SpaceKey: opts.SpaceKey,
	}

	if opts.SetActive || cfg.ActiveContext == "" {
		cfg.ActiveContext = contextName
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(ios.Out, "Logged in to %s (context: %s)\n", rawHost, contextName)
	return nil
}

func newStatusCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display authentication and context status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.ResolveConfig()
			if err != nil {
				return err
			}

			statusData := map[string]any{
				"active_context": cfg.ActiveContext,
				"contexts_count": len(cfg.Contexts),
				"hosts_count":    len(cfg.Hosts),
			}

			if activeName, activeCtx, activeHost, err := cfg.ActiveContextInfo(); err == nil {
				statusData["context"] = map[string]any{
					"name":       activeName,
					"host":       activeCtx.Host,
					"space":      activeCtx.SpaceKey,
					"kind":       activeHost.Kind,
					"auth":       activeHost.AuthMethod,
					"user":       activeHost.Username,
					"email":      activeHost.Email,
					"has_token":  secret.EnvToken() != "" || (func() bool { _, e := secret.Get(secret.TokenKey(activeHost.BaseURL)); return e == nil })(),
					"has_cookie": secret.EnvSessionCookie() != "" || (func() bool { _, e := secret.Get(secret.SessionKey(activeHost.BaseURL)); return e == nil })(),
				}
			}

			ios, err := f.Streams()
			if err != nil {
				return err
			}

			return cmdutil.PrintData(cmd, ios.Out, statusData, func() error {
				fmt.Fprintf(ios.Out, "Active Context: %s\n", cfg.ActiveContext)
				if activeCtx, ok := cfg.Contexts[cfg.ActiveContext]; ok {
					fmt.Fprintf(ios.Out, "Host:           %s\n", activeCtx.Host)
					if activeCtx.SpaceKey != "" {
						fmt.Fprintf(ios.Out, "Default Space:  %s\n", activeCtx.SpaceKey)
					}
					if host, ok := cfg.Hosts[activeCtx.Host]; ok {
						fmt.Fprintf(ios.Out, "Platform:       %s\n", strings.ToUpper(host.Kind))
						fmt.Fprintf(ios.Out, "Auth Method:    %s\n", host.AuthMethod)
						if host.Email != "" {
							fmt.Fprintf(ios.Out, "Account Email:  %s\n", host.Email)
						}
						if host.Username != "" {
							fmt.Fprintf(ios.Out, "Username:       %s\n", host.Username)
						}
					}
				} else {
					fmt.Fprintf(ios.Out, "No active context configured.\n")
				}
				return nil
			})
		},
	}
}

func newLogoutCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "logout [host]",
		Short: "Remove stored credentials for a Confluence host",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.ResolveConfig()
			if err != nil {
				return err
			}

			targetHost := ""
			if len(args) > 0 {
				targetHost = args[0]
			} else if host, err := cfg.ActiveHost(); err == nil {
				targetHost = host.BaseURL
			}

			if targetHost == "" {
				return errors.New("no host specified and no active host found")
			}

			_ = secret.Delete(secret.TokenKey(targetHost))
			_ = secret.Delete(secret.SessionKey(targetHost))

			delete(cfg.Hosts, targetHost)
			_ = cfg.Save()

			ios, _ := f.Streams()
			fmt.Fprintf(ios.Out, "Logged out from %s.\n", targetHost)
			return nil
		},
	}
}

func newDoctorCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Verify Confluence connectivity and authentication",
		RunE: func(cmd *cobra.Command, args []string) error {
			ios, err := f.Streams()
			if err != nil {
				return err
			}

			fmt.Fprintf(ios.Out, "Checking Confluence connection...\n")
			cl, err := f.ConfluenceClient(cmd)
			if err != nil {
				return fmt.Errorf("initialization failed: %w", err)
			}

			fmt.Fprintf(ios.Out, "Connecting to %s ...\n", cl.BaseURL())
			spaces, err := cl.ListSpaces(cmd.Context(), 1, 0, "")
			if err != nil {
				return fmt.Errorf("connection failed: %w", err)
			}

			fmt.Fprintf(ios.Out, "SUCCESS: Authenticated and reachable. Found accessible spaces.\n")
			if len(spaces.Results) > 0 {
				fmt.Fprintf(ios.Out, "Sample space accessible: %s (%s)\n", spaces.Results[0].Name, spaces.Results[0].Key)
			}
			return nil
		},
	}
}

func newSessionCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "session",
		Short: "Inspect session cookie details",
		RunE: func(cmd *cobra.Command, args []string) error {
			ios, err := f.Streams()
			if err != nil {
				return err
			}

			cfg, err := f.ResolveConfig()
			if err != nil {
				return err
			}

			host, err := cfg.ActiveHost()
			if err != nil {
				return err
			}

			cookieVal := secret.EnvSessionCookie()
			source := "environment variable (CONFLUENCE_SESSION_COOKIE)"
			if cookieVal == "" {
				cookieVal, _ = secret.Get(secret.SessionKey(host.BaseURL))
				source = "OS keychain"
			}

			if cookieVal == "" {
				fmt.Fprintf(ios.Out, "No session cookie found for host %s.\n", host.BaseURL)
				return nil
			}

			fmt.Fprintf(ios.Out, "Host:   %s\n", host.BaseURL)
			fmt.Fprintf(ios.Out, "Source: %s\n", source)
			fmt.Fprintf(ios.Out, "Status: Active (length: %d chars)\n", len(cookieVal))
			return nil
		},
	}
}
