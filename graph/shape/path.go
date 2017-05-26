package shape

import (
	"fmt"
	"regexp"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/iterator"
	"github.com/cayleygraph/cayley/quad"
	"golang.org/x/net/context"
)

func StartNodes(nodes ...graph.Value) Path {
	var s Shape = AllNodes{}
	if len(nodes) != 0 {
		s = Fixed(nodes)
	}
	return StartFrom(s)
}

func Start(vals ...quad.Value) Path {
	var s Shape = AllNodes{}
	if len(vals) != 0 {
		s = Lookup(vals)
	}
	return StartFrom(s)
}

func StartFrom(s Shape) Path {
	return Path{root: s}
}

type Path struct {
	root   Shape
	labels Shape
}

func (p Path) Zero() bool {
	return p.root == nil
}

func (p Path) Shape() Shape {
	if p.root == nil {
		return AllNodes{}
	}
	return p.root
}

func intersect(s1, s2 Shape) Shape {
	switch s1 := s1.(type) {
	case AllNodes:
		return s2
	case Intersect:
		if s2, ok := s2.(Intersect); ok {
			return append(s1, s2...)
		}
		return append(s1, s2)
	}
	return Intersect{s1, s2}
}

func union(s1, s2 Shape) Union {
	if s1, ok := s1.(Union); ok {
		if s2, ok := s2.(Union); ok {
			return append(s1, s2...)
		}
		return append(s1, s2)
	}
	return Union{s1, s2}
}

func (p Path) Is(vals ...graph.Value) Path {
	if len(vals) == 0 {
		return p
	}
	p.root = intersect(p.root, Fixed(vals))
	return p
}

func (p Path) IsValue(vals ...quad.Value) Path {
	if len(vals) == 0 {
		return p
	}
	p.root = intersect(p.root, Lookup(vals))
	return p
}

func (p Path) Filter(filters ...ValueFilter) Path {
	if fl, ok := p.root.(Filter); ok {
		fl.Filters = append(fl.Filters, filters...)
		p.root = fl
	} else {
		p.root = Filter{From: p.root, Filters: filters}
	}
	return p
}

func (p Path) Regexp(pattern *regexp.Regexp) Path {
	return p.Filter(Regexp{Re: pattern, Refs: false})
}

func (p Path) RegexpWithRefs(pattern *regexp.Regexp) Path {
	return p.Filter(Regexp{Re: pattern, Refs: true})
}

func (p Path) Compare(op iterator.Operator, node quad.Value) Path {
	return p.Filter(Comparison{Op: op, Val: node})
}

func (p Path) Tag(tags ...string) Path {
	if len(tags) == 0 {
		return p
	}
	p.root = Save{
		From: p.root,
		Tags: tags,
	}
	return p
}

func (p Path) LabelContext(via ...quad.Value) Path {
	if len(via) == 0 {
		p.labels = nil
	} else {
		p.labels = Lookup(via)
	}
	return p
}

func asNodes(arr []interface{}) Shape {
	nodes := make(Fixed, 0, len(arr))
	vals := make(Lookup, 0, len(arr))
	for _, v := range arr {
		if qv, ok := quad.AsValue(v); ok {
			vals = append(vals, qv)
		} else if gv, ok := v.(graph.Value); ok { // FIXME: this should be the first cast
			if gv != nil {
				nodes = append(nodes, gv)
			}
		} else {
			panic(fmt.Errorf("Invalid type passed to buildViaPath: %v (%T)", v, v))
		}
	}
	if len(vals) != 0 && len(nodes) != 0 {
		return Union{nodes, vals}
	} else if len(vals) != 0 {
		return vals
	}
	return nodes
}

func buildVia(via []interface{}) Shape {
	if len(via) == 0 {
		return AllNodes{}
	} else if len(via) == 1 {
		switch v := via[0].(type) {
		case nil:
			return AllNodes{}
		case Shape:
			return v // TODO: clone
		case Path:
			return v.root // TODO: clone
		case quad.Value:
			return Lookup{v}
		case graph.Value:
			if v != nil {
				return Fixed{v}
			}
			return Null{}
		}
	}
	return asNodes(via)
}

func buildOut(from, via, labels Shape, tags []string, in bool) Shape {
	start, goal := quad.Subject, quad.Object
	if in {
		start, goal = goal, start
	}
	if len(tags) != 0 {
		via = Save{From: via, Tags: tags}
	}

	quads := make(Quads, 0, 3)
	if _, ok := from.(AllNodes); !ok {
		quads = append(quads, QuadFilter{
			Dir: start, Values: from,
		})
	}
	if _, ok := via.(AllNodes); !ok {
		quads = append(quads, QuadFilter{
			Dir: quad.Predicate, Values: via,
		})
	}
	if labels != nil {
		if _, ok := labels.(AllNodes); !ok {
			quads = append(quads, QuadFilter{
				Dir: quad.Label, Values: labels,
			})
		}
	}
	return NodesFrom{Quads: quads, Dir: goal}
}

func Out(from, via, labels Shape, tags ...string) Shape {
	return buildOut(from, via, labels, tags, false)
}

func In(from, via, labels Shape, tags ...string) Shape {
	return buildOut(from, via, labels, tags, true)
}

func (p Path) Out(via ...interface{}) Path {
	pred := buildVia(via)
	p.root = buildOut(p.root, pred, p.labels, nil, false)
	return p
}

func (p Path) In(via ...interface{}) Path {
	pred := buildVia(via)
	p.root = buildOut(p.root, pred, p.labels, nil, true)
	return p
}

// InWithTags, OutWithTags, Both, BothWithTags

func Predicates(from Shape, in bool) Shape {
	dir := quad.Subject
	if in {
		dir = quad.Object
	}
	return Unique{NodesFrom{
		Quads: Quads{
			{Dir: dir, Values: from},
		},
		Dir: quad.Predicate,
	}}
}

func (p Path) Predicates(in bool) Path {
	p.root = Predicates(p.root, in)
	return p
}

func (p Path) Intersect(shape ...Shape) Path {
	for _, s := range shape {
		p.root = intersect(p.root, s)
	}
	return p
}

func (p Path) And(shape ...Shape) Path {
	return p.Intersect(shape...)
}

func (p Path) Union(shape ...Shape) Path {
	for _, s := range shape {
		p.root = union(p.root, s)
	}
	return p
}

func (p Path) Or(shape ...Shape) Path {
	return p.Union(shape...)
}
func (p Path) Except(s Shape) Path {
	p.root = Except{From: p.root, Exclude: s}
	return p
}
func (p Path) Unique() Path {
	p.root = Unique{p.root}
	return p
}

func SaveVia(from, via Shape, tag string, rev, opt bool) Shape {
	nodes := Save{
		From: AllNodes{},
		Tags: []string{tag},
	}
	start, goal := quad.Subject, quad.Object
	if rev {
		start, goal = goal, start
	}

	var save Shape = NodesFrom{
		Quads: Quads{
			{Dir: goal, Values: nodes},
			{Dir: quad.Predicate, Values: via},
		},
		Dir: start,
	}
	if opt {
		save = Optional{save}
	}
	return intersect(from, save)
}

func (p Path) Save(via interface{}, tag string) Path {
	return p.SaveOpt(via, tag, false, false)
}

func (p Path) SaveReverse(via interface{}, tag string) Path {
	return p.SaveOpt(via, tag, true, false)
}

func (p Path) SaveOpt(via interface{}, tag string, rev, opt bool) Path {
	pred := buildVia([]interface{}{via})
	p.root = SaveVia(p.root, pred, tag, rev, opt)
	return p
}

func Has(from, via, nodes Shape, rev bool) Shape {
	start, goal := quad.Subject, quad.Object
	if rev {
		start, goal = goal, start
	}

	quads := make(Quads, 0, 2)
	if _, ok := nodes.(AllNodes); !ok {
		quads = append(quads, QuadFilter{
			Dir: goal, Values: nodes,
		})
	}
	if _, ok := via.(AllNodes); !ok {
		quads = append(quads, QuadFilter{
			Dir: quad.Predicate, Values: via,
		})
	}
	if len(quads) == 0 {
		panic("empty has")
	}
	return intersect(from, NodesFrom{
		Quads: quads, Dir: start,
	})
}

func (p Path) Has(via interface{}, rev bool, nodes ...graph.Value) Path {
	pred := buildVia([]interface{}{via})
	var ends Shape = AllNodes{}
	if len(nodes) != 0 {
		ends = Fixed(nodes)
	}
	p.root = Has(p.root, pred, ends, rev)
	return p
}

func (p Path) HasValues(via interface{}, rev bool, vals ...quad.Value) Path {
	pred := buildVia([]interface{}{via})
	var ends Shape = AllNodes{}
	if len(vals) != 0 {
		ends = Lookup(vals)
	}
	p.root = Has(p.root, pred, ends, rev)
	return p
}

func (p Path) Page(skip, limit int64) Path {
	p.root = Page{From: p.root, Skip: skip, Limit: limit}
	return p
}

func (p Path) Limit(limit int64) Path {
	return p.Page(0, limit)
}

func Iterate(ctx context.Context, qs graph.QuadStore, s Shape) *graph.IterateChain {
	it := BuildIterator(qs, s)
	return graph.Iterate(ctx, it).On(qs)
}

func (p Path) Iterate(ctx context.Context, qs graph.QuadStore) *graph.IterateChain {
	return Iterate(ctx, qs, p.root)
}
