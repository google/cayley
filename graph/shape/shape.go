package shape

import (
	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/iterator"
	"github.com/cayleygraph/cayley/quad"
	"regexp"
)

type Shape interface {
	BuildIterator(qs graph.QuadStore) graph.Iterator
	Optimize(qs graph.QuadStore) (Shape, bool)
	//Size(qs graph.QuadStore) (int64, bool)
}

func IsNull(s Shape) bool {
	_, ok := s.(Null)
	return s == nil || ok
}

func BuildIterator(qs graph.QuadStore, s Shape) graph.Iterator {
	if s != nil {
		s, _ = s.Optimize(qs)
	}
	if IsNull(s) {
		return iterator.NewNull()
	}
	return s.BuildIterator(qs)
}

type Null struct{}

func (Null) BuildIterator(qs graph.QuadStore) graph.Iterator {
	return iterator.NewNull()
}
func (Null) Optimize(qs graph.QuadStore) (Shape, bool) {
	return nil, true
}

type Except struct {
	Nodes Shape
	All   Shape
}

func (s Except) BuildIterator(qs graph.QuadStore) graph.Iterator {
	var all graph.Iterator
	if s.All != nil {
		all = s.All.BuildIterator(qs)
	} else {
		all = qs.NodesAllIterator()
	}
	if IsNull(s.Nodes) {
		return all
	}
	return iterator.NewNot(s.Nodes.BuildIterator(qs), all)
}
func (s Except) Optimize(qs graph.QuadStore) (Shape, bool) {
	nd, opt := s.Nodes.Optimize(qs)
	if opt {
		s.Nodes = nd
	}
	if s.All != nil {
		nd, opta := s.All.Optimize(qs)
		if opta {
			s.All = nd
		}
		opt = opt || opta
	}
	if IsNull(s.Nodes) {
		return AllNodes{}, true
	} else if _, ok := s.Nodes.(AllNodes); ok {
		return nil, true
	}
	return s, opt
}

type AllNodes struct{}

func (s AllNodes) BuildIterator(qs graph.QuadStore) graph.Iterator {
	return qs.NodesAllIterator()
}
func (s AllNodes) Optimize(qs graph.QuadStore) (Shape, bool) {
	return s, false
}

type ValueFilter struct {
	Op  iterator.Operator
	Val quad.Value
}

type Filter struct {
	From    Shape
	Filters []ValueFilter
}

func (s Filter) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if IsNull(s.From) {
		return iterator.NewNull()
	}
	it := s.From.BuildIterator(qs)
	for _, f := range s.Filters {
		it = iterator.NewComparison(it, f.Op, f.Val, qs)
	}
	return it
}
func (s Filter) Optimize(qs graph.QuadStore) (Shape, bool) {
	if IsNull(s.From) {
		return nil, true
	}
	var opt bool
	s.From, opt = s.From.Optimize(qs)
	if IsNull(s.From) {
		return nil, true
	} else if len(s.Filters) == 0 {
		return s.From, true
	}
	return s, opt
}

type Regexp struct {
	From Shape
	Re   *regexp.Regexp
	Refs bool
}

func (s Regexp) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if IsNull(s.From) {
		return iterator.NewNull()
	}
	it := s.From.BuildIterator(qs)
	if s.Re == nil {
		return it
	}
	rit := iterator.NewRegex(it, s.Re, qs)
	rit.AllowRefs(s.Refs)
	return rit
}
func (s Regexp) Optimize(qs graph.QuadStore) (Shape, bool) {
	if IsNull(s.From) {
		return nil, true
	}
	var opt bool
	s.From, opt = s.From.Optimize(qs)
	if IsNull(s.From) {
		return nil, true
	} else if s.Re == nil {
		return s.From, true
	}
	return s, opt
}

type Count struct {
	Values Shape
}

var _ graph.PreFetchedValue = fetchedValue{}

type fetchedValue struct {
	Val quad.Value
}

func (v fetchedValue) IsNode() bool       { return true }
func (v fetchedValue) NameOf() quad.Value { return v.Val }

func (s Count) BuildIterator(qs graph.QuadStore) graph.Iterator {
	var it graph.Iterator
	if IsNull(s.Values) {
		it = iterator.NewNull()
	} else {
		it = s.Values.BuildIterator(qs)
	}
	return iterator.NewCount(it, qs)
}
func (s Count) Optimize(qs graph.QuadStore) (Shape, bool) {
	if IsNull(s.Values) {
		return Fixed{fetchedValue{quad.Int(0)}}, true
	}
	var opt bool
	s.Values, opt = s.Values.Optimize(qs)
	if IsNull(s.Values) {
		return Fixed{fetchedValue{quad.Int(0)}}, true
	}
	// TODO: ask QS to estimate size - if it exact, then we can use it
	return s, opt
}

type QuadFilter struct {
	Dir    quad.Direction
	Values Shape
}

// not exposed to force to use Quads and group filters
func (s QuadFilter) buildIterator(qs graph.QuadStore) graph.Iterator {
	if s.Values == nil {
		return iterator.NewNull()
	} else if v, ok := One(s.Values); ok {
		return qs.QuadIterator(s.Dir, v)
	}
	if s.Dir == quad.Any {
		panic("direction is not set")
	}
	sub := s.Values.BuildIterator(qs)
	return iterator.NewLinksTo(qs, sub, s.Dir)
}

type Quads []QuadFilter

func (s Quads) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if len(s) == 0 {
		return qs.QuadsAllIterator()
	}
	its := make([]graph.Iterator, 0, len(s))
	for _, f := range s {
		its = append(its, f.buildIterator(qs))
	}
	if len(its) == 1 {
		return its[0]
	}
	return iterator.NewAnd(qs, its...)
}
func (s Quads) Optimize(qs graph.QuadStore) (Shape, bool) {
	var nq Quads
	// TODO: multiple constraints on the same dir -> merge as Intersect on Values of this dir
	for i, f := range s {
		if f.Values == nil {
			continue
		}
		v, ok := f.Values.Optimize(qs)
		if !ok {
			continue
		} else if v == nil {
			return nil, true
		}
		if nq == nil {
			nq = make(Quads, len(s))
			copy(nq, s)
		}
		nq[i].Values = v
	}
	opt := nq != nil
	if opt {
		s = nq
	}
	return s, opt
}

// aka HasA
type QuadDirection struct {
	Dir   quad.Direction
	Quads Shape
}

func (s QuadDirection) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if IsNull(s.Quads) {
		return iterator.NewNull()
	}
	sub := s.Quads.BuildIterator(qs)
	if s.Dir == quad.Any {
		panic("direction is not set")
	}
	return iterator.NewHasA(qs, sub, s.Dir)
}
func (s QuadDirection) Optimize(qs graph.QuadStore) (Shape, bool) {
	if IsNull(s.Quads) {
		return nil, true
	}
	f, opt := s.Quads.Optimize(qs)
	if opt {
		s.Quads = f
	}
	q, ok := s.Quads.(Quads)
	if !ok {
		return s, opt
	}
	if len(q) == 1 && q[0].Dir == s.Dir {
		return q[0].Values, true
	}
	var (
		filt map[quad.Direction]graph.Value
		save map[quad.Direction][]string
		n    int
	)
	for _, f := range q {
		if v, ok := One(f.Values); ok {
			if filt == nil {
				filt = make(map[quad.Direction]graph.Value)
			}
			if _, ok := filt[f.Dir]; ok {
				return s, opt
			}
			filt[f.Dir] = v
			n++
		} else if sv, ok := f.Values.(Save); ok {
			if _, ok = sv.From.(AllNodes); ok {
				if save == nil {
					save = make(map[quad.Direction][]string)
				}
				save[f.Dir] = append(save[f.Dir], sv.Tags...)
				n++
			}
		}
	}
	if n == len(q) {
		return QuadsAct{
			Result: s.Dir,
			Filter: filt,
			Save:   save,
		}, true
	}
	// TODO
	return s, opt
}

type QuadsAct struct {
	Result quad.Direction
	Save   map[quad.Direction][]string
	Filter map[quad.Direction]graph.Value
}

func (s QuadsAct) BuildIterator(qs graph.QuadStore) graph.Iterator {
	q := make(Quads, 0, len(s.Save)+len(s.Filter))
	for dir, val := range s.Filter {
		q = append(q, QuadFilter{Dir: dir, Values: Fixed{val}})
	}
	for dir, tags := range s.Save {
		q = append(q, QuadFilter{Dir: dir, Values: Save{From: AllNodes{}, Tags: tags}})
	}
	h := QuadDirection{Dir: s.Result, Quads: q}
	return h.BuildIterator(qs)
}
func (s QuadsAct) Optimize(qs graph.QuadStore) (Shape, bool) {
	return s, false
}

func One(s Shape) (graph.Value, bool) {
	switch s := s.(type) {
	case Fixed:
		if len(s) == 1 {
			return s[0], true
		}
	}
	return nil, false
}

type Fixed []graph.Value

func (s Fixed) BuildIterator(qs graph.QuadStore) graph.Iterator {
	it := qs.FixedIterator()
	for _, v := range s {
		if _, ok := v.(quad.Value); ok {
			panic("quad value in fixed iterator")
		}
		it.Add(v)
	}
	return it
}
func (s Fixed) Optimize(qs graph.QuadStore) (Shape, bool) {
	if len(s) == 0 {
		return nil, true
	}
	return s, false
}

type Lookup []quad.Value

func (s Lookup) resolve(qs graph.QuadStore) Shape {
	// TODO: check if QS supports batch lookup
	vals := make([]graph.Value, 0, len(s))
	for _, v := range s {
		if gv := qs.ValueOf(v); gv != nil {
			vals = append(vals, gv)
		}
	}
	if len(vals) == 0 {
		return nil
	}
	return Fixed(vals)
}
func (s Lookup) BuildIterator(qs graph.QuadStore) graph.Iterator {
	f := s.resolve(qs)
	if IsNull(f) {
		return iterator.NewNull()
	}
	return f.BuildIterator(qs)
}
func (s Lookup) Optimize(qs graph.QuadStore) (Shape, bool) {
	return s.resolve(qs), true
}

type Intersect []Shape

func (s Intersect) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if len(s) == 0 {
		return iterator.NewNull()
	}
	sub := make([]graph.Iterator, 0, len(s))
	for _, c := range s {
		sub = append(sub, c.BuildIterator(qs))
	}
	if len(sub) == 1 {
		return sub[0]
	}
	return iterator.NewAnd(qs, sub...)
}
func (s Intersect) Optimize(qs graph.QuadStore) (Shape, bool) {
	var opt bool
	realloc := func() {
		if !opt {
			arr := make(Intersect, len(s))
			copy(arr, s)
			s = arr
		}
	}
	// optimize subiterators, return empty set if Null is found
	for i := 0; i < len(s); i++ {
		c := s[i]
		if IsNull(c) {
			return nil, true
		}
		v, ok := c.Optimize(qs)
		if !ok {
			continue
		}
		realloc()
		opt = true
		if IsNull(v) {
			return nil, true
		}
		s[i] = v
	}
	var (
		quads Quads
		fixed Fixed
	)
	remove := func(i *int, o bool) {
		realloc()
		if o {
			opt = true
		}
		v := *i
		s = append(s[:v], s[v+1:]...)
		v--
		*i = v
	}
	// second pass - remove AllNodes, merge Quads, merge Fixed, merge Intersects
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c := c.(type) {
		case AllNodes:
			remove(&i, true)
		case Quads:
			remove(&i, false)
			if quads == nil {
				quads = c[:len(c):len(c)]
			} else {
				opt = true
				quads = append(quads, c...)
			}
		case Fixed:
			remove(&i, true)
			fixed = append(fixed, c...)
		case Intersect:
			remove(&i, true)
			s = append(s, c...)
		}
	}
	if quads != nil {
		nq, qopt := quads.Optimize(qs)
		if IsNull(nq) {
			return nil, true
		}
		opt = opt || qopt
		s = append(s, nq)
	}
	if len(fixed) != 0 {
		s = append(s, nil)
		copy(s[1:], s)
		s[0] = fixed
	}
	if len(s) == 0 {
		return nil, true
	} else if len(s) == 1 {
		return s[0], true
	}
	// TODO: optimize order, intersect Fixed
	return s, opt
}

type Union []Shape

func (s Union) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if len(s) == 0 {
		return iterator.NewNull()
	}
	sub := make([]graph.Iterator, 0, len(s))
	for _, c := range s {
		sub = append(sub, c.BuildIterator(qs))
	}
	if len(sub) == 1 {
		return sub[0]
	}
	return iterator.NewOr(sub...)
}
func (s Union) Optimize(qs graph.QuadStore) (Shape, bool) {
	var opt bool
	realloc := func() {
		if !opt {
			arr := make(Union, len(s))
			copy(arr, s)
			s = arr
		}
	}
	// optimize subiterators
	for i := 0; i < len(s); i++ {
		c := s[i]
		v, ok := c.Optimize(qs)
		if !ok {
			continue
		}
		realloc()
		opt = true
		s[i] = v
	}
	// second pass - remove Null
	for i := 0; i < len(s); i++ {
		c := s[i]
		if IsNull(c) {
			realloc()
			opt = true
			s = append(s[:i], s[i+1:]...)
		}
	}
	if len(s) == 0 {
		return nil, true
	} else if len(s) == 1 {
		return s[0], true
	}
	// TODO: join Fixed
	return s, opt
}

type Page struct {
	From  Shape
	Skip  int64
	Limit int64
}

func (s Page) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if IsNull(s.From) {
		return iterator.NewNull()
	}
	it := s.From.BuildIterator(qs)
	if s.Skip > 0 {
		it = iterator.NewSkip(it, s.Skip)
	}
	if s.Limit > 0 {
		it = iterator.NewLimit(it, s.Limit)
	}
	return it
}
func (s Page) Optimize(qs graph.QuadStore) (Shape, bool) {
	if IsNull(s.From) {
		return nil, true
	}
	f, opt := s.From.Optimize(qs)
	if opt {
		s.From = f
	}
	if s.Skip <= 0 && s.Limit <= 0 {
		return s.From, true
	}
	// TODO: check size
	return s, false
}

type Unique struct {
	From Shape
}

func (s Unique) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if IsNull(s.From) {
		return iterator.NewNull()
	}
	it := s.From.BuildIterator(qs)
	return iterator.NewUnique(it)
}
func (s Unique) Optimize(qs graph.QuadStore) (Shape, bool) {
	if IsNull(s.From) {
		return nil, true
	}
	f, opt := s.From.Optimize(qs)
	s.From = f
	if IsNull(s.From) {
		return nil, true
	}
	return s, opt
}

type Save struct {
	From Shape
	Tags []string
}

func (s Save) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if IsNull(s.From) {
		return iterator.NewNull()
	}
	it := s.From.BuildIterator(qs)
	if len(s.Tags) != 0 {
		it.Tagger().Add(s.Tags...)
	}
	return it
}
func (s Save) Optimize(qs graph.QuadStore) (Shape, bool) {
	if IsNull(s.From) {
		return nil, true
	}
	f, opt := s.From.Optimize(qs)
	if opt {
		s.From = f
	}
	if len(s.Tags) == 0 {
		return s.From, true
	}
	return s, opt
}

type Optional struct {
	From Shape
}

func (s Optional) BuildIterator(qs graph.QuadStore) graph.Iterator {
	if IsNull(s.From) {
		return iterator.NewNull()
	}
	return iterator.NewOptional(s.From.BuildIterator(qs))
}
func (s Optional) Optimize(qs graph.QuadStore) (Shape, bool) {
	if IsNull(s.From) {
		return nil, true
	}
	f, opt := s.From.Optimize(qs)
	s.From = f
	if IsNull(s.From) {
		return nil, true
	}
	return s, opt
}
