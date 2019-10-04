package iterator

import (
	"context"
	"sort"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/quad"
)

var _ graph.Iterator = &Sort{}

type value struct {
	result result
	name   quad.Value
	str    string
}
type values []value

func (v values) Len() int { return len(v) }
func (v values) Less(i, j int) bool {
	return v[i].str < v[j].str
}
func (v values) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

// Sort iterator removes duplicate values from it's subiterator.
type Sort struct {
	uid      uint64
	namer    graph.Namer
	subIt    graph.Iterator
	result   graph.Value
	index    int
	runstats graph.IteratorStats
	err      error
	ordered  *values
}

func getSortedValues(namer graph.Namer, it graph.Iterator) (*values, error) {
	var v values
	var ctx = context.TODO()

	for it.Next(ctx) {
		var id = it.Result()
		var name = namer.NameOf(id)
		var str = name.String()
		var tags = make(map[string]graph.Value)
		it.TagResults(tags)
		result := result{id, tags}
		value := value{result, name, str}
		v = append(v, value)
		err := it.Err()
		if err != nil {
			return &v, err
		}
	}

	sort.Sort(v)

	it.Reset()

	return &v, nil
}

func NewSort(namer graph.Namer, subIt graph.Iterator) *Sort {
	return &Sort{
		namer:   namer,
		uid:     NextUID(),
		subIt:   subIt,
		ordered: nil,
	}
}

func (it *Sort) UID() uint64 {
	return it.uid
}

// Reset resets the internal iterators and the iterator itself.
func (it *Sort) Reset() {
	it.result = nil
	it.index = 0
	it.subIt.Reset()
}

func (it *Sort) TagResults(dst map[string]graph.Value) {
	var prevIndex = it.index - 1
	var ordered = *it.ordered
	for tag, value := range ordered[prevIndex].result.tags {
		dst[tag] = value
	}
}

// SubIterators returns a slice of the sub iterators.
func (it *Sort) SubIterators() []graph.Iterator {
	return []graph.Iterator{it.subIt}
}

// Next advances the subiterator, continuing until it returns a value which it
// has not previously seen.
func (it *Sort) Next(ctx context.Context) bool {
	it.runstats.Next++
	if it.ordered == nil {
		v, err := getSortedValues(it.namer, it.subIt)
		it.ordered = v
		it.err = err
	}
	if it.index < len(*it.ordered) {
		ordered := *it.ordered
		it.result = ordered[it.index].result.id
		it.index += 1
		return true
	}
	return false
}

func (it *Sort) Err() error {
	return it.err
}

func (it *Sort) Result() graph.Value {
	return it.result
}

// Contains checks whether the passed value is part of the primary iterator,
// which is irrelevant for Sortness.
func (it *Sort) Contains(ctx context.Context, val graph.Value) bool {
	it.runstats.Contains += 1
	return it.subIt.Contains(ctx, val)
}

// NextPath for Sort always returns false. If we were to return multiple
// paths, we'd no longer be a Sort result, so we have to choose only the first
// path that got us here. Sort is serious on this point.
func (it *Sort) NextPath(ctx context.Context) bool {
	return false
}

// Close closes the primary iterators.
func (it *Sort) Close() error {
	it.ordered = nil
	return it.subIt.Close()
}

func (it *Sort) Optimize() (graph.Iterator, bool) {
	newIt, optimized := it.subIt.Optimize()
	if optimized {
		it.subIt = newIt
	}
	return it, false
}

func (it *Sort) Stats() graph.IteratorStats {
	subStats := it.subIt.Stats()
	return graph.IteratorStats{
		NextCost:     subStats.NextCost,
		ContainsCost: subStats.ContainsCost,
		Size:         int64(len(*it.ordered)),
		ExactSize:    true,
		Next:         it.runstats.Next,
		Contains:     it.runstats.Contains,
		ContainsNext: it.runstats.ContainsNext,
	}
}

func (it *Sort) Size() (int64, bool) {
	st := it.Stats()
	return st.Size, st.ExactSize
}

func (it *Sort) String() string {
	return "Sort{" + it.subIt.String() + "}"
}
