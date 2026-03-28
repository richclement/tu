package count

import (
	"errors"
	"unicode/utf8"

	"github.com/tiktoken-go/tokenizer"

	"github.com/richclement/tu/internal/report"
)

const (
	ExactProvider     = "openai"
	HeuristicProvider = "heuristic"
)

type Result struct {
	Tokens   int64
	Method   report.Method
	Provider string
}

type textCounter interface {
	CountText(text string) (Result, error)
}

type Counter struct {
	exact     textCounter
	heuristic textCounter
}

func NewCounter() *Counter {
	exact, err := NewExactLocalCounter()
	if err != nil {
		exact = nil
	}

	return &Counter{
		exact:     exact,
		heuristic: HeuristicCounter{},
	}
}

func NewCounterWithImplementations(exact textCounter, heuristic textCounter) *Counter {
	if heuristic == nil {
		heuristic = HeuristicCounter{}
	}

	return &Counter{
		exact:     exact,
		heuristic: heuristic,
	}
}

func (c *Counter) CountText(text string) Result {
	if c != nil && c.exact != nil {
		result, err := c.exact.CountText(text)
		if err == nil {
			return result
		}
	}

	result, err := c.heuristic.CountText(text)
	if err != nil {
		return Result{
			Tokens:   0,
			Method:   report.MethodHeuristic,
			Provider: HeuristicProvider,
		}
	}

	return result
}

type ExactLocalCounter struct {
	codec tokenizer.Codec
}

func NewExactLocalCounter() (*ExactLocalCounter, error) {
	codec, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, err
	}

	return &ExactLocalCounter{codec: codec}, nil
}

func (c *ExactLocalCounter) CountText(text string) (Result, error) {
	if c == nil || c.codec == nil {
		return Result{}, errors.New("exact local counter is not initialized")
	}

	tokenCount, err := c.codec.Count(text)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Tokens:   int64(tokenCount),
		Method:   report.MethodExact,
		Provider: ExactProvider,
	}, nil
}

type HeuristicCounter struct{}

func (HeuristicCounter) CountText(text string) (Result, error) {
	return Result{
		Tokens:   estimateTokens(text),
		Method:   report.MethodHeuristic,
		Provider: HeuristicProvider,
	}, nil
}

func estimateTokens(text string) int64 {
	if text == "" {
		return 0
	}

	runeCount := utf8.RuneCountInString(text)
	tokens := int64(runeCount / 4)
	if runeCount%4 != 0 {
		tokens++
	}
	if tokens == 0 {
		return 1
	}

	return tokens
}
