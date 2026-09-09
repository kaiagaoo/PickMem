package evaluation

import "testing"

func TestBundledDatasetRuns(t *testing.T) {
	data, err := DefaultDataset()
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(data, 3)
	if err != nil {
		t.Fatal(err)
	}
	if report.Cases < 8 || len(report.Results) != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}
	for _, result := range report.Results {
		if result.Metrics.RecallAtK < 0 || result.Metrics.RecallAtK > 1 {
			t.Fatalf("invalid metrics: %+v", result)
		}
	}
}

func TestInvalidDatasetIsRejected(t *testing.T) {
	if _, err := Run([]byte(`{"notes":[],"cases":[]}`), 3); err == nil {
		t.Fatal("Run() accepted an empty dataset")
	}
}
