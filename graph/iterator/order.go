package iterator

import (
	"context"
	"sort"
	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/quad"
)

var _ graph.Iterator = &Order{}

type values []quad.Value

func (a values) Len() int           { return len(a) }
func (a values) Less(i, j int) bool { return a[i].String() < a[j].String() }
func (a values) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// Order iterator removes duplicate values from it's subiterator.
type Order struct {
	uid      uint64
	qs		 graph.QuadStore
	subIt    graph.Iterator
	result   graph.Value
	resultIndex int
	runstats graph.IteratorStats
	err      error
	ordered   values
}

func getOrderedValues(qs graph.QuadStore, subIt graph.Iterator) values {
	var vals = make(values, 0)
	var ctx = context.TODO()
	
	for subIt.Next(ctx) {
		var result = subIt.Result()
		var value = qs.NameOf(result)
		vals = append(vals, value)
	}

	sort.Sort(vals)

	subIt.Reset()

	return vals
}

func NewOrder(qs graph.QuadStore, subIt graph.Iterator) *Order {
	return &Order{
		qs: 	 qs,
		uid:     NextUID(),
		subIt: 	 subIt,
		ordered: getOrderedValues(qs, subIt),
	}
}

func (it *Order) UID() uint64 {
	return it.uid
}

// Reset resets the internal iterators and the iterator itself.
func (it *Order) Reset() {
	it.result = nil
	it.resultIndex = 0
	it.subIt.Reset()
}

func (it *Order) TagResults(dst map[string]graph.Value) {
	if it.subIt != nil {
		it.subIt.TagResults(dst)
	}
}

// SubIterators returns a slice of the sub iterators. The first iterator is the
// primary iterator, for which the complement is generated.
func (it *Order) SubIterators() []graph.Iterator {
	return []graph.Iterator{it.subIt}
}

// Next advances the subiterator, continuing until it returns a value which it
// has not previously seen.
func (it *Order) Next(ctx context.Context) bool {
	it.runstats.Next += 1
	if it.resultIndex < len(it.ordered) - 1 {
		it.resultIndex += 1
		it.result = it.qs.ValueOf(it.ordered[it.resultIndex])
		return true
	}
	return false
}

func (it *Order) Err() error {
	return it.err
}

func (it *Order) Result() graph.Value {
	return it.result
}

// Contains checks whether the passed value is part of the primary iterator,
// which is irrelevant for Orderness.
func (it *Order) Contains(ctx context.Context, val graph.Value) bool {
	it.runstats.Contains += 1
	return it.subIt.Contains(ctx, val)
}

// NextPath for Order always returns false. If we were to return multiple
// paths, we'd no longer be a Order result, so we have to choose only the first
// path that got us here. Order is serious on this point.
func (it *Order) NextPath(ctx context.Context) bool {
	return false
}

// Close closes the primary iterators.
func (it *Order) Close() error {
	it.ordered = nil
	return it.subIt.Close()
}

func (it *Order) Optimize() (graph.Iterator, bool) {
	newIt, optimized := it.subIt.Optimize()
	if optimized {
		it.subIt = newIt
	}
	return it, false
}

const OrdernessFactor = 2

func (it *Order) Stats() graph.IteratorStats {
	subStats := it.subIt.Stats()
	return graph.IteratorStats{
		NextCost:     1,
		ContainsCost: subStats.ContainsCost,
		Size:         int64(len(it.ordered)),
		ExactSize:    true,
		Next:         it.runstats.Next,
		Contains:     it.runstats.Contains,
		ContainsNext: it.runstats.ContainsNext,
	}
}

func (it *Order) Size() (int64, bool) {
	st := it.Stats()
	return st.Size, st.ExactSize
}

func (it *Order) String() string {
	return "Order"
}
