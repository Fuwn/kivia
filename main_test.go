package main

import (
	"github.com/Fuwn/kivia/internal/analyze"
	"github.com/Fuwn/kivia/internal/collect"
	"testing"
)

func TestParseOptionsDefaults(testingContext *testing.T) {
	options, err := parseOptions([]string{"--path", "./testdata"})

	if err != nil {
		testingContext.Fatalf("parseOptions returned an error: %v", err)
	}

	if options.MinimumEvaluationLength != 1 {
		testingContext.Fatalf("Expected min-eval-length to default to 1, got %d.", options.MinimumEvaluationLength)
	}
}

func TestParseOptionsRejectsStrictFlag(testingContext *testing.T) {
	_, err := parseOptions([]string{"--path", "./testdata", "--strict"})

	if err == nil {
		testingContext.Fatalf("Expected parseOptions to fail when --strict is provided.")
	}
}

func TestParseOptionsReadsMultipleIgnoreFlags(testingContext *testing.T) {
	options, err := parseOptions([]string{
		"--path", "./testdata",
		"--ignore", "name=ctx",
		"--ignore", "file=_test.go",
		"--ignore", "reason=too short",
	})

	if err != nil {
		testingContext.Fatalf("parseOptions returned an error: %v", err)
	}

	if len(options.Ignore) != 3 {
		testingContext.Fatalf("Expected three ignore values, got %d.", len(options.Ignore))
	}
}

func TestParseOptionsRejectsInvalidMinimumEvaluationLength(testingContext *testing.T) {
	_, err := parseOptions([]string{"--path", "./testdata", "--min-eval-length", "0"})

	if err == nil {
		testingContext.Fatalf("Expected parseOptions to fail for min-eval-length=0.")
	}
}

func TestApplyIgnoreFilters(testingContext *testing.T) {
	input := analyzeResultFixture()
	filtered := applyIgnoreFilters(input, []string{
		"name=ctx",
		"reason=too short",
		"file=_test.go",
	})

	if len(filtered.Violations) != 1 {
		testingContext.Fatalf("Expected one remaining violation, got %d.", len(filtered.Violations))
	}

	if filtered.Violations[0].Identifier.Name != "userNum" {
		testingContext.Fatalf("Unexpected remaining violation: %q.", filtered.Violations[0].Identifier.Name)
	}
}

func analyzeResultFixture() analyze.Result {
	return analyze.Result{
		Violations: []analyze.Violation{
			{
				Identifier: collect.Identifier{
					Name: "ctx",
					Kind: "parameter",
					File: "sample.go",
					Context: collect.Context{
						EnclosingFunction: "Handle",
					},
				},
				Reason: "Contains abbreviation: ctx.",
			},
			{
				Identifier: collect.Identifier{
					Name: "t",
					Kind: "parameter",
					File: "main_test.go",
				},
				Reason: "Name is too short to be self-documenting.",
			},
			{
				Identifier: collect.Identifier{
					Name: "userNum",
					Kind: "parameter",
					File: "sample.go",
				},
				Reason: "Contains abbreviation: num.",
			},
		},
	}
}
