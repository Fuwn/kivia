package nlp

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/sajari/fuzzy"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var wordPattern = regexp.MustCompile(`[A-Za-z]+`)

type Dictionary struct {
	model                 *fuzzy.Model
	words                 map[string]struct{}
	wordsByFirstCharacter map[rune][]string
}

func NewDictionary() (*Dictionary, error) {
	words, err := loadWords()

	if err != nil {
		return nil, err
	}

	wordSet := makeWordSet(words)
	wordsByFirstCharacter := makeWordsByFirstCharacter(words)
	model, loadErr := loadCachedModel()

	if loadErr == nil {
		return &Dictionary{model: model, words: wordSet, wordsByFirstCharacter: wordsByFirstCharacter}, nil
	}

	model = fuzzy.NewModel()

	model.SetThreshold(1)
	model.SetDepth(1)
	model.SetUseAutocomplete(false)
	model.Train(words)

	_ = saveCachedModel(model)

	return &Dictionary{model: model, words: wordSet, wordsByFirstCharacter: wordsByFirstCharacter}, nil
}

func (dictionary *Dictionary) IsWord(token string) bool {
	token = normalizeToken(token)

	if token == "" || dictionary == nil {
		return false
	}

	return dictionary.isLexiconWord(token)
}

func (dictionary *Dictionary) Suggest(token string) string {
	token = normalizeToken(token)

	if token == "" || dictionary == nil || dictionary.model == nil {
		return ""
	}

	if dictionary.isLexiconWord(token) {
		return ""
	}

	suggestions := dictionary.model.SpellCheckSuggestions(token, 1)

	if len(suggestions) == 0 {
		return ""
	}

	if suggestions[0] == token {
		return ""
	}

	return suggestions[0]
}

func (dictionary *Dictionary) isLexiconWord(token string) bool {
	if dictionary == nil {
		return false
	}

	if _, ok := dictionary.words[token]; ok {
		return true
	}

	candidates := make([]string, 0, 16)
	candidates = append(candidates, inflectionCandidates(token)...)
	candidates = append(candidates, spellingVariantCandidates(token)...)

	for _, candidate := range inflectionCandidates(token) {
		candidates = append(candidates, spellingVariantCandidates(candidate)...)
	}

	uniqueCandidates := make(map[string]struct{}, len(candidates))

	for _, candidate := range candidates {
		if candidate == "" || candidate == token {
			continue
		}

		if _, seen := uniqueCandidates[candidate]; seen {
			continue
		}

		uniqueCandidates[candidate] = struct{}{}

		if _, ok := dictionary.words[candidate]; ok {
			return true
		}
	}

	return false
}

func (dictionary *Dictionary) AbbreviationExpansion(token string) (string, bool) {
	token = normalizeToken(token)

	if token == "" || dictionary == nil {
		return "", false
	}

	tokenLength := utf8.RuneCountInString(token)

	if tokenLength <= 1 || tokenLength > 4 {
		return "", false
	}

	firstCharacter, _ := utf8.DecodeRuneInString(token)
	candidates := dictionary.wordsByFirstCharacter[firstCharacter]

	if len(candidates) == 0 {
		return "", false
	}

	bestCandidate := ""
	bestScore := 1 << 30

	for _, candidate := range candidates {
		if !isLikelyAbbreviationForToken(token, candidate) {
			continue
		}

		score := abbreviationScore(token, candidate)

		if score < bestScore {
			bestScore = score
			bestCandidate = candidate
		}
	}

	if bestCandidate == "" {
		return "", false
	}

	return bestCandidate, true
}

func isLikelyAbbreviationForToken(token string, candidate string) bool {
	if candidate == "" || token == "" || token == candidate {
		return false
	}

	tokenLength := utf8.RuneCountInString(token)
	candidateLength := utf8.RuneCountInString(candidate)

	if candidateLength <= tokenLength {
		return false
	}

	if !isSubsequence(token, candidate) {
		return false
	}

	if strings.HasPrefix(candidate, token) && tokenLength <= 4 {
		return true
	}

	tokenConsonants := consonantSkeleton(token)
	candidateConsonants := consonantSkeleton(candidate)

	if tokenConsonants == "" || candidateConsonants == "" {
		return false
	}

	if isSubsequence(tokenConsonants, candidateConsonants) && tokenLength <= 5 {
		return true
	}

	return false
}

func abbreviationScore(token string, candidate string) int {
	tokenLength := utf8.RuneCountInString(token)
	candidateLength := utf8.RuneCountInString(candidate)
	lengthGap := max(candidateLength-tokenLength, 0)
	score := lengthGap * 10

	if strings.HasPrefix(candidate, token) {
		score -= 3
	}

	return score
}

func isSubsequence(shorter string, longer string) bool {
	shorterRunes := []rune(shorter)
	longerRunes := []rune(longer)
	shorterIndex := 0

	for _, character := range longerRunes {
		if shorterIndex >= len(shorterRunes) {
			break
		}

		if shorterRunes[shorterIndex] == character {
			shorterIndex++
		}
	}

	return shorterIndex == len(shorterRunes)
}

func consonantSkeleton(word string) string {
	var builder strings.Builder

	for _, character := range word {
		switch character {
		case 'a', 'e', 'i', 'o', 'u':
			continue
		default:
			builder.WriteRune(character)
		}
	}

	return builder.String()
}

func inflectionCandidates(token string) []string {
	candidates := make([]string, 0, 8)

	if strings.HasSuffix(token, "ies") && len(token) > 3 {
		candidates = append(candidates, token[:len(token)-3]+"y")
	}

	if strings.HasSuffix(token, "es") && len(token) > 2 {
		candidates = append(candidates, token[:len(token)-2])
	}

	if strings.HasSuffix(token, "s") && len(token) > 1 {
		candidates = append(candidates, token[:len(token)-1])
	}

	if strings.HasSuffix(token, "ed") && len(token) > 2 {
		candidateWithoutSuffix := token[:len(token)-2]
		candidates = append(candidates, candidateWithoutSuffix)
		candidates = append(candidates, candidateWithoutSuffix+"e")

		if len(candidateWithoutSuffix) >= 2 {
			lastCharacter := candidateWithoutSuffix[len(candidateWithoutSuffix)-1]
			secondToLastCharacter := candidateWithoutSuffix[len(candidateWithoutSuffix)-2]

			if lastCharacter == secondToLastCharacter {
				candidates = append(candidates, candidateWithoutSuffix[:len(candidateWithoutSuffix)-1])
			}
		}
	}

	if strings.HasSuffix(token, "ing") && len(token) > 3 {
		candidateWithoutSuffix := token[:len(token)-3]
		candidates = append(candidates, candidateWithoutSuffix)
		candidates = append(candidates, candidateWithoutSuffix+"e")
	}

	if strings.HasSuffix(token, "er") && len(token) > 2 {
		candidateWithoutSuffix := token[:len(token)-2]
		candidates = append(candidates, candidateWithoutSuffix)
		candidates = append(candidates, candidateWithoutSuffix+"e")

		if len(candidateWithoutSuffix) >= 2 {
			lastCharacter := candidateWithoutSuffix[len(candidateWithoutSuffix)-1]
			secondToLastCharacter := candidateWithoutSuffix[len(candidateWithoutSuffix)-2]

			if lastCharacter == secondToLastCharacter {
				candidates = append(candidates, candidateWithoutSuffix[:len(candidateWithoutSuffix)-1])
			}
		}
	}

	if strings.HasSuffix(token, "ize") && len(token) > 3 {
		candidates = append(candidates, token[:len(token)-3])
	}

	if strings.HasSuffix(token, "ized") && len(token) > 4 {
		candidates = append(candidates, token[:len(token)-4])
	}

	if strings.HasSuffix(token, "izing") && len(token) > 5 {
		candidates = append(candidates, token[:len(token)-5])
	}

	if strings.HasSuffix(token, "izer") && len(token) > 4 {
		candidates = append(candidates, token[:len(token)-4])
	}

	if strings.HasSuffix(token, "ization") && len(token) > 7 {
		candidates = append(candidates, token[:len(token)-7])
	}

	return candidates
}

func spellingVariantCandidates(token string) []string {
	candidates := make([]string, 0, 8)

	appendSuffixVariant(&candidates, token, "isation", "ization")
	appendSuffixVariant(&candidates, token, "ization", "isation")
	appendSuffixVariant(&candidates, token, "ising", "izing")
	appendSuffixVariant(&candidates, token, "izing", "ising")
	appendSuffixVariant(&candidates, token, "ised", "ized")
	appendSuffixVariant(&candidates, token, "ized", "ised")
	appendSuffixVariant(&candidates, token, "iser", "izer")
	appendSuffixVariant(&candidates, token, "izer", "iser")
	appendSuffixVariant(&candidates, token, "ise", "ize")
	appendSuffixVariant(&candidates, token, "ize", "ise")
	appendSuffixVariant(&candidates, token, "our", "or")
	appendSuffixVariant(&candidates, token, "or", "our")
	appendSuffixVariant(&candidates, token, "tre", "ter")
	appendSuffixVariant(&candidates, token, "ter", "tre")

	return candidates
}

func appendSuffixVariant(candidates *[]string, token string, fromSuffix string, toSuffix string) {
	if !strings.HasSuffix(token, fromSuffix) || len(token) <= len(fromSuffix) {
		return
	}

	root := token[:len(token)-len(fromSuffix)]
	*candidates = append(*candidates, root+toSuffix)
}

func makeWordSet(words []string) map[string]struct{} {
	set := make(map[string]struct{}, len(words))

	for _, word := range words {
		set[word] = struct{}{}
	}

	return set
}

func makeWordsByFirstCharacter(words []string) map[rune][]string {
	grouped := make(map[rune][]string)

	for _, word := range words {
		firstCharacter, size := utf8.DecodeRuneInString(word)

		if firstCharacter == utf8.RuneError && size == 0 {
			continue
		}

		grouped[firstCharacter] = append(grouped[firstCharacter], word)
	}

	for firstCharacter := range grouped {
		sort.Strings(grouped[firstCharacter])
	}

	return grouped
}

func loadWords() ([]string, error) {
	configuredDictionaryPaths := parseDictionaryPaths(os.Getenv("KIVIA_DICTIONARY_PATH"))

	if len(configuredDictionaryPaths) > 0 {
		words, err := loadWordsFromPaths(configuredDictionaryPaths, true)

		if err != nil {
			return nil, err
		}

		if len(words) == 0 {
			return nil, errors.New("configured dictionary sources contain no usable words")
		}

		return words, nil
	}

	words, err := loadWordsFromPaths(defaultDictionaryPaths, false)

	if err != nil {
		return nil, err
	}

	if len(words) == 0 {
		return nil, errors.New("no usable dictionary words found; set KIVIA_DICTIONARY_PATH")
	}

	return words, nil
}

func readWordsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	words := make([]string, 0, 1024)
	scanner := bufio.NewScanner(file)
	isSpellDictionaryFile := strings.EqualFold(path.Ext(filePath), ".dic")
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		line := normalizeDictionaryLine(scanner.Text(), lineNumber, isSpellDictionaryFile)

		if line == "" {
			continue
		}

		words = append(words, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return normalizeWords(words), nil
}

func parseDictionaryPaths(value string) []string {
	trimmedValue := strings.TrimSpace(value)

	if trimmedValue == "" {
		return nil
	}

	expandedValue := strings.ReplaceAll(trimmedValue, ",", string(os.PathListSeparator))
	parts := strings.Split(expandedValue, string(os.PathListSeparator))
	paths := make([]string, 0, len(parts))

	for _, entry := range parts {
		candidate := strings.TrimSpace(entry)

		if candidate == "" {
			continue
		}

		paths = append(paths, candidate)
	}

	return paths
}

func loadWordsFromPaths(paths []string, strict bool) ([]string, error) {
	combinedWords := make([]string, 0, 4096)

	for _, dictionaryPath := range paths {
		words, err := readWordsFromFile(dictionaryPath)

		if err != nil {
			if strict {
				return nil, fmt.Errorf("failed to read dictionary %q: %w", dictionaryPath, err)
			}

			continue
		}

		combinedWords = append(combinedWords, words...)
	}

	return normalizeWords(combinedWords), nil
}

func normalizeDictionaryLine(line string, lineNumber int, isSpellDictionaryFile bool) string {
	trimmedLine := strings.TrimSpace(line)

	if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
		return ""
	}

	if isSpellDictionaryFile && lineNumber == 1 {
		if _, err := strconv.Atoi(trimmedLine); err == nil {
			return ""
		}
	}

	if slashIndex := strings.Index(trimmedLine, "/"); slashIndex >= 0 {
		trimmedLine = trimmedLine[:slashIndex]
	}

	return trimmedLine
}

func normalizeWords(words []string) []string {
	unique := make(map[string]struct{}, len(words))

	for _, word := range words {
		normalized := normalizeToken(word)

		if normalized == "" {
			continue
		}

		if len(normalized) <= 1 {
			continue
		}

		unique[normalized] = struct{}{}
	}

	output := make([]string, 0, len(unique))

	for word := range unique {
		output = append(output, word)
	}

	sort.Strings(output)

	return output
}

func normalizeToken(token string) string {
	token = strings.ToLower(strings.TrimSpace(token))

	if token == "" {
		return ""
	}

	match := wordPattern.FindString(token)

	if match == "" {
		return ""
	}

	return match
}

func cachePath() (string, error) {
	base, err := os.UserCacheDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(base, "kivia", "fuzzy_model_v1.json"), nil
}

func loadCachedModel() (*fuzzy.Model, error) {
	path, err := cachePath()

	if err != nil {
		return nil, err
	}

	model, err := fuzzy.Load(path)

	if err != nil {
		return nil, err
	}

	return model, nil
}

func saveCachedModel(model *fuzzy.Model) error {
	if model == nil {
		return errors.New("Model cannot be nil.")
	}

	path, err := cachePath()

	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return model.Save(path)
}

var defaultDictionaryPaths = []string{
	"/usr/share/dict/words",
	"/usr/dict/words",
	"/usr/share/dict/web2",
	"/usr/share/dict/web2a",
	"/usr/share/dict/propernames",
	"/usr/share/dict/connectives",
	"/usr/share/hunspell/en_US.dic",
	"/usr/share/hunspell/en_GB.dic",
	"/usr/share/hunspell/en_CA.dic",
	"/usr/share/hunspell/en_AU.dic",
	"/usr/share/myspell/en_US.dic",
	"/usr/share/myspell/en_GB.dic",
	"/opt/homebrew/share/hunspell/en_US.dic",
	"/opt/homebrew/share/hunspell/en_GB.dic",
	"/usr/local/share/hunspell/en_US.dic",
	"/usr/local/share/hunspell/en_GB.dic",
}
