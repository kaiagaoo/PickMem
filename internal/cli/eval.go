package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kaiagaoo/PickMem/internal/evaluation"
	"github.com/spf13/cobra"
)

func newEvalCmd() *cobra.Command {
	var datasetPath string
	var topK int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Run the offline context-retrieval evaluation",
		Long: `Evaluate context-ranking strategies on a labeled JSON dataset.
The bundled dataset is synthetic and serves as a reproducible regression suite;
use --dataset with real, consented examples before making quality claims.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var data []byte
			var err error
			if datasetPath == "" {
				data, err = evaluation.DefaultDataset()
			} else {
				data, err = os.ReadFile(datasetPath)
			}
			if err != nil {
				return err
			}
			report, err := evaluation.Run(data, topK)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Dataset: %s (%d cases, %d notes)\n", report.Dataset, report.Cases, report.Notes)
			fmt.Fprintf(cmd.OutOrStdout(), "Metric cutoff: k=%d\n\n", report.TopK)
			fmt.Fprintln(cmd.OutOrStdout(), "strategy                 precision@k  recall@k  MRR    context reduction")
			for _, result := range report.Results {
				m := result.Metrics
				fmt.Fprintf(cmd.OutOrStdout(), "%-24s %.3f        %.3f     %.3f  %.1f%%\n",
					result.Name, m.PrecisionAtK, m.RecallAtK, m.MeanReciprocal, m.ContextReduction*100)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "\nCaveat:", report.Caveat)
			return nil
		},
	}
	cmd.Flags().StringVar(&datasetPath, "dataset", "", "path to a labeled evaluation dataset (defaults to bundled synthetic data)")
	cmd.Flags().IntVar(&topK, "top", 3, "retrieval cutoff used for metrics")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the report as structured JSON")
	return cmd
}
