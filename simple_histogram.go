// Package simple_histogram buckets numeric values into configurable bins
// and reports counts and percentages per bin.
//
// The design decision that matters most here: bins are half-open on the
// right, [lo, hi), except the final bin which is closed on the right so the
// maximum value lands somewhere. Without that exception the top edge of the
// range would silently drop values, which is a common source of off-by-one
// pain in histograms.
package simple_histogram

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Bin is a single bucket with an inclusive lower edge and an exclusive upper
// edge, except for the last bin in a Histogram which also includes its upper
// edge.
type Bin struct {
	Low   float64
	High  float64
	Count int
}

// Histogram holds an ordered set of bins and the total number of observations.
type Histogram struct {
	bins  []Bin
	total int
}

// New creates a Histogram from a sorted slice of bin edges.
//
// Edges must contain at least two values, be strictly increasing, and contain
// no NaN. The number of bins is len(edges)-1. Each bin i covers
// [edges[i], edges[i+1]) with the final bin closed on the right.
func New(edges []float64) (*Histogram, error) {
	if len(edges) < 2 {
		return nil, errors.New("simple_histogram: need at least two edges")
	}
	for i, e := range edges {
		if math.IsNaN(e) {
			return nil, fmt.Errorf("simple_histogram: edge %d is NaN", i)
		}
	}
	for i := 1; i < len(edges); i++ {
		if !(edges[i] > edges[i-1]) {
			return nil, fmt.Errorf("simple_histogram: edges must be strictly increasing, got %v then %v", edges[i-1], edges[i])
		}
	}
	bins := make([]Bin, len(edges)-1)
	for i := range bins {
		bins[i] = Bin{Low: edges[i], High: edges[i+1]}
	}
	return &Histogram{bins: bins}, nil
}

// NewUniform builds a Histogram with n equal-width bins spanning [min, max].
//
// n must be positive and max must be greater than min. Equal-width bins are
// the common case and worth a helper because getting the edge arithmetic
// right by hand is easy to fumble.
func NewUniform(min, max float64, n int) (*Histogram, error) {
	if n <= 0 {
		return nil, fmt.Errorf("simple_histogram: bin count must be positive, got %d", n)
	}
	if math.IsNaN(min) || math.IsNaN(max) {
		return nil, errors.New("simple_histogram: bounds must not be NaN")
	}
	if !(max > min) {
		return nil, fmt.Errorf("simple_histogram: max must be greater than min, got %v and %v", min, max)
	}
	edges := make([]float64, n+1)
	width := (max - min) / float64(n)
	for i := range edges {
		edges[i] = min + float64(i)*width
	}
	// Pin the last edge exactly to max so floating-point drift does not leave a
	// sliver of the range uncovered.
	edges[n] = max
	return New(edges)
}

// Add records a single observation. Values outside the histogram's range are
// ignored and reported by the returned bool: true if the value landed in a
// bin, false if it fell outside. NaN never lands anywhere.
func (h *Histogram) Add(v float64) bool {
	if math.IsNaN(v) {
		return false
	}
	idx := h.indexOf(v)
	if idx < 0 {
		return false
	}
	h.bins[idx].Count++
	h.total++
	return true
}

// AddAll records each observation in order and returns the number that fell
// outside the range.
func (h *Histogram) AddAll(values []float64) int {
	missed := 0
	for _, v := range values {
		if !h.Add(v) {
			missed++
		}
	}
	return missed
}

// indexOf finds the bin for v using binary search. Returns -1 when v is out
// of range.
func (h *Histogram) indexOf(v float64) int {
	// sort.SearchFloat64s would need a slice; a direct binary search over the
	// bin edges avoids allocating one per call.
	lo, hi := 0, len(h.bins)
	for lo < hi {
		mid := (lo + hi) / 2
		if v < h.bins[mid].Low {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	// lo is the number of bins whose Low <= v. That bin index is lo-1.
	idx := lo - 1
	if idx < 0 {
		return -1
	}
	if idx >= len(h.bins) {
		return -1
	}
	b := h.bins[idx]
	if v < b.Low {
		return -1
	}
	if v < b.High {
		return idx
	}
	// v == b.High. Only the final bin is closed on the right.
	if idx == len(h.bins)-1 && v == b.High {
		return idx
	}
	return -1
}

// Count returns the number of observations recorded in bin i. It panics if i
// is out of range, matching the behaviour of indexing a slice.
func (h *Histogram) Count(i int) int {
	return h.bins[i].Count
}

// Total returns the number of observations that landed in some bin.
func (h *Histogram) Total() int {
	return h.total
}

// Len returns the number of bins.
func (h *Histogram) Len() int {
	return len(h.bins)
}

// Bins returns a copy of the bins so callers cannot mutate internal state.
func (h *Histogram) Bins() []Bin {
	out := make([]Bin, len(h.bins))
	copy(out, h.bins)
	return out
}

// Percent returns the share of observations in bin i as a fraction in [0,1].
// When the histogram is empty it returns 0 rather than NaN, because a caller
// summing percentages should not have to special-case the empty histogram.
func (h *Histogram) Percent(i int) float64 {
	if h.total == 0 {
		return 0
	}
	return float64(h.bins[i].Count) / float64(h.total)
}

// Percentages returns the fraction of observations in each bin in bin order.
func (h *Histogram) Percentages() []float64 {
	out := make([]float64, len(h.bins))
	for i := range h.bins {
		out[i] = h.Percent(i)
	}
	return out
}

// Reset clears every bin and the total, keeping the bin edges.
func (h *Histogram) Reset() {
	for i := range h.bins {
		h.bins[i].Count = 0
	}
	h.total = 0
}

// Merge adds the counts of other into h. Both histograms must have identical
// bin edges; otherwise an error is returned and h is left unchanged.
func (h *Histogram) Merge(other *Histogram) error {
	if other == nil {
		return errors.New("simple_histogram: cannot merge a nil histogram")
	}
	if len(h.bins) != len(other.bins) {
		return fmt.Errorf("simple_histogram: bin count mismatch: %d vs %d", len(h.bins), len(other.bins))
	}
	for i := range h.bins {
		if h.bins[i].Low != other.bins[i].Low || h.bins[i].High != other.bins[i].High {
			return fmt.Errorf("simple_histogram: bin %d edges differ", i)
		}
	}
	for i := range h.bins {
		h.bins[i].Count += other.bins[i].Count
	}
	h.total += other.total
	return nil
}

// Edges returns the bin edges in ascending order. The result has Len()+1
// entries. The slice is freshly allocated on each call.
func (h *Histogram) Edges() []float64 {
	out := make([]float64, len(h.bins)+1)
	for i := range h.bins {
		out[i] = h.bins[i].Low
	}
	out[len(h.bins)] = h.bins[len(h.bins)-1].High
	return out
}

// SortedValues is a small helper used by tests and callers who want to feed
// values in any order and have them processed deterministically. It returns a
// sorted copy.
func SortedValues(values []float64) []float64 {
	out := make([]float64, len(values))
	copy(out, values)
	sort.Float64s(out)
	return out
}
