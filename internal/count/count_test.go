package count

import (
	"errors"
	"testing"

	"github.com/richclement/tu/internal/report"
)

func TestCounterUsesExactPathWhenAvailable(t *testing.T) {
	t.Parallel()

	counter := NewCounter()
	result := counter.CountText("Hello, world!")

	if result.Method != report.MethodExact {
		t.Fatalf("expected exact method, got %q", result.Method)
	}
	if result.Provider != ExactProvider {
		t.Fatalf("expected exact provider %q, got %q", ExactProvider, result.Provider)
	}
	if result.Tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", result.Tokens)
	}
}

func TestCounterFallsBackToHeuristicWhenExactFails(t *testing.T) {
	t.Parallel()

	counter := NewCounterWithImplementations(
		stubCounter{err: errors.New("boom")},
		stubCounter{result: Result{Tokens: 12, Method: report.MethodHeuristic, Provider: HeuristicProvider}},
	)

	result := counter.CountText("fallback")
	if result.Method != report.MethodHeuristic {
		t.Fatalf("expected heuristic method, got %q", result.Method)
	}
	if result.Provider != HeuristicProvider {
		t.Fatalf("expected heuristic provider %q, got %q", HeuristicProvider, result.Provider)
	}
	if result.Tokens != 12 {
		t.Fatalf("expected fallback token count 12, got %d", result.Tokens)
	}
}

func TestHeuristicCounterCountsRunes(t *testing.T) {
	t.Parallel()

	result, err := HeuristicCounter{}.CountText("abcdé")
	if err != nil {
		t.Fatalf("heuristic counter returned error: %v", err)
	}

	if result.Method != report.MethodHeuristic {
		t.Fatalf("expected heuristic method, got %q", result.Method)
	}
	if result.Provider != HeuristicProvider {
		t.Fatalf("expected heuristic provider %q, got %q", HeuristicProvider, result.Provider)
	}
	if result.Tokens != 2 {
		t.Fatalf("expected 2 heuristic tokens, got %d", result.Tokens)
	}
}

type stubCounter struct {
	result Result
	err    error
}

func (s stubCounter) CountText(string) (Result, error) {
	return s.result, s.err
}
