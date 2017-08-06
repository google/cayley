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

package shapetest

import (
	"context"
	"reflect"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/graphtest"
	"github.com/cayleygraph/cayley/graph/iterator"
	_ "github.com/cayleygraph/cayley/graph/memstore"
	. "github.com/cayleygraph/cayley/graph/shape"
	"github.com/cayleygraph/cayley/quad"
	_ "github.com/cayleygraph/cayley/writer"
)

// This is a simple test graph.
//
//  +-------+                        +------+
//  | alice |-----                 ->| fred |<--
//  +-------+     \---->+-------+-/  +------+   \-+-------+
//                ----->| #bob# |       |         | emily |
//  +---------+--/  --->+-------+       |         +-------+
//  | charlie |    /                    v
//  +---------+   /                  +--------+
//    \---    +--------+             | #greg# |
//        \-->| #dani# |------------>+--------+
//            +--------+

func makeTestStore(t testing.TB, fnc graphtest.DatabaseFunc, quads ...quad.Quad) (graph.QuadStore, func()) {
	if len(quads) == 0 {
		quads = graphtest.LoadGraph(t, "data/testdata.nq")
	}
	var (
		qs     graph.QuadStore
		opts   graph.Options
		closer = func() {}
	)
	if fnc != nil {
		qs, opts, closer = fnc(t)
	} else {
		qs, _ = graph.NewQuadStore("memstore", "", nil)
	}
	_ = graphtest.MakeWriter(t, qs, opts, quads...)
	return qs, closer
}

func runTopLevel(qs graph.QuadStore, path Path, opt bool) ([]quad.Value, error) {
	pb := path.Iterate(context.TODO(), qs)
	if !opt {
		pb = pb.UnOptimized()
	}
	return pb.Paths(false).AllValues(qs)
}

func runTag(qs graph.QuadStore, path Path, tag string, opt bool) ([]quad.Value, error) {
	var out []quad.Value
	pb := path.Iterate(context.TODO(), qs)
	if !opt {
		pb = pb.UnOptimized()
	}
	err := pb.Paths(true).TagEach(func(tags map[string]graph.Value) {
		if t, ok := tags[tag]; ok {
			out = append(out, qs.NameOf(t))
		}
	})
	return out, err
}

// Define morphisms without a QuadStore

const (
	vFollows   = quad.IRI("follows")
	vAre       = quad.IRI("are")
	vStatus    = quad.IRI("status")
	vPredicate = quad.IRI("predicates")

	vCool       = quad.String("cool_person")
	vSmart      = quad.String("smart_person")
	vSmartGraph = quad.IRI("smart_graph")

	vAlice   = quad.IRI("alice")
	vBob     = quad.IRI("bob")
	vCharlie = quad.IRI("charlie")
	vDani    = quad.IRI("dani")
	vFred    = quad.IRI("fred")
	vGreg    = quad.IRI("greg")
	vEmily   = quad.IRI("emily")
)

//var (
//	grandfollows = StartMorphism().Out(vFollows).Out(vFollows)
//)

var cases = []struct {
	message   string
	path      Path
	expect    []quad.Value
	expectAlt []quad.Value
	tag       string
}{
	{
		message: "use out",
		path:    Start(vAlice).Out(vFollows),
		expect:  []quad.Value{vBob},
	},
	{
		message: "use out (any)",
		path:    Start(vBob).Out(),
		expect:  []quad.Value{vFred, vCool},
	},
	{
		message: "use out (raw)",
		path:    Start(quad.Raw(vAlice.String())).Out(quad.Raw(vFollows.String())),
		expect:  []quad.Value{vBob},
	},
	{
		message: "use in",
		path:    Start(vBob).In(vFollows),
		expect:  []quad.Value{vAlice, vCharlie, vDani},
	},
	{
		message: "use in (any)",
		path:    Start(vBob).In(),
		expect:  []quad.Value{vAlice, vCharlie, vDani},
	},
	{
		message: "use in with filter",
		path:    Start(vBob).In(vFollows).Compare(iterator.CompareGT, quad.IRI("c")),
		expect:  []quad.Value{vCharlie, vDani},
	},
	{
		message: "use in with regex",
		path:    Start(vBob).In(vFollows).Regexp(regexp.MustCompile("ar?li.*e")),
		expect:  nil,
	},
	{
		message: "use in with regex (include IRIs)",
		path:    Start(vBob).In(vFollows).RegexpWithRefs(regexp.MustCompile("ar?li.*e")),
		expect:  []quad.Value{vAlice, vCharlie},
	},
	{
		message: "use path Out",
		path:    Start(vBob).Out(Start(vPredicate).Out(vAre)),
		expect:  []quad.Value{vFred, vCool},
	},
	{
		message: "use path Out (raw)",
		path:    Start(quad.Raw(vBob.String())).Out(Start(quad.Raw(vPredicate.String())).Out(quad.Raw(vAre.String()))),
		expect:  []quad.Value{vFred, vCool},
	},
	{
		message: "use And",
		path: Start(vDani).Out(vFollows).And(
			Start(vCharlie).Out(vFollows).Shape()),
		expect: []quad.Value{vBob},
	},
	{
		message: "use Or",
		path: Start(vFred).Out(vFollows).Or(
			Start(vAlice).Out(vFollows).Shape()),
		expect: []quad.Value{vBob, vGreg},
	},
	{
		message: "implicit All",
		path:    Start(),
		expect:  []quad.Value{vAlice, vBob, vCharlie, vDani, vEmily, vFred, vGreg, vFollows, vStatus, vCool, vPredicate, vAre, vSmartGraph, vSmart},
	},
	//{
	//	message: "follow",
	//	path:    StartValues(vCharlie).Follow(StartMorphism().Out(vFollows).Out(vFollows)),
	//	expect:  []quad.Value{vBob, vFred, vGreg},
	//},
	//{
	//	message: "followR",
	//	path:    StartValues(vFred).FollowReverse(StartMorphism().Out(vFollows).Out(vFollows)),
	//	expect:  []quad.Value{vAlice, vCharlie, vDani},
	//},
	//{
	//	message: "is, tag, instead of FollowR",
	//	path:    StartValues().Tag("first").Follow(StartMorphism().Out(vFollows).Out(vFollows)).Is(vFred),
	//	expect:  []quad.Value{vAlice, vCharlie, vDani},
	//	tag:     "first",
	//},
	{
		message: "use Except to filter out a single vertex",
		path:    Start(vAlice, vBob).Except(Start(vAlice).Shape()),
		expect:  []quad.Value{vBob},
	},
	{
		message: "use chained Except",
		path:    Start(vAlice, vBob, vCharlie).Except(Start(vBob).Shape()).Except(Start(vAlice).Shape()),
		expect:  []quad.Value{vCharlie},
	},
	{
		message: "use Unique",
		path:    Start(vAlice, vBob, vCharlie).Out(vFollows).Unique(),
		expect:  []quad.Value{vBob, vDani, vFred},
	},
	{
		message: "show a simple save",
		path:    Start().Save(vStatus, "somecool"),
		tag:     "somecool",
		expect:  []quad.Value{vCool, vCool, vCool, vSmart, vSmart},
	},
	{
		message: "show a simple saveR",
		path:    Start(vCool).SaveReverse(vStatus, "who"),
		tag:     "who",
		expect:  []quad.Value{vGreg, vDani, vBob},
	},
	{
		message: "show a simple Has",
		path:    Start().HasValues(vStatus, false, vCool),
		expect:  []quad.Value{vGreg, vDani, vBob},
	},
	{
		message:   "use Limit",
		path:      Start().HasValues(vStatus, false, vCool).Limit(2),
		expect:    []quad.Value{vBob, vDani},
		expectAlt: []quad.Value{vBob, vGreg}, // TODO(dennwc): resolve this ordering issue
	},
	{
		message:   "use Skip",
		path:      Start().HasValues(vStatus, false, vCool).Page(2, 0),
		expect:    []quad.Value{vGreg},
		expectAlt: []quad.Value{vDani},
	},
	{
		message:   "use Skip and Limit",
		path:      Start().HasValues(vStatus, false, vCool).Page(1, 1),
		expect:    []quad.Value{vDani},
		expectAlt: []quad.Value{vGreg},
	},
	{
		message: "show a double Has",
		path:    Start().HasValues(vStatus, false, vCool).HasValues(vFollows, false, vFred),
		expect:  []quad.Value{vBob},
	},
	{
		message: "show a simple HasReverse",
		path:    Start().HasValues(vStatus, true, vBob),
		expect:  []quad.Value{vCool},
	},
	//{
	//	message: "use .Tag()-.Is()-.Back()",
	//	path:    StartValues(vBob).In(vFollows).Tag("foo").Out(vStatus).Is(vCool).Back("foo"),
	//	expect:  []quad.Value{vDani},
	//},
	//{
	//	message: "do multiple .Back()s",
	//	path:    StartValues(vEmily).Out(vFollows).Tag("f").Out(vFollows).Out(vStatus).Is(vCool).Back("f").In(vFollows).In(vFollows).Tag("acd").Out(vStatus).Is(vCool).Back("f"),
	//	tag:     "acd",
	//	expect:  []quad.Value{vDani},
	//},
	{
		message: "InPredicates()",
		path:    Start(vBob).Predicates(true),
		expect:  []quad.Value{vFollows},
	},
	{
		message: "OutPredicates()",
		path:    Start(vBob).Predicates(false),
		expect:  []quad.Value{vFollows, vStatus},
	},
	// Morphism tests
	//{
	//	message: "show simple morphism",
	//	path:    StartValues(vCharlie).Follow(grandfollows),
	//	expect:  []quad.Value{vGreg, vFred, vBob},
	//},
	//{
	//	message: "show reverse morphism",
	//	path:    StartValues(vFred).FollowReverse(grandfollows),
	//	expect:  []quad.Value{vAlice, vCharlie, vDani},
	//},
	// Context tests
	{
		message: "query without label limitation",
		path:    Start(vGreg).Out(vStatus),
		expect:  []quad.Value{vSmart, vCool},
	},
	{
		message: "query with label limitation",
		path:    Start(vGreg).LabelContext(vSmartGraph).Out(vStatus),
		expect:  []quad.Value{vSmart},
	},
	//{
	//	message: "reverse context",
	//	path:    StartValues(vGreg).Tag("base").LabelContext(vSmartGraph).Out(vStatus).Tag("status").Back("base"),
	//	expect:  []quad.Value{vGreg},
	//},
	// Optional tests
	{
		message: "save limits top level",
		path:    Start(vBob, vCharlie).Out(vFollows).Save(vStatus, "statustag"),
		expect:  []quad.Value{vBob, vDani},
	},
	{
		message: "optional still returns top level",
		path:    Start(vBob, vCharlie).Out(vFollows).SaveOpt(vStatus, "statustag", false, true),
		expect:  []quad.Value{vBob, vFred, vDani},
	},
	{
		message: "optional has the appropriate tags",
		path:    Start(vBob, vCharlie).Out(vFollows).SaveOpt(vStatus, "statustag", false, true),
		tag:     "statustag",
		expect:  []quad.Value{vCool, vCool},
	},
	//{
	//	message: "composite paths (clone paths)",
	//	path: func() *Path {
	//		alice_path := StartValues(vAlice)
	//		_ = alice_path.Out(vFollows)
	//
	//		return alice_path
	//	}(),
	//	expect: []quad.Value{vAlice},
	//},
	//{
	//	message: "follow recursive",
	//	path:    StartValues(vCharlie).FollowRecursive(vFollows, nil),
	//	expect:  []quad.Value{vBob, vDani, vFred, vGreg},
	//},
	{
		message: "find non-existent",
		path:    Start(quad.IRI("<not-existing>")),
		expect:  nil,
	},
}

func RunTestShapes(t testing.TB, fnc graphtest.DatabaseFunc) {
	for _, ftest := range []func(testing.TB, graphtest.DatabaseFunc){
	//testFollowRecursive,
	} {
		ftest(t, fnc)
	}
	qs, closer := makeTestStore(t, fnc)
	defer closer()

	for _, test := range cases {
		var (
			got []quad.Value
			err error
		)
		//t.Logf("tree:\n%#v", test.path.Shape())
		for _, opt := range []bool{true, false} {
			start := time.Now()
			if test.tag == "" {
				got, err = runTopLevel(qs, test.path, opt)
			} else {
				got, err = runTag(qs, test.path, test.tag, opt)
			}
			dt := time.Since(start)
			unopt := ""
			if !opt {
				unopt = " (unoptimized)"
			}
			if err != nil {
				t.Errorf("Failed to %s%s: %v", test.message, unopt, err)
				continue
			}
			sort.Sort(quad.ByValueString(got))
			sort.Sort(quad.ByValueString(test.expect))
			eq := reflect.DeepEqual(got, test.expect)
			if !eq && test.expectAlt != nil {
				eq = reflect.DeepEqual(got, test.expectAlt)
			}
			if !eq {
				t.Errorf("Failed to %s%s, got: %v(%d) expected: %v(%d)", test.message, unopt, got, len(got), test.expect, len(test.expect))
				//t.Errorf("tree:\n%#v", test.path.Shape())
			} else {
				t.Logf("%s%s: %v", test.message, unopt, dt)
			}
		}
	}
}

//func testFollowRecursive(t testing.TB, fnc graphtest.DatabaseFunc) {
//	qs, closer := makeTestStore(t, fnc, []quad.Quad{
//		quad.MakeIRI("a", "parent", "b", ""),
//		quad.MakeIRI("b", "parent", "c", ""),
//		quad.MakeIRI("c", "parent", "d", ""),
//		quad.MakeIRI("c", "labels", "tag", ""),
//		quad.MakeIRI("d", "parent", "e", ""),
//		quad.MakeIRI("d", "labels", "tag", ""),
//	}...)
//	defer closer()
//
//	qu := StartValues(quad.IRI("a")).FollowRecursive(
//		StartMorphism().Out(quad.IRI("parent")), nil,
//	).Has(quad.IRI("labels"), quad.IRI("tag"))
//
//	expect := []quad.Value{quad.IRI("c"), quad.IRI("d")}
//
//	const msg = "follows recursive order"
//
//	for _, opt := range []bool{true, false} {
//		got, err := runTopLevel(qs, qu, opt)
//		unopt := ""
//		if !opt {
//			unopt = " (unoptimized)"
//		}
//		if err != nil {
//			t.Errorf("Failed to check %s%s: %v", msg, unopt, err)
//			continue
//		}
//		sort.Sort(quad.ByValueString(got))
//		sort.Sort(quad.ByValueString(expect))
//		if !reflect.DeepEqual(got, expect) {
//			t.Errorf("Failed to %s%s, got: %v(%d) expected: %v(%d)", msg, unopt, got, len(got), expect, len(expect))
//		}
//	}
//}
