package simple_histogram

import (
	"math"
	"testing"
)

const eps = 1e-9

func almostEq(a, b float64) bool {
	return math.Abs(a-b) < eps
}

func mustNew(t *testing.T, edges []float64) *Histogram {
	t.Helper()
	h, err := New(edges)
	if err != nil {
		t.Fatalf("New(%v) returned error: %v", edges, err)
	}
	return h
}

func TestNewRejectsBadEdges(t *testing.T) {
	cases := []struct {
		name  string
		edges []float64
	}{
		{"empty", nil},
		{"single", []float64{1}},
		{"unsorted", []float64{2, 1}},
		{"duplicate", []float64{1, 1, 2}},
		{"nan", []float64{0, math.NaN(), 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.edges); err == nil {
				t.Fatalf("New(%v) expected error, got nil", tc.edges)
			}
		})
	}
}

func TestNewUniformRejectsBadArgs(t *testing.T) {
	if _, err := NewUniform(0, 1, 0); err == nil {
		t.Fatal("NewUniform with n=0 should fail")
	}
	if _, err := NewUniform(1, 1, 4); err == nil {
		t.Fatal("NewUniform with max==min should fail")
	}
	if _, err := NewUniform(1, 0, 4); err == nil {
		t.Fatal("NewUniform with max<min should fail")
	}
	if _, err := NewUniform(math.NaN(), 1, 4); err == nil {
		t.Fatal("NewUniform with NaN min should fail")
	}
}

func TestNewUniformEdges(t *testing.T) {
	h, err := NewUniform(0, 10, 5)
	if err != nil {
		t.Fatalf("NewUniform: %v", err)
	}
	got := h.Edges()
	want := []float64{0, 2, 4, 6, 8, 10}
	if len(got) != len(want) {
		t.Fatalf("edge count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if !almostEq(got[i], want[i]) {
			t.Errorf("edge %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestAddHalfOpenExceptLast(t *testing.T) {
	h := mustNew(t, []float64{0, 1, 2, 3})
	// 0 -> bin 0, 1 -> bin 1, 2 -> bin 2, 3 -> bin 2 (last bin closed).
	if !h.Add(0) {
		t.Fatal("0 should land in a bin")
	}
	if !h.Add(1) {
		t.Fatal("1 should land in a bin")
	}
	if !h.Add(2) {
		t.Fatal("2 should land in a bin")
	}
	if !h.Add(3) {
		t.Fatal("3 should land in the last bin")
	}
	want := []int{1, 1, 2}
	for i, w := range want {
		if got := h.Count(i); got != w {
			t.Errorf("Count(%d) = %d, want %d", i, got, w)
		}
	}
	if h.Total() != 4 {
		t.Errorf("Total = %d, want 4", h.Total())
	}
}

func TestAddOutOfRange(t *testing.T) {
	h := mustNew(t, []float64{0, 1, 2})
	if h.Add(-0.5) {
		t.Fatal("below range should not land")
	}
	if h.Add(2.5) {
		t.Fatal("above range should not land")
	}
	if h.Add(math.NaN()) {
		t.Fatal("NaN should not land")
	}
	if h.Total() != 0 {
		t.Errorf("Total = %d, want 0", h.Total())
	}
}

func TestAddAllCountsMissed(t *testing.T) {
	h := mustNew(t, []float64{0, 1, 2})
	missed := h.AddAll([]float64{-1, 0, 0.5, 1, 1.5, 5})
	if missed != 2 {
		t.Errorf("missed = %d, want 2", missed)
	}
	if h.Total() != 4 {
		t.Errorf("Total = %d, want 4", h.Total())
	}
}

func TestPercentages(t *testing.T) {
	h := mustNew(t, []float64{0, 1, 2})
	h.AddAll([]float64{0.1, 0.2, 0.3, 1.1})
	p := h.Percentages()
	if len(p) != 2 {
		t.Fatalf("len(Percentages) = %d, want 2", len(p))
	}
	if !almostEq(p[0], 0.75) {
		t.Errorf("p[0] = %v, want 0.75", p[0])
	}
	if !almostEq(p[1], 0.25) {
		t.Errorf("p[1] = %v, want 0.25", p[1])
	}
}

func TestPercentEmptyIsZero(t *testing.T) {
	h := mustNew(t, []float64{0, 1, 2})
	if got := h.Percent(0); got != 0 {
		t.Errorf("Percent on empty = %v, want 0", got)
	}
}

func TestResetKeepsEdges(t *testing.T) {
	h := mustNew(t, []float64{0, 1, 2})
	h.AddAll([]float64{0.1, 1.1})
	h.Reset()
	if h.Total() != 0 {
		t.Errorf("Total after Reset = %d, want 0", h.Total())
	}
	if h.Len() != 2 {
		t.Errorf("Len after Reset = %d, want 2", h.Len())
	}
	if h.Count(0) != 0 || h.Count(1) != 0 {
		t.Error("counts not cleared by Reset")
	}
}

func TestMerge(t *testing.T) {
	a := mustNew(t, []float64{0, 1, 2})
	b := mustNew(t, []float64{0, 1, 2})
	a.AddAll([]float64{0.1, 0.2})
	b.AddAll([]float64{0.3, 1.1})
	if err := a.Merge(b); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if a.Count(0) != 3 {
		t.Errorf("Count(0) = %d, want 3", a.Count(0))
	}
	if a.Count(1) != 1 {
		t.Errorf("Count(1) = %d, want 1", a.Count(1))
	}
	if a.Total() != 4 {
		t.Errorf("Total = %d, want 4", a.Total())
	}
}

func TestMergeRejectsMismatch(t *testing.T) {
	a := mustNew(t, []float64{0, 1, 2})
	b := mustNew(t, []float64{0, 1, 3})
	if err := a.Merge(b); err == nil {
		t.Fatal("Merge with different edges should fail")
	}
	if a.Total() != 0 {
		t.Errorf("Total changed after failed Merge = %d", a.Total())
	}
	if err := a.Merge(nil); err == nil {
		t.Fatal("Merge(nil) should fail")
	}
}

func TestBinsIsCopy(t *testing.T) {
	h := mustNew(t, []float64{0, 1, 2})
	h.Add(0.5)
	bins := h.Bins()
	bins[0].Count = 99
	if h.Count(0) != 1 {
		t.Errorf("Bins() returns internal state: Count(0) = %d, want 1", h.Count(0))
	}
}

func TestSortedValues(t *testing.T) {
	got := SortedValues([]float64{3, 1, 2})
	want := []float64{1, 2, 3}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SortedValues[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}
