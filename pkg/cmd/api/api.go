package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/atlassian/confluence-cli/pkg/cmdutil"
)

type apiOptions struct {
	Method  string
	Input   string
	Fields  []string
	Headers []string
	Params  []string
}

// NewCmdAPI exposes a raw REST escape hatch akin to gh api / jira api.
func NewCmdAPI(f *cmdutil.Factory) *cobra.Command {
	opts := &apiOptions{}
	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "Make raw Confluence REST API requests",
		Long: `Call Confluence REST APIs directly for endpoints that do not yet have dedicated commands.

Examples:
  confluence api /rest/api/space
  confluence api /rest/api/content/12345 --json | jq '.title'
  confluence api /rest/api/content --method POST -F title="New Page" -F type=page
  confluence api /wiki/api/v2/pages --param limit=10`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAPI(cmd, f, opts, args[0])
		},
	}

	cmd.Flags().StringVarP(&opts.Method, "method", "X", "", "HTTP method (default GET, or POST when a body is supplied)")
	cmd.Flags().StringVarP(&opts.Input, "input", "d", "", "JSON string to use as the request body")
	cmd.Flags().StringArrayVarP(&opts.Fields, "field", "F", nil, "Add JSON body field (key=value, repeatable)")
	cmd.Flags().StringArrayVarP(&opts.Headers, "header", "H", nil, "Add an HTTP request header (Key: Value)")
	cmd.Flags().StringArrayVarP(&opts.Params, "param", "P", nil, "Append query parameter (key=value)")

	return cmd
}

func runAPI(cmd *cobra.Command, f *cmdutil.Factory, opts *apiOptions, path string) error {
	method := strings.ToUpper(strings.TrimSpace(opts.Method))

	var body any
	if len(opts.Fields) > 0 && opts.Input != "" {
		return fmt.Errorf("--field and --input flags cannot be combined")
	}

	if len(opts.Fields) > 0 {
		payload := make(map[string]any, len(opts.Fields))
		for _, field := range opts.Fields {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format %q: expected key=value", field)
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			payload[key] = inferJSONValue(val)
		}
		body = payload
	} else if strings.TrimSpace(opts.Input) != "" {
		var raw any
		if err := json.Unmarshal([]byte(opts.Input), &raw); err != nil {
			body = opts.Input
		} else {
			body = raw
		}
	}

	if method == "" {
		if body != nil {
			method = "POST"
		} else {
			method = "GET"
		}
	}

	// Append query parameters to path
	if len(opts.Params) > 0 {
		q := url.Values{}
		for _, p := range opts.Params {
			parts := strings.SplitN(p, "=", 2)
			if len(parts) == 2 {
				q.Add(parts[0], parts[1])
			} else {
				q.Add(parts[0], "")
			}
		}
		if strings.Contains(path, "?") {
			path = path + "&" + q.Encode()
		} else {
			path = path + "?" + q.Encode()
		}
	}

	headersMap := make(map[string]string)
	for _, h := range opts.Headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			headersMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	cl, err := f.ConfluenceClient(cmd)
	if err != nil {
		return err
	}

	resp, rawBytes, err := cl.Raw(cmd.Context(), method, path, body, headersMap)
	if err != nil {
		return err
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}

	// Parse JSON if possible for jq / formatting
	var parsedJSON any
	if jsonErr := json.Unmarshal(rawBytes, &parsedJSON); jsonErr == nil {
		return cmdutil.PrintData(cmd, ios.Out, parsedJSON, func() error {
			fmt.Fprintln(ios.Out, string(rawBytes))
			return nil
		})
	}

	// Plain text / non-JSON response
	fmt.Fprintf(ios.Out, "HTTP %d\n%s\n", resp.StatusCode, string(rawBytes))
	return nil
}

func inferJSONValue(v string) any {
	if v == "true" {
		return true
	}
	if v == "false" {
		return false
	}
	if v == "null" {
		return nil
	}
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return v
}
