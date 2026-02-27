package analyze_test

import (
	"github.com/Fuwn/kivia/internal/analyze"
	"github.com/Fuwn/kivia/internal/collect"
	"os"
	"path/filepath"
	"testing"
)

func dictionaryPathForTests(testingContext *testing.T) string {
	testingContext.Helper()

	return filepath.Join("..", "..", "testdata", "dictionary", "words.txt")
}

func TestAnalyzeFlagsAbbreviations(testingContext *testing.T) {
	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryPathForTests(testingContext))

	root := filepath.Join("..", "..", "testdata", "samplepkg")
	identifiers, err := collect.FromPath(root)

	if err != nil {
		testingContext.Fatalf("collect.FromPath returned an error: %v", err)
	}

	result, err := analyze.Run(identifiers, analyze.Options{})

	if err != nil {
		testingContext.Fatalf("analyze.Run returned an error: %v", err)
	}

	if len(result.Violations) == 0 {
		testingContext.Fatalf("Expected at least one violation, got none.")
	}

	mustContainViolation(testingContext, result, "ctx")
	mustContainViolation(testingContext, result, "userNum")
	mustContainViolation(testingContext, result, "usr")
}

func TestAnalyzeFlagsTechnicalTermsNotInDictionary(testingContext *testing.T) {
	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryPathForTests(testingContext))

	identifiers := []collect.Identifier{
		{Name: "userID", Kind: "variable"},
		{Name: "httpClient", Kind: "variable"},
	}
	result, err := analyze.Run(identifiers, analyze.Options{})

	if err != nil {
		testingContext.Fatalf("analyze.Run returned an error: %v", err)
	}

	if len(result.Violations) == 0 {
		testingContext.Fatalf("Expected violations, got none.")
	}

	mustContainViolation(testingContext, result, "userID")
	mustContainViolation(testingContext, result, "httpClient")
}

func TestAnalyzeDoesNotFlagNormalDictionaryWords(testingContext *testing.T) {
	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryPathForTests(testingContext))

	identifiers := []collect.Identifier{
		{Name: "options", Kind: "variable"},
		{Name: "parsedResource", Kind: "variable"},
		{Name: "hasResources", Kind: "variable"},
		{Name: "allowlist", Kind: "variable"},
	}
	result, err := analyze.Run(identifiers, analyze.Options{})

	if err != nil {
		testingContext.Fatalf("analyze.Run returned an error: %v", err)
	}

	if len(result.Violations) != 0 {
		testingContext.Fatalf("Expected no violations, got %d.", len(result.Violations))
	}
}

func TestAnalyzeMinEvaluationLengthSkipsSingleLetterIdentifiers(testingContext *testing.T) {
	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryPathForTests(testingContext))

	identifiers := []collect.Identifier{
		{Name: "t", Kind: "parameter"},
		{Name: "v", Kind: "receiver"},
		{Name: "ctx", Kind: "parameter"},
	}
	result, err := analyze.Run(identifiers, analyze.Options{
		MinEvaluationLength: 2,
	})

	if err != nil {
		testingContext.Fatalf("analyze.Run returned an error: %v", err)
	}

	if len(result.Violations) != 1 {
		testingContext.Fatalf("Expected one violation, got %d.", len(result.Violations))
	}

	if result.Violations[0].Identifier.Name != "ctx" {
		testingContext.Fatalf("Expected only ctx to be evaluated, got %q.", result.Violations[0].Identifier.Name)
	}
}

func TestAnalyzeFlagsExpressionAbbreviation(testingContext *testing.T) {
	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryPathForTests(testingContext))

	identifiers := []collect.Identifier{
		{Name: "expr", Kind: "variable"},
	}
	result, err := analyze.Run(identifiers, analyze.Options{
		MinEvaluationLength: 1,
	})

	if err != nil {
		testingContext.Fatalf("analyze.Run returned an error: %v", err)
	}

	if len(result.Violations) != 1 {
		testingContext.Fatalf("Expected one violation, got %d.", len(result.Violations))
	}

	if result.Violations[0].Identifier.Name != "expr" {
		testingContext.Fatalf("Expected expr to be flagged, got %q.", result.Violations[0].Identifier.Name)
	}
}

func TestAnalyzeAllowsUpperCaseTokens(testingContext *testing.T) {
	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryPathForTests(testingContext))

	identifiers := []collect.Identifier{
		{Name: "JSON", Kind: "variable"},
	}
	result, err := analyze.Run(identifiers, analyze.Options{})

	if err != nil {
		testingContext.Fatalf("analyze.Run returned an error: %v", err)
	}

	if len(result.Violations) != 0 {
		testingContext.Fatalf("Expected no violations, got %d.", len(result.Violations))
	}
}

func TestAnalyzeFailsWhenDictionaryIsUnavailable(testingContext *testing.T) {
	emptyDictionaryPath := filepath.Join(testingContext.TempDir(), "empty.txt")

	if err := os.WriteFile(emptyDictionaryPath, []byte("\n"), 0o644); err != nil {
		testingContext.Fatalf("os.WriteFile returned an error: %v", err)
	}

	testingContext.Setenv("KIVIA_DICTIONARY_PATH", emptyDictionaryPath)

	_, err := analyze.Run([]collect.Identifier{{Name: "ctx", Kind: "parameter"}}, analyze.Options{})

	if err == nil {
		testingContext.Fatalf("Expected analyze.Run to fail when dictionary data is unavailable.")
	}
}

func mustContainViolation(testingContext *testing.T, result analyze.Result, name string) {
	testingContext.Helper()

	for _, violation := range result.Violations {
		if violation.Identifier.Name == name {
			return
		}
	}

	testingContext.Fatalf("Expected a violation for %q.", name)
}
