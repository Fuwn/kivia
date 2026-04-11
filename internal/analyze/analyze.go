package analyze

import (
	"github.com/Fuwn/kivia/internal/collect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Options struct {
	MinEvaluationLength int
}

type Result struct {
	Violations []Violation `json:"violations"`
}

type Violation struct {
	Identifier collect.Identifier `json:"identifier"`
	Reason     string             `json:"reason"`
}

func Run(identifiers []collect.Identifier, options Options) (Result, error) {
	minimumEvaluationLength := options.MinEvaluationLength

	if minimumEvaluationLength <= 0 {
		minimumEvaluationLength = 1
	}

	resources, err := getResources()

	if err != nil {
		return Result{}, err
	}

	violations := make([]Violation, 0)

	for _, identifier := range identifiers {
		if utf8.RuneCountInString(strings.TrimSpace(identifier.Name)) < minimumEvaluationLength {
			continue
		}

		evaluation := evaluateIdentifier(identifier, resources, minimumEvaluationLength)

		if !evaluation.isViolation {
			continue
		}

		violation := Violation{
			Identifier: identifier,
			Reason:     evaluation.reason,
		}
		violations = append(violations, violation)
	}

	return Result{Violations: violations}, nil
}

type evaluationResult struct {
	isViolation bool
	reason      string
}

func evaluateIdentifier(identifier collect.Identifier, resources resources, minimumTokenLength int) evaluationResult {
	name := strings.TrimSpace(identifier.Name)

	if name == "" {
		return evaluationResult{}
	}

	tokens := tokenize(name)

	if len(tokens) == 0 {
		return evaluationResult{}
	}

	for _, token := range tokens {
		if utf8.RuneCountInString(token) < minimumTokenLength {
			continue
		}

		if !isAlphabeticToken(token) {
			continue
		}

		if resources.dictionary.IsWord(token) {
			continue
		}

		if isUpperCaseToken(name, token) {
			continue
		}

		if isDisallowedAbbreviation(token, resources) {
			return evaluationResult{isViolation: true, reason: "Contains abbreviation: " + token + "."}
		}

		return evaluationResult{isViolation: true, reason: "Term not found in dictionary: " + token + "."}
	}

	return evaluationResult{}
}

func isUpperCaseToken(identifierName string, token string) bool {
	tokenLength := utf8.RuneCountInString(token)

	if tokenLength < 2 || tokenLength > 8 {
		return false
	}

	upperToken := strings.ToUpper(token)

	if !strings.Contains(identifierName, upperToken) {
		return false
	}

	tokenIndex := strings.Index(identifierName, upperToken)
	afterIndex := tokenIndex + len(upperToken)

	if afterIndex >= len(identifierName) {
		return true
	}

	nextRune, _ := utf8.DecodeRuneInString(identifierName[afterIndex:])

	return !unicode.IsLower(nextRune)
}

func tokenize(name string) []string {
	name = strings.TrimSpace(name)

	if name == "" {
		return nil
	}

	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	if len(parts) == 0 {
		return nil
	}

	result := make([]string, 0, len(parts)*2)

	for _, part := range parts {
		if part == "" {
			continue
		}

		result = append(result, splitCamel(part)...)
	}

	return result
}

func splitCamel(input string) []string {
	if input == "" {
		return nil
	}

	runes := []rune(input)

	if len(runes) == 0 {
		return nil
	}

	tokens := make([]string, 0, 2)
	start := 0

	for index := 1; index < len(runes); index++ {
		current := runes[index]
		previous := runes[index-1]
		next := rune(0)

		if index+1 < len(runes) {
			next = runes[index+1]
		}

		isBoundary := false

		if unicode.IsLower(previous) && unicode.IsUpper(current) {
			isBoundary = true
		}

		if unicode.IsDigit(previous) != unicode.IsDigit(current) {
			isBoundary = true
		}

		if unicode.IsUpper(previous) && unicode.IsUpper(current) && next != 0 && unicode.IsLower(next) {
			isBoundary = true
		}

		if isBoundary {
			tokens = append(tokens, strings.ToLower(string(runes[start:index])))
			start = index
		}
	}

	tokens = append(tokens, strings.ToLower(string(runes[start:])))

	return tokens
}

func isDisallowedAbbreviation(token string, resources resources) bool {
	_, hasExpansion := resources.dictionary.AbbreviationExpansion(token)

	return hasExpansion
}

func isAlphabeticToken(token string) bool {
	if token == "" {
		return false
	}

	for _, character := range token {
		if !unicode.IsLetter(character) {
			return false
		}
	}

	return true
}
