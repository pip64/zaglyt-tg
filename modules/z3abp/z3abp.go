package z3abp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/bbalet/stopwords"
	"github.com/crawlab-team/bm25"
	"github.com/kljensen/snowball"
)

type MyStemWrapper struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Scanner
	mu  sync.Mutex
}

type MystemItem struct {
	Analysis []struct {
		Lex string `json:"lex"`
		Gr  string `json:"gr"`
	} `json:"analysis"`
	Text string `json:"text"`
}

var Mystem *MyStemWrapper

func StartMyStem() (*MyStemWrapper, error) {
	binPath := "mystem"
	if _, err := os.Stat("./mystem"); err == nil {
		binPath = "./mystem"
	}

	cmd := exec.Command(binPath, "-icd", "--format", "json")

	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(outPipe)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	return &MyStemWrapper{
		cmd: cmd,
		in:  in,
		out: scanner,
	}, nil
}

func (m *MyStemWrapper) Analyze(text string) ([]MystemItem, error) {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.TrimSpace(text)

	if text == "" {
		return nil, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.in.Write([]byte(text + "\n"))
	if err != nil {
		return nil, err
	}

	if m.out.Scan() {
		line := m.out.Bytes()
		var res []MystemItem
		if err := json.Unmarshal(line, &res); err != nil {
			return nil, err
		}
		return res, nil
	}

	return nil, m.out.Err()
}

func init() {
	rand.Seed(time.Now().UnixNano())

	var err error
	Mystem, err = StartMyStem()
	if err != nil {
		log.Fatalf("failed to start mystem: %v", err)
	}
}

type Config struct {
	BM25K1      float64
	BM25B       float64
	TopKForGen  int
	MinGenWords int
	MaxGenWords int
}

func DefaultConfig() Config {
	return Config{
		BM25K1:      1.5,
		BM25B:       0.75,
		TopKForGen:  5,
		MinGenWords: 2,
		MaxGenWords: 10,
	}
}

type DocScore struct {
	Document string
	Score    float64
}

func Tokenize(s string) []string {
	s = strings.ToLower(s)
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

func StemWords(words []string) []string {
	stemmed := make([]string, 0, len(words))
	for _, word := range words {
		stem, _ := snowball.Stem(word, "russian", true)
		stemmed = append(stemmed, stem)
	}
	return stemmed
}

func ContainsQuery(line, query string) bool {
	line = strings.ToLower(strings.TrimSpace(line))
	query = strings.ToLower(strings.TrimSpace(query))
	return line == query
}

func FilterLines(db []string, stems []string, query string) []string {
	var foundLines []string

	limit := 5000
	startIdx := 0
	if len(db) > limit {
		startIdx = len(db) - limit
	}

	for _, line := range db[startIdx:] {
		if ContainsQuery(line, query) {
			continue
		}

		words := Tokenize(line)
		lineStems := StemWords(words)
		matched := false

		for _, wordStem := range lineStems {
			for _, queryStem := range stems {
				if wordStem == queryStem {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}

		if matched {
			foundLines = append(foundLines, line)
		}
	}
	return foundLines
}

func RankDocuments(docs []string, query []string, k1, b float64) ([]DocScore, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	bm, err := bm25.NewBM25Okapi(docs, Tokenize, k1, b, nil)
	if err != nil {
		return nil, err
	}

	scores, err := bm.GetScores(query)
	if err != nil {
		return nil, err
	}

	rankedResults := make([]DocScore, len(scores))
	for i, score := range scores {
		rankedResults[i] = DocScore{
			Document: docs[i],
			Score:    score,
		}
	}

	sort.Slice(rankedResults, func(i, j int) bool {
		return rankedResults[i].Score > rankedResults[j].Score
	})

	return rankedResults, nil
}

type MorphKey struct {
	Lemma   string
	Grammar string
}

type MorphToken struct {
	Original string
	Key      MorphKey
}

type MorphMarkovModel struct {
	Starts2      [][2]MorphToken
	Starts1      []MorphToken
	Transitions2 map[[2]MorphKey][]MorphToken
	Transitions1 map[MorphKey][]MorphToken
}

func AnalyzeText(text string) []MorphToken {
	resp, err := Mystem.Analyze(text)
	if err != nil || len(resp) == 0 {
		return nil
	}

	var tokens []MorphToken
	for _, item := range resp {
		if len(item.Analysis) > 0 {
			analysis := item.Analysis[0]
			tokens = append(tokens, MorphToken{
				Original: item.Text,
				Key: MorphKey{
					Lemma:   analysis.Lex,
					Grammar: analysis.Gr,
				},
			})
		} else {
			punc := strings.TrimSpace(item.Text)
			if punc != "" && len(tokens) > 0 {
				tokens[len(tokens)-1].Original += punc
			}
		}
	}
	return tokens
}

func buildMorphMarkovChain(docs []string) MorphMarkovModel {
	model := MorphMarkovModel{
		Transitions2: make(map[[2]MorphKey][]MorphToken),
		Transitions1: make(map[MorphKey][]MorphToken),
	}

	for _, doc := range docs {
		tokens := AnalyzeText(doc)
		if len(tokens) == 0 {
			continue
		}

		if len(tokens) == 1 {
			model.Starts1 = append(model.Starts1, tokens[0])
			model.Transitions1[tokens[0].Key] = append(model.Transitions1[tokens[0].Key], MorphToken{})
			continue
		}

		model.Starts2 = append(model.Starts2, [2]MorphToken{tokens[0], tokens[1]})
		model.Starts1 = append(model.Starts1, tokens[0])

		model.Transitions1[tokens[0].Key] = append(model.Transitions1[tokens[0].Key], tokens[1])

		for i := 0; i < len(tokens)-2; i++ {
			t1, t2, t3 := tokens[i], tokens[i+1], tokens[i+2]

			key2 := [2]MorphKey{t1.Key, t2.Key}
			key1 := t2.Key

			model.Transitions2[key2] = append(model.Transitions2[key2], t3)
			model.Transitions1[key1] = append(model.Transitions1[key1], t3)
		}

		lastToken := tokens[len(tokens)-1]
		prevToken := tokens[len(tokens)-2]

		keyLast2 := [2]MorphKey{prevToken.Key, lastToken.Key}
		keyLast1 := lastToken.Key

		model.Transitions2[keyLast2] = append(model.Transitions2[keyLast2], MorphToken{})
		model.Transitions1[keyLast1] = append(model.Transitions1[keyLast1], MorphToken{})
	}

	return model
}

func generateMorphMarkovText(model MorphMarkovModel, minWords, maxWords int) string {
	if len(model.Starts1) == 0 && len(model.Starts2) == 0 {
		return ""
	}

	for attempt := 0; attempt < 50; attempt++ {
		var result []string
		var currKey1 MorphKey
		var currKey2 [2]MorphKey
		useOrder2 := false

		if len(model.Starts2) > 0 && rand.Float32() < 0.8 {
			start := model.Starts2[rand.Intn(len(model.Starts2))]
			result = append(result, start[0].Original, start[1].Original)
			currKey2 = [2]MorphKey{start[0].Key, start[1].Key}
			currKey1 = start[1].Key
			useOrder2 = true
		} else if len(model.Starts1) > 0 {
			start := model.Starts1[rand.Intn(len(model.Starts1))]
			result = append(result, start.Original)
			currKey1 = start.Key
		} else {
			continue
		}

		for len(result) < maxWords {
			var nextTokens []MorphToken
			var exists bool

			if useOrder2 {
				nextTokens, exists = model.Transitions2[currKey2]
			}

			if !exists || len(nextTokens) == 0 || rand.Float32() < 0.15 {
				altTokens, altExists := model.Transitions1[currKey1]
				if altExists && len(altTokens) > 0 {
					nextTokens = altTokens
					exists = true
				}
			}

			if !exists || len(nextTokens) == 0 {
				break
			}

			nextToken := nextTokens[rand.Intn(len(nextTokens))]
			if nextToken.Original == "" {
				break
			}

			result = append(result, nextToken.Original)

			if len(result) >= 2 {
				currKey2 = [2]MorphKey{currKey1, nextToken.Key}
				useOrder2 = true
			}
			currKey1 = nextToken.Key
		}

		if len(result) >= minWords && len(result) <= maxWords {
			sentence := strings.Join(result, " ")
			cleaned := cleanTrailingPunctuation(sentence)
			if cleaned != "" {
				return cleaned
			}
		}
	}

	return ""
}

func cleanTrailingPunctuation(s string) string {
	s = strings.TrimRight(s, " ,-;:«»'\"")
	words := strings.Fields(s)

	for len(words) > 0 {
		lastWord := strings.ToLower(words[len(words)-1])
		lastWordNorm := strings.Trim(lastWord, ".,!?:;\"'()[]{}«»")

		cleaned := stopwords.CleanString(lastWordNorm, "ru", false)

		if strings.TrimSpace(cleaned) == "" {
			words = words[:len(words)-1]
		} else {
			break
		}
	}

	if len(words) == 0 {
		return ""
	}

	res := strings.Join(words, " ")
	return strings.TrimRight(res, " ,-;:«»'\"")
}

func GenerateBestResponse(queryStr string, db []string, cfg Config) (string, error) {
	queryWords := Tokenize(queryStr)
	stemmedQuery := StemWords(queryWords)

	matchedLines := FilterLines(
		db,
		stemmedQuery,
		queryStr,
	)

	if len(matchedLines) == 0 {
		return "", fmt.Errorf("matched lines not found")
	}

	rankedDb, err := RankDocuments(
		matchedLines,
		queryWords,
		cfg.BM25K1,
		cfg.BM25B,
	)

	if err != nil {
		return "", fmt.Errorf("sort error: %w", err)
	}

	topCount := cfg.TopKForGen
	if len(rankedDb) < topCount {
		topCount = len(rankedDb)
	}

	var topDocs []string
	for _, doc := range rankedDb[:topCount] {
		topDocs = append(topDocs, doc.Document)
	}

	model := buildMorphMarkovChain(topDocs)

	generatedText := generateMorphMarkovText(model, cfg.MinGenWords, cfg.MaxGenWords)

	if generatedText == "" {
		bestDoc := rankedDb[0].Document
		words := strings.Fields(bestDoc)
		limit := cfg.MaxGenWords
		if len(words) < limit {
			limit = len(words)
		}
		if limit == 0 {
			return "", fmt.Errorf("best document is empty")
		}
		generatedText = cleanTrailingPunctuation(strings.Join(words[:limit], " "))
	}

	return generatedText, nil
}

func GenerateRandomMarkov(db []string, minWords, maxWords int) string {
	if len(db) == 0 {
		return ""
	}

	limit := 50
	if len(db) < limit {
		limit = len(db)
	}

	randDb := make([]string, len(db))
	copy(randDb, db)
	rand.Shuffle(len(randDb), func(i, j int) {
		randDb[i], randDb[j] = randDb[j], randDb[i]
	})

	subset := randDb[:limit]

	model := buildMorphMarkovChain(subset)
	return generateMorphMarkovText(model, minWords, maxWords)
}
