# simple_histogram

Buckets numeric values into configurable bins and reports the count and the
percentage of observations in each bin.

```go
package main

import (
	"fmt"

	"simple_histogram"
)

func main() {
	h, err := simple_histogram.NewUniform(0, 10, 5)
	if err != nil {
		panic(err)
	}
	missed := h.AddAll([]float64{0.5, 1.5, 1.7, 9.9, 42, -3})
	fmt.Println("outside range:", missed) // 2

	for i, b := range h.Bins() {
		fmt.Printf("[%g, %g) count=%d percent=%.2f\n",
			b.Low, b.High, h.Count(i), h.Percent(i))
	}
}
```

## Why this exists

Most histogram code I have written by hand gets the top edge wrong. If bins are
`[lo, hi)` everywhere, the maximum value in the range falls off the end and you
never notice until a chart looks short. I chose to make every bin half-open on
the right except the last, which is closed on the right. That means a value
exactly equal to the highest edge lands in the final bin, and values above the
highest edge are reported as missed by `Add` returning false and by `AddAll`
returning a count. The trade-off is that the last bin is half a step wider than
the others when the edges are uniform, which is a small asymmetry in exchange
for not silently dropping the endpoint.

## The edge you will hit

`NewUniform(min, max, n)` pins the last edge exactly to `max` after computing
equal widths. Without that, floating-point drift on the multiplication can
leave the very top of the range uncovered by a sliver, and a value of exactly
`max` would be reported as outside. If you build bins yourself with `New`, the
same rule applies: the last edge is inclusive, all the others are not.

`Percent` returns `0` for an empty histogram rather than `NaN`, so summing
percentages never produces `NaN` just because nothing has been recorded yet.

## API

- `New(edges []float64) (*Histogram, error)` — build from strictly increasing
  edges; needs at least two.
- `NewUniform(min, max float64, n int) (*Histogram, error)` — `n` equal-width
  bins over `[min, max]`.
- `(*Histogram) Add(v float64) bool` — record one value; false if it fell
  outside every bin or was `NaN`.
- `(*Histogram) AddAll(values []float64) int` — record many; returns how many
  were missed.
- `(*Histogram) Count(i int) int` — observations in bin `i`.
- `(*Histogram) Total() int` — observations that landed in some bin.
- `(*Histogram) Len() int` — number of bins.
- `(*Histogram) Bins() []Bin` — a copy of the bins (`Bin` has `Low`, `High`,
  `Count`).
- `(*Histogram) Percent(i int) float64` — share in bin `i`, in `[0,1]`.
- `(*Histogram) Percentages() []float64` — shares for every bin in order.
- `(*Histogram) Reset()` — zero the counts, keep the edges.
- `(*Histogram) Merge(other *Histogram) error` — add another histogram's
  counts; edges must match exactly.
- `(*Histogram) Edges() []float64` — the bin edges, `Len()+1` of them.
- `SortedValues(values []float64) []float64` — a sorted copy, handy for feeding
  values deterministically.
