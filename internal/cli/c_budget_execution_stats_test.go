package cli

import (
	"math"
	"slices"
	"testing"
)

const executionPairs = 12
const executionBudget = 1.08

// A conservative measurement-quality heuristic, not the performance budget
// and not a confidence interval. Fix it before inspecting measured data.
const executionSpanLimit = 0.10

type executionDistribution struct {
	Median       float64 `json:"median"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	MAD          float64 `json:"median_absolute_deviation"`
	RelativeSpan float64 `json:"relative_full_span"`
}

type executionBudgetSummary struct {
	C                executionDistribution `json:"c_ns"`
	Fairy            executionDistribution `json:"fairy_ns"`
	Paired           executionDistribution `json:"paired_ratio"`
	Ratios           []float64             `json:"paired_ratios_in_order"`
	RatioOfMedians   float64               `json:"ratio_of_medians"`
	CFirstMedian     float64               `json:"c_first_ratio_median"`
	FairyFirstMedian float64               `json:"fairy_first_ratio_median"`
	Status           string                `json:"status"`
	Reasons          []string              `json:"reasons"`
}

func executionMedian(values []float64) float64 {
	copy := slices.Clone(values)
	slices.Sort(copy)
	n := len(copy)
	return (copy[(n-1)/2] + copy[n/2]) / 2
}

func executionDescribe(values []float64) executionDistribution {
	median := executionMedian(values)
	deviations := make([]float64, len(values))
	for i, value := range values {
		deviations[i] = math.Abs(value - median)
	}
	low, high := slices.Min(values), slices.Max(values)
	return executionDistribution{Median: median, Min: low, Max: high, MAD: executionMedian(deviations), RelativeSpan: (high - low) / median}
}

func summarizeExecutionBudget(c, f []float64) executionBudgetSummary {
	result := executionBudgetSummary{Status: "inconclusive", Reasons: []string{}}
	if len(c) != executionPairs || len(f) != executionPairs {
		result.Reasons = append(result.Reasons, "incomplete pairs")
		return result
	}
	for i := range c {
		if c[i] <= 0 || f[i] <= 0 || math.IsNaN(c[i]) || math.IsNaN(f[i]) || math.IsInf(c[i], 0) || math.IsInf(f[i], 0) {
			result.Reasons = append(result.Reasons, "invalid durations")
			return result
		}
	}
	result.C, result.Fairy = executionDescribe(c), executionDescribe(f)
	var cFirst, fFirst []float64
	for i := range c {
		ratio := f[i] / c[i]
		result.Ratios = append(result.Ratios, ratio)
		if i%2 == 0 {
			fFirst = append(fFirst, ratio)
		} else {
			cFirst = append(cFirst, ratio)
		}
	}
	result.Paired = executionDescribe(result.Ratios)
	result.RatioOfMedians = result.Fairy.Median / result.C.Median
	result.CFirstMedian, result.FairyFirstMedian = executionMedian(cFirst), executionMedian(fFirst)
	if result.C.RelativeSpan > executionSpanLimit || result.Fairy.RelativeSpan > executionSpanLimit || result.Paired.RelativeSpan > executionSpanLimit {
		result.Reasons = append(result.Reasons, "relative full span exceeds predeclared 10% stability limit")
	}
	within := result.Paired.Median <= executionBudget
	if (result.RatioOfMedians <= executionBudget) != within || (result.CFirstMedian <= executionBudget) != within || (result.FairyFirstMedian <= executionBudget) != within {
		result.Reasons = append(result.Reasons, "estimators or execution-order groups disagree at 8% budget")
	}
	if len(result.Reasons) == 0 {
		if within {
			result.Status = "observed-within-budget"
		} else {
			result.Status = "observed-exceeds-budget"
		}
	}
	return result
}

func TestCBudgetExecutionSummary(t *testing.T) {
	constant := func(value float64) []float64 {
		v := make([]float64, executionPairs)
		for i := range v {
			v[i] = value
		}
		return v
	}
	for _, tc := range []struct {
		name string
		c, f []float64
		want string
	}{
		{"within", constant(100), constant(105), "observed-within-budget"},
		{"exceeds", constant(100), constant(112), "observed-exceeds-budget"},
		{"incomplete", constant(100)[:11], constant(105), "inconclusive"},
		{"invalid", constant(0), constant(105), "inconclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := summarizeExecutionBudget(tc.c, tc.f); got.Status != tc.want {
				t.Fatalf("%+v", got)
			}
		})
	}
	c, f := constant(100), constant(105)
	f[4] = 150
	if got := summarizeExecutionBudget(c, f); got.Status != "inconclusive" || got.Paired.Median != 1.05 {
		t.Fatalf("outlier must not be hidden by median: %+v", got)
	}
	f = constant(107)
	for i := range f {
		if i%2 == 0 {
			f[i] = 109
		}
	}
	if got := summarizeExecutionBudget(c, f); got.Status != "inconclusive" {
		t.Fatalf("order bias: %+v", got)
	}
	d := executionDescribe([]float64{1, 2, 3, 4})
	if d.Median != 2.5 || d.MAD != 1 || d.Min != 1 || d.Max != 4 || d.RelativeSpan != 1.2 {
		t.Fatalf("distribution: %+v", d)
	}
}
