// Package evaluation provides a reproducible offline benchmark for PickMem's
// context recommendation strategies. The bundled dataset is synthetic and is
// intended as a regression suite, not as evidence of production accuracy.
package evaluation

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/kaiagaoo/PickMem/internal/retrieval"
	"github.com/kaiagaoo/PickMem/internal/vault"
)

//go:embed testdata/retrieval.json
var defaultDataset []byte

type Dataset struct {
	Name  string `json:"name"`
	Notes []struct {
		ID    string   `json:"id"`
		Label string   `json:"label"`
		Group string   `json:"group"`
		Tags  []string `json:"tags"`
		Body  string   `json:"body"`
	} `json:"notes"`
	Cases []struct {
		Query       string   `json:"query"`
		RelevantIDs []string `json:"relevant_ids"`
	} `json:"cases"`
}

type Metrics struct {
	PrecisionAtK     float64 `json:"precision_at_k"`
	RecallAtK        float64 `json:"recall_at_k"`
	MeanReciprocal   float64 `json:"mean_reciprocal_rank"`
	ContextReduction float64 `json:"context_reduction"`
}

type StrategyResult struct {
	Name    string  `json:"name"`
	Metrics Metrics `json:"metrics"`
}

type Report struct {
	Dataset string           `json:"dataset"`
	Cases   int              `json:"cases"`
	Notes   int              `json:"notes"`
	TopK    int              `json:"top_k"`
	Caveat  string           `json:"caveat"`
	Results []StrategyResult `json:"results"`
}

func DefaultDataset() ([]byte, error) {
	out := make([]byte, len(defaultDataset))
	copy(out, defaultDataset)
	return out, nil
}

func Run(data []byte, topK int) (Report, error) {
	if topK < 1 {
		return Report{}, fmt.Errorf("top_k must be at least 1")
	}
	var ds Dataset
	if err := json.Unmarshal(data, &ds); err != nil {
		return Report{}, fmt.Errorf("decode evaluation dataset: %w", err)
	}
	if len(ds.Notes) == 0 || len(ds.Cases) == 0 {
		return Report{}, fmt.Errorf("dataset requires notes and cases")
	}
	notes := make([]*vault.Note, 0, len(ds.Notes))
	seen := map[string]bool{}
	for _, item := range ds.Notes {
		if item.ID == "" || seen[item.ID] {
			return Report{}, fmt.Errorf("note ids must be non-empty and unique: %q", item.ID)
		}
		seen[item.ID] = true
		notes = append(notes, &vault.Note{Frontmatter: vault.Frontmatter{
			ID: item.ID, Label: item.Label, Group: item.Group, Tags: item.Tags,
			Status: vault.StatusActive,
		}, Body: item.Body})
	}

	rankers := []struct {
		name string
		fn   func(string, []*vault.Note, int) []retrieval.Match
	}{
		{"field_weighted_bm25", retrieval.Rank},
		{"token_overlap_baseline", retrieval.RankOverlap},
	}
	report := Report{
		Dataset: ds.Name, Cases: len(ds.Cases), Notes: len(ds.Notes), TopK: topK,
		Caveat: "Synthetic regression dataset; do not present these scores as production accuracy.",
	}
	for _, strategy := range rankers {
		var precision, recall, reciprocal, reduction float64
		for _, c := range ds.Cases {
			relevant := map[string]bool{}
			for _, id := range c.RelevantIDs {
				relevant[id] = true
			}
			if len(relevant) == 0 {
				return Report{}, fmt.Errorf("case %q has no relevant_ids", c.Query)
			}
			matches := strategy.fn(c.Query, notes, topK)
			hits := 0
			firstRank := 0
			for i, match := range matches {
				if relevant[match.Note.ID] {
					hits++
					if firstRank == 0 {
						firstRank = i + 1
					}
				}
			}
			precision += float64(hits) / float64(topK)
			recall += float64(hits) / float64(len(relevant))
			if firstRank > 0 {
				reciprocal += 1 / float64(firstRank)
			}
			reduction += 1 - float64(len(matches))/float64(len(notes))
		}
		n := float64(len(ds.Cases))
		report.Results = append(report.Results, StrategyResult{Name: strategy.name, Metrics: Metrics{
			PrecisionAtK: precision / n, RecallAtK: recall / n,
			MeanReciprocal: reciprocal / n, ContextReduction: reduction / n,
		}})
	}
	return report, nil
}
