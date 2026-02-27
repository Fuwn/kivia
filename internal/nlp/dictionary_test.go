package nlp_test

import (
	"github.com/Fuwn/kivia/internal/nlp"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDictionaryRecognizesLexiconWords(testingContext *testing.T) {
	dictionaryFile := filepath.Join("..", "..", "testdata", "dictionary", "words.txt")

	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryFile)

	dictionary, err := nlp.NewDictionary()

	if err != nil {
		testingContext.Fatalf("NewDictionary returned an error: %v", err)
	}

	if !dictionary.IsWord("options") {
		testingContext.Fatalf("Expected options to be recognized.")
	}

	if !dictionary.IsWord("has") {
		testingContext.Fatalf("Expected has to be recognized.")
	}

	if !dictionary.IsWord("resources") {
		testingContext.Fatalf("Expected resources to be recognized through plural inflection.")
	}
}

func TestDictionaryFindsAbbreviationExpansions(testingContext *testing.T) {
	dictionaryFile := filepath.Join("..", "..", "testdata", "dictionary", "words.txt")

	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryFile)

	dictionary, err := nlp.NewDictionary()

	if err != nil {
		testingContext.Fatalf("NewDictionary returned an error: %v", err)
	}

	cases := map[string]string{
		"expr": "expression",
		"ctx":  "context",
		"err":  "error",
	}

	for token, expectedExpansion := range cases {
		expansion, ok := dictionary.AbbreviationExpansion(token)

		if !ok {
			testingContext.Fatalf("Expected an abbreviation expansion for %q.", token)
		}

		if expansion != expectedExpansion {
			testingContext.Fatalf("Expected %q to expand to %q, got %q.", token, expectedExpansion, expansion)
		}
	}
}

func TestDictionaryLoadsFromMultipleDictionaryFiles(testingContext *testing.T) {
	tempDirectory := testingContext.TempDir()
	firstDictionaryPath := filepath.Join(tempDirectory, "first.txt")
	secondDictionaryPath := filepath.Join(tempDirectory, "second.txt")
	combinedPathList := strings.Join([]string{firstDictionaryPath, secondDictionaryPath}, string(os.PathListSeparator))

	if err := os.WriteFile(firstDictionaryPath, []byte("alpha\n"), 0o644); err != nil {
		testingContext.Fatalf("os.WriteFile returned an error: %v", err)
	}

	if err := os.WriteFile(secondDictionaryPath, []byte("beta\n"), 0o644); err != nil {
		testingContext.Fatalf("os.WriteFile returned an error: %v", err)
	}

	testingContext.Setenv("KIVIA_DICTIONARY_PATH", combinedPathList)

	dictionary, err := nlp.NewDictionary()

	if err != nil {
		testingContext.Fatalf("NewDictionary returned an error: %v", err)
	}

	if !dictionary.IsWord("alpha") {
		testingContext.Fatalf("Expected alpha to be recognized.")
	}

	if !dictionary.IsWord("beta") {
		testingContext.Fatalf("Expected beta to be recognized.")
	}
}

func TestDictionaryFailsWhenConfiguredPathHasNoWords(testingContext *testing.T) {
	tempDirectory := testingContext.TempDir()
	emptyDictionaryPath := filepath.Join(tempDirectory, "empty.txt")

	if err := os.WriteFile(emptyDictionaryPath, []byte("\n"), 0o644); err != nil {
		testingContext.Fatalf("os.WriteFile returned an error: %v", err)
	}

	testingContext.Setenv("KIVIA_DICTIONARY_PATH", emptyDictionaryPath)

	_, err := nlp.NewDictionary()

	if err == nil {
		testingContext.Fatalf("Expected NewDictionary to fail when configured dictionary has no usable words.")
	}
}

func TestDictionaryRecognizesDerivedForms(testingContext *testing.T) {
	tempDirectory := testingContext.TempDir()
	dictionaryPath := filepath.Join(tempDirectory, "base_words.txt")

	if err := os.WriteFile(dictionaryPath, []byte("trim\ntoken\n"), 0o644); err != nil {
		testingContext.Fatalf("os.WriteFile returned an error: %v", err)
	}

	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryPath)

	dictionary, err := nlp.NewDictionary()

	if err != nil {
		testingContext.Fatalf("NewDictionary returned an error: %v", err)
	}

	if !dictionary.IsWord("trimmed") {
		testingContext.Fatalf("Expected trimmed to be recognized from trim.")
	}

	if !dictionary.IsWord("tokenize") {
		testingContext.Fatalf("Expected tokenize to be recognized from token.")
	}
}

func TestDictionaryRecognizesBritishAndAmericanVariants(testingContext *testing.T) {
	tempDirectory := testingContext.TempDir()
	dictionaryPath := filepath.Join(tempDirectory, "british_words.txt")

	if err := os.WriteFile(dictionaryPath, []byte("normalise\ncolour\ncentre\n"), 0o644); err != nil {
		testingContext.Fatalf("os.WriteFile returned an error: %v", err)
	}

	testingContext.Setenv("KIVIA_DICTIONARY_PATH", dictionaryPath)

	dictionary, err := nlp.NewDictionary()

	if err != nil {
		testingContext.Fatalf("NewDictionary returned an error: %v", err)
	}

	if !dictionary.IsWord("normalize") {
		testingContext.Fatalf("Expected normalize to be recognized from normalise.")
	}

	if !dictionary.IsWord("color") {
		testingContext.Fatalf("Expected color to be recognized from colour.")
	}

	if !dictionary.IsWord("center") {
		testingContext.Fatalf("Expected center to be recognized from centre.")
	}
}
