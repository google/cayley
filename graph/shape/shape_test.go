// Copyright 2014 The Cayley Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shape_test

import (
	"testing"

	"github.com/cayleygraph/cayley/graph"
	. "github.com/cayleygraph/cayley/graph/shape"
	"github.com/cayleygraph/cayley/graph/shape/shapetest"
	"github.com/cayleygraph/cayley/quad"
	"github.com/stretchr/testify/assert"
)

func TestPaths(t *testing.T) {
	shapetest.RunTestShapes(t, nil)
}

type intVal int
func (v intVal) Key() interface{} { return v }

var optimizeCases = []struct {
	from   Shape
	expect Shape
	opt    bool
	qs     lookupQuadStore
}{
	{
		from:   AllNodes{},
		opt:    false,
		expect: AllNodes{},
	},
	{ // intersect quads and lookup resolutions
		from: Intersect{
			Quads{
				{Dir: quad.Subject, Values: Lookup{quad.IRI("bob")}},
			},
			Quads{
				{Dir: quad.Object, Values: Lookup{quad.IRI("alice")}},
			},
		},
		opt: true,
		expect: Quads{
			{Dir: quad.Subject, Values: Fixed{intVal(1)}},
			{Dir: quad.Object, Values: Fixed{intVal(2)}},
		},
		qs: lookupQuadStore{
			quad.IRI("bob"):   intVal(1),
			quad.IRI("alice"): intVal(2),
		},
	},
	{ // intersect nodes, remove all, join intersects
		from: Intersect{
			AllNodes{},
			QuadDirection{Dir: quad.Subject, Quads: Quads{}},
			Intersect{
				Lookup{quad.IRI("alice")},
				Unique{QuadDirection{Dir: quad.Object, Quads: Quads{}}},
			},
		},
		opt: true,
		expect: Intersect{
			Fixed{intVal(1)},
			QuadsAct{Result: quad.Subject},
			Unique{QuadsAct{Result: quad.Object}},
		},
		qs: lookupQuadStore{
			quad.IRI("alice"): intVal(1),
		},
	},
	{ // push Save out of intersect
		from: Intersect{
			Save{
				Tags: []string{"id"},
				From: QuadDirection{Dir: quad.Subject, Quads: Quads{}},
			},
			Unique{QuadDirection{Dir: quad.Object, Quads: Quads{}}},
		},
		opt: true,
		expect: Save{
			Tags: []string{"id"},
			From: Intersect{
				QuadsAct{Result: quad.Subject},
				Unique{QuadsAct{Result: quad.Object}},
			},
		},
	},
	{ // collapse empty set
		from: Intersect{Quads{
			{Dir: quad.Subject, Values: Optional{Union{
				Unique{QuadDirection{
					Dir: quad.Predicate,
					Quads: Intersect{Quads{
						{Dir: quad.Object,
							Values: Lookup{quad.IRI("no")},
						},
					}},
				}},
			}}},
		}},
		opt:    true,
		expect: nil,
	},
	{ // remove "all nodes" in intersect, merge Fixed and order them first
		from: Intersect{
			AllNodes{},
			Fixed{intVal(1), intVal(2)},
			Save{From: AllNodes{}, Tags: []string{"all"}},
			Fixed{intVal(2)},
		},
		opt: true,
		expect: Save{
			From: Intersect{
				Fixed{intVal(1), intVal(2)},
				Fixed{intVal(2)},
			},
			Tags: []string{"all"},
		},
	},
	{ // remove HasA-LinksTo pairs
		from: QuadDirection{
			Dir: quad.Subject,
			Quads: Quads{{
				Dir:    quad.Subject,
				Values: Fixed{intVal(1)},
			}},
		},
		opt:    true,
		expect: Fixed{intVal(1)},
	},
	{ // pop fixed tags to the top of the tree
		from: QuadDirection{Dir: quad.Subject, Quads: Quads{
			QuadFilter{Dir: quad.Predicate, Values: Intersect{
				FixedTags{
					Tags: map[string]graph.Value{"foo": intVal(1)},
					On: QuadDirection{Dir: quad.Subject,
						Quads: Quads{
							QuadFilter{Dir: quad.Object, Values: FixedTags{
								Tags: map[string]graph.Value{"bar": intVal(2)},
								On:   Fixed{intVal(3)},
							}},
						},
					},
				},
			}},
		}},
		opt: true,
		expect: FixedTags{
			Tags: map[string]graph.Value{"foo": intVal(1), "bar": intVal(2)},
			On: QuadDirection{Dir: quad.Subject, Quads: Quads{
				QuadFilter{Dir: quad.Predicate, Values: QuadsAct{
					Result: quad.Subject,
					Filter: map[quad.Direction]graph.Value{quad.Object: intVal(3)},
				}},
			}},
		},
	},
}

type lookupQuadStore map[quad.Value]graph.Value

func (lookupQuadStore) ApplyDeltas(_ []graph.Delta, _ graph.IgnoreOpts) error {
	panic("not implemented")
}
func (lookupQuadStore) Quad(_ graph.Value) quad.Quad {
	panic("not implemented")
}
func (lookupQuadStore) QuadIterator(_ quad.Direction, _ graph.Value) graph.Iterator {
	panic("not implemented")
}
func (lookupQuadStore) NodesAllIterator() graph.Iterator {
	panic("not implemented")
}
func (lookupQuadStore) QuadsAllIterator() graph.Iterator {
	panic("not implemented")
}
func (qs lookupQuadStore) ValueOf(v quad.Value) graph.Value {
	return qs[v]
}
func (lookupQuadStore) NameOf(_ graph.Value) quad.Value {
	panic("not implemented")
}
func (lookupQuadStore) Size() int64 {
	panic("not implemented")
}
func (lookupQuadStore) Horizon() graph.PrimaryKey {
	panic("not implemented")
}
func (lookupQuadStore) FixedIterator() graph.FixedIterator {
	panic("not implemented")
}
func (lookupQuadStore) OptimizeIterator(_ graph.Iterator) (graph.Iterator, bool) {
	panic("not implemented")
}
func (lookupQuadStore) Close() error {
	panic("not implemented")
}
func (lookupQuadStore) QuadDirection(_ graph.Value, _ quad.Direction) graph.Value {
	panic("not implemented")
}
func (lookupQuadStore) Type() string {
	panic("not implemented")
}

func TestOptimize(t *testing.T) {
	for _, c := range optimizeCases {
		qs := c.qs
		got, opt := c.from.Optimize(qs)
		assert.Equal(t, c.expect, got)
		assert.Equal(t, c.opt, opt)
	}
}
