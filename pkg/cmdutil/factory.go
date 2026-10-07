package cmdutil

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/internal/config"
	"github.com/atlassian/confluence-cli/pkg/browser"
	"github.com/atlassian/confluence-cli/pkg/confluence"
	"github.com/atlassian/confluence-cli/pkg/httpx"
	"github.com/atlassian/confluence-cli/pkg/iostreams"
)

// Factory wires together shared services used by Cobra commands.
type Factory struct {
	AppVersion     string
	ExecutableName string

	IOStreams *iostreams.IOStreams
	Config    func() (*config.Config, error)

	Browser browser.Browser

	once struct {
		cfg sync.Once
	}
	cfg    *config.Config
	cfgErr error
	ioOnce sync.Once
	ios    *iostreams.IOStreams
}

// ResolveConfig loads configuration, caching the result.
func (f *Factory) ResolveConfig() (*config.Config, error) {
	f.once.cfg.Do(func() {
		if f.Config == nil {
			f.cfg, f.cfgErr = config.Load()
			return
		}
		f.cfg, f.cfgErr = f.Config()
	})
	return f.cfg, f.cfgErr
}

// Streams returns process IO streams, initialising them lazily.
func (f *Factory) Streams() (*iostreams.IOStreams, error) {
	f.ioOnce.Do(func() {
		if f.IOStreams != nil {
			f.ios = f.IOStreams
			return
		}
		f.ios = iostreams.System()
	})
	return f.ios, nil
}

// BrowserOpener returns a Browser implementation.
func (f *Factory) BrowserOpener() browser.Browser {
	if f.Browser == nil {
		f.Browser = browser.NewSystem()
	}
	return f.Browser
}

// ResolveContext returns the effective context name, context, and host.
func (f *Factory) ResolveContext(cmd *cobra.Command) (string, *config.Context, *config.Host, error) {
	cfg, err := f.ResolveConfig()
	if err != nil {
		return "", nil, nil, err
	}

	contextOverride, _ := cmd.Flags().GetString("context")
	if contextOverride == "" {
		contextOverride = os.Getenv("CONFLUENCE_CONTEXT")
	}

	targetContext := cfg.ActiveContext
	if contextOverride != "" {
		targetContext = contextOverride
	}

	if targetContext == "" {
		return "", nil, nil, errors.New("no active Confluence context — run `confluence context create` or `confluence auth login` first")
	}

	ctx, ok := cfg.Contexts[targetContext]
	if !ok {
		return "", nil, nil, fmt.Errorf("context %q not found", targetContext)
	}

	host, ok := cfg.Hosts[ctx.Host]
	if !ok {
		return "", nil, nil, fmt.Errorf("host %q not found for context %q", ctx.Host, targetContext)
	}

	return targetContext, ctx, host, nil
}

// ResolveSpace returns the effective space key from --space flag, env, context, or error if required and empty.
func (f *Factory) ResolveSpace(cmd *cobra.Command, required bool) (string, error) {
	spaceFlag, _ := cmd.Flags().GetString("space")
	if spaceFlag != "" {
		return strings.ToUpper(strings.TrimSpace(spaceFlag)), nil
	}

	cfg, err := f.ResolveConfig()
	if err == nil {
		if resolved := cfg.ResolveSpaceKey(""); resolved != "" {
			return resolved, nil
		}
	}

	if required {
		return "", errors.New("space key required: specify via --space flag, CONFLUENCE_SPACE env, or context default")
	}
	return "", nil
}

// ConfluenceClient creates a Confluence API client for the active command context.
func (f *Factory) ConfluenceClient(cmd *cobra.Command) (*confluence.Client, error) {
	_, _, host, err := f.ResolveContext(cmd)
	if err != nil {
		return nil, err
	}

	httpCl, err := httpx.New(host)
	if err != nil {
		return nil, fmt.Errorf("init http client: %w", err)
	}

	return confluence.NewClient(httpCl), nil
}
