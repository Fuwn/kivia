package report

import (
	"encoding/json"
	"fmt"
	"github.com/Fuwn/kivia/internal/analyze"
	"github.com/Fuwn/kivia/internal/collect"
	"io"
	"strings"
)

func Render(writer io.Writer, result analyze.Result, format string, includeContext bool) error {
	switch strings.ToLower(format) {
	case "json":
		return renderJSON(writer, result, includeContext)
	case "text", "":
		return renderText(writer, result, includeContext)
	default:
		return fmt.Errorf("Unsupported output format %q. Use \"text\" or \"json\".", format)
	}
}

func renderText(writer io.Writer, result analyze.Result, includeContext bool) error {
	if len(result.Violations) == 0 {
		_, err := fmt.Fprintln(writer, "No naming violations found.")

		return err
	}

	for _, violation := range result.Violations {
		if _, err := fmt.Fprintf(writer, "%s:%d:%d %s %q: %s\n",
			violation.Identifier.File,
			violation.Identifier.Line,
			violation.Identifier.Column,
			violation.Identifier.Kind,
			violation.Identifier.Name,
			violation.Reason,
		); err != nil {
			return err
		}

		if includeContext {
			contextParts := make([]string, 0, 3)

			if violation.Identifier.Context.Type != "" {
				contextParts = append(contextParts, "type="+violation.Identifier.Context.Type)
			}

			if violation.Identifier.Context.ValueExpression != "" {
				contextParts = append(contextParts, "value="+violation.Identifier.Context.ValueExpression)
			}

			if violation.Identifier.Context.EnclosingFunction != "" {
				contextParts = append(contextParts, "function="+violation.Identifier.Context.EnclosingFunction)
			}

			if len(contextParts) > 0 {
				if _, err := fmt.Fprintf(writer, "  context: %s\n", strings.Join(contextParts, ", ")); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func renderJSON(writer io.Writer, result analyze.Result, includeContext bool) error {
	if !includeContext {
		for index := range result.Violations {
			result.Violations[index].Identifier.Context = collect.Context{}
		}
	}

	encoder := json.NewEncoder(writer)

	encoder.SetIndent("", "  ")

	return encoder.Encode(result)
}
