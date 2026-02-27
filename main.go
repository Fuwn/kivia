package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Fuwn/kivia/internal/analyze"
	"github.com/Fuwn/kivia/internal/collect"
	"github.com/Fuwn/kivia/internal/report"
	"os"
	"slices"
	"strings"
)

type options struct {
	Path                    string
	OmitContext             bool
	MinimumEvaluationLength int
	Format                  string
	FailOnViolation         bool
	Ignore                  []string
}

func parseOptions(arguments []string) (options, error) {
	flagSet := flag.NewFlagSet("kivia", flag.ContinueOnError)

	flagSet.SetOutput(os.Stderr)

	var parsed options
	var ignoreValues stringSliceFlag

	flagSet.StringVar(&parsed.Path, "path", "./...", "Path to analyze (directory, file, or ./...).")
	flagSet.BoolVar(&parsed.OmitContext, "omit-context", false, "Hide usage context in output.")
	flagSet.IntVar(&parsed.MinimumEvaluationLength, "min-eval-length", 1, "Minimum identifier length in runes to evaluate.")
	flagSet.StringVar(&parsed.Format, "format", "text", "Output format: text or JSON.")
	flagSet.BoolVar(&parsed.FailOnViolation, "fail-on-violation", false, "Exit with code 1 when violations are found.")
	flagSet.Var(&ignoreValues, "ignore", "Ignore violations by matcher. Repeat this flag as needed. Prefixes: name=, kind=, file=, reason=, func=.")

	if err := flagSet.Parse(arguments); err != nil {
		return options{}, err
	}

	if parsed.MinimumEvaluationLength < 1 {
		return options{}, errors.New("The --min-eval-length value must be at least 1.")
	}

	parsed.Ignore = slices.Clone(ignoreValues)

	return parsed, nil
}

func run(parsed options) error {
	identifiers, err := collect.FromPath(parsed.Path)

	if err != nil {
		return err
	}

	result, err := analyze.Run(identifiers, analyze.Options{
		MinEvaluationLength: parsed.MinimumEvaluationLength,
	})

	if err != nil {
		return err
	}

	result = applyIgnoreFilters(result, parsed.Ignore)

	if err := report.Render(os.Stdout, result, parsed.Format, !parsed.OmitContext); err != nil {
		return err
	}

	if parsed.FailOnViolation && len(result.Violations) > 0 {
		return exitCodeError(1)
	}

	return nil
}

func main() {
	parsed, err := parseOptions(os.Args[1:])

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	if err := run(parsed); err != nil {
		var codeError exitCodeError

		if errors.As(err, &codeError) {
			os.Exit(int(codeError))
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type exitCodeError int

func (errorCode exitCodeError) Error() string {
	return fmt.Sprintf("Process exited with code %d.", int(errorCode))
}

type stringSliceFlag []string

func (values *stringSliceFlag) String() string {
	if values == nil {
		return ""
	}

	return strings.Join(*values, ",")
}

func (values *stringSliceFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if trimmed == "" {
		return errors.New("Ignore matcher cannot be empty.")
	}

	*values = append(*values, trimmed)

	return nil
}

func applyIgnoreFilters(result analyze.Result, ignoreMatchers []string) analyze.Result {
	if len(ignoreMatchers) == 0 || len(result.Violations) == 0 {
		return result
	}

	filteredViolations := make([]analyze.Violation, 0, len(result.Violations))

	for _, violation := range result.Violations {
		if shouldIgnoreViolation(violation, ignoreMatchers) {
			continue
		}

		filteredViolations = append(filteredViolations, violation)
	}

	result.Violations = filteredViolations

	return result
}

func shouldIgnoreViolation(violation analyze.Violation, ignoreMatchers []string) bool {
	for _, matcher := range ignoreMatchers {
		if matchesViolation(matcher, violation) {
			return true
		}
	}

	return false
}

func matchesViolation(matcher string, violation analyze.Violation) bool {
	normalizedMatcher := strings.ToLower(strings.TrimSpace(matcher))

	if normalizedMatcher == "" {
		return false
	}

	identifier := violation.Identifier

	if strings.HasPrefix(normalizedMatcher, "name=") {
		return strings.Contains(strings.ToLower(identifier.Name), strings.TrimPrefix(normalizedMatcher, "name="))
	}

	if strings.HasPrefix(normalizedMatcher, "kind=") {
		return strings.Contains(strings.ToLower(identifier.Kind), strings.TrimPrefix(normalizedMatcher, "kind="))
	}

	if strings.HasPrefix(normalizedMatcher, "file=") {
		return strings.Contains(strings.ToLower(identifier.File), strings.TrimPrefix(normalizedMatcher, "file="))
	}

	if strings.HasPrefix(normalizedMatcher, "reason=") {
		return strings.Contains(strings.ToLower(violation.Reason), strings.TrimPrefix(normalizedMatcher, "reason="))
	}

	if strings.HasPrefix(normalizedMatcher, "func=") {
		return strings.Contains(strings.ToLower(identifier.Context.EnclosingFunction), strings.TrimPrefix(normalizedMatcher, "func="))
	}

	composite := strings.ToLower(strings.Join([]string{
		identifier.Name,
		identifier.Kind,
		identifier.File,
		violation.Reason,
		identifier.Context.EnclosingFunction,
	}, " "))

	return strings.Contains(composite, normalizedMatcher)
}
