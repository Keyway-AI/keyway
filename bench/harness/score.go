package main

// Scorecard holds accuracy metrics for one layer (PRD §13.3).
type Scorecard struct {
	Layer          string                `json:"layer"` // L1|L2|L3|L4
	TP, FP, TN, FN int                   `json:"-"`
	TruePositives  int                   `json:"tp"`
	FalsePositives int                   `json:"fp"`
	TrueNegatives  int                   `json:"tn"`
	FalseNegatives int                   `json:"fn"`
	TPR            float64               `json:"tpr"`
	FPR            float64               `json:"fpr"`
	Precision      float64               `json:"precision"`
	Recall         float64               `json:"recall"`
	F1             float64               `json:"f1"`
	Youden         float64               `json:"youden"`
	PerClass       map[string]ClassScore `json:"per_class,omitempty"`
}

// ClassScore is a per-classification breakdown.
type ClassScore struct {
	TP        int     `json:"tp"`
	FP        int     `json:"fp"`
	TN        int     `json:"tn"`
	FN        int     `json:"fn"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
}

// Compute derives the rate metrics from the confusion-matrix counts. It is the
// headline math the CI gate checks against PRD §13.4.
func (s *Scorecard) Compute() {
	s.TruePositives, s.FalsePositives = s.TP, s.FP
	s.TrueNegatives, s.FalseNegatives = s.TN, s.FN
	s.TPR = ratio(s.TP, s.TP+s.FN)
	s.FPR = ratio(s.FP, s.FP+s.TN)
	s.Precision = ratio(s.TP, s.TP+s.FP)
	s.Recall = s.TPR
	if s.Precision+s.Recall > 0 {
		s.F1 = 2 * s.Precision * s.Recall / (s.Precision + s.Recall)
	}
	s.Youden = s.TPR - s.FPR // headline metric (PRD §13.3)
}

func ratio(num, denom int) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) / float64(denom)
}

// Gate is a single CI threshold (PRD §13.4).
type Gate struct {
	Layer   string
	Metric  string
	Get     func(Scorecard) float64
	FailIf  func(v float64) bool
	Explain string
}

// DefaultGates encodes the PRD §13.4 fail thresholds.
func DefaultGates() []Gate {
	return []Gate{
		{"L1", "derivation recall", func(s Scorecard) float64 { return s.Recall }, func(v float64) bool { return v < 0.70 }, "consumers found / actual < 70%"},
		{"L2", "probe accuracy", func(s Scorecard) float64 { return s.Precision }, func(v float64) bool { return v < 0.95 }, "correct verdicts < 95%"},
		{"L3", "diff FPR", func(s Scorecard) float64 { return s.FPR }, func(v float64) bool { return v > 0.05 }, "alerts on no-op commits > 5%"},
		{"L3", "Youden", func(s Scorecard) float64 { return s.Youden }, func(v float64) bool { return v < 0.70 }, "TPR - FPR < 0.70"},
		{"L4", "attribution", func(s Scorecard) float64 { return s.Precision }, func(v float64) bool { return v < 0.60 }, "correctly bound < 60%"},
	}
}
