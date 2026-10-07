package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// PrintData handles structured (--json, --yaml, --jq) or human-formatted printing.
func PrintData(cmd *cobra.Command, out io.Writer, data any, humanFn func() error) error {
	isJSON, _ := cmd.Flags().GetBool("json")
	isYAML, _ := cmd.Flags().GetBool("yaml")
	jqExpr, _ := cmd.Flags().GetString("jq")

	if jqExpr != "" {
		isJSON = true
	}

	if isYAML {
		encoder := yaml.NewEncoder(out)
		encoder.SetIndent(2)
		return encoder.Encode(data)
	}

	if isJSON {
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}

		if jqExpr != "" {
			return applyJQ(jsonData, jqExpr, out)
		}

		_, err = fmt.Fprintln(out, string(jsonData))
		return err
	}

	if humanFn != nil {
		return humanFn()
	}

	// Fallback to json if no human function provided
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(jsonData))
	return err
}

func applyJQ(jsonData []byte, expr string, out io.Writer) error {
	var input any
	if err := json.Unmarshal(jsonData, &input); err != nil {
		return fmt.Errorf("parse json for jq: %w", err)
	}

	query, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("parse jq expression %q: %w", expr, err)
	}

	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return fmt.Errorf("jq evaluation: %w", err)
		}

		switch val := v.(type) {
		case string:
			fmt.Fprintln(out, val)
		default:
			b, err := json.MarshalIndent(val, "", "  ")
			if err != nil {
				fmt.Fprintln(out, val)
			} else {
				fmt.Fprintln(out, strings.TrimSpace(string(b)))
			}
		}
	}
	return nil
}
