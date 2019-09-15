// Copyright 2017 The Cayley Authors. All rights reserved.
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

package gizmo

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/graphtest/testutil"
	_ "github.com/cayleygraph/cayley/graph/memstore"
	"github.com/cayleygraph/cayley/quad"
	"github.com/cayleygraph/cayley/query"
	_ "github.com/cayleygraph/cayley/writer"

	// register global namespace for tests
	_ "github.com/cayleygraph/cayley/voc/rdf"
)

// This is a simple test graph used for testing
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
//

func makeTestSession(data []quad.Quad) *Session {
	qs, _ := graph.NewQuadStore("memstore", "", nil)
	w, _ := graph.NewQuadWriter("single", qs, nil)
	for _, t := range data {
		w.AddQuad(t)
	}
	return NewSession(qs)
}

func intVal(v int) string {
	return quad.Int(v).String()
}

const multiGraphTestFile = "../../data/testdata_multigraph.nq"


type IDDocument = map[string]string;

func newIDDocument(id string) IDDocument {
	return map[string]string{ "@id": id }
}

var testQueries = []struct {
	message string
	data    []quad.Quad
	query   string
	limit   int
	tag     string
	file 	string
	expect  []interface{}
	err     bool // TODO(dennwc): define error types for Gizmo and handle them
}{
	// Simple query tests.
	{
		message: "get a single vertex",
		query: `
			g.V("<alice>").All()
		`,
		expect: []interface{}{newIDDocument("alice")},
	},
	{
		message: "use .GetLimit",
		query: `
			g.V().GetLimit(5)
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("bob"), newIDDocument("follows"), newIDDocument("fred"), newIDDocument("status")},
	},
	{
		message: "get a single vertex (IRI)",
		query: `
			g.V(iri("alice")).All()
		`,
		expect: []interface{}{newIDDocument("alice")},
	},
	{
		message: "use .Out()",
		query: `
			g.V("<alice>").Out("<follows>").All()
		`,
		expect: []interface{}{newIDDocument("bob")},
	},
	{
		message: "use .Out() (IRI)",
		query: `
			g.V(iri("alice")).Out(iri("follows")).All()
		`,
		expect: []interface{}{newIDDocument("bob")},
	},
	{
		message: "use .Out() (any)",
		query: `
			g.V("<bob>").Out().All()
		`,
		expect: []interface{}{newIDDocument("fred"), "cool_person"},
	},
	{
		message: "use .In()",
		query: `
			g.V("<bob>").In("<follows>").All()
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("charlie"), newIDDocument("dani")},
	},
	{
		message: "use .In() (any)",
		query: `
			g.V("<bob>").In().All()
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("charlie"), newIDDocument("dani")},
	},
	{
		message: "use .In() with .Filter()",
		query: `
			g.V("<bob>").In("<follows>").Filter(gt(iri("c")),lt(iri("d"))).All()
		`,
		expect: []interface{}{newIDDocument("charlie")},
	},
	{
		message: "use .In() with .Filter(regex)",
		query: `
			g.V("<bob>").In("<follows>").Filter(regex("ar?li.*e")).All()
		`,
		expect: nil,
	},
	{
		message: "use .In() with .Filter(prefix)",
		query: `
			g.V("<bob>").In("<follows>").Filter(like("al%")).All()
		`,
		expect: []interface{}{newIDDocument("alice")},
	},
	{
		message: "use .In() with .Filter(wildcard)",
		query: `
			g.V("<bob>").In("<follows>").Filter(like("a?i%e")).All()
		`,
		expect: []interface{}{newIDDocument("alice")},
	},
	{
		message: "use .In() with .Filter(regex with IRIs)",
		query: `
			g.V("<bob>").In("<follows>").Filter(regex("ar?li.*e", true)).All()
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("charlie")},
	},
	{
		message: "use .In() with .Filter(regex with IRIs)",
		query: `
			g.V("<bob>").In("<follows>").Filter(regex(iri("ar?li.*e"))).All()
		`,
		err: true,
	},
	{
		message: "use .In() with .Filter(regex,gt)",
		query: `
			g.V("<bob>").In("<follows>").Filter(regex("ar?li.*e", true),gt(iri("c"))).All()
		`,
		expect: []interface{}{newIDDocument("charlie")},
	},
	{
		message: "use .Both()",
		query: `
			g.V("<fred>").Both("<follows>").All()
		`,
		expect: []interface{}{newIDDocument("bob"), newIDDocument("greg"), newIDDocument("emily")},
	},
	{
		message: "use .Both() with tag",
		query: `
			g.V("<fred>").Both(null, "pred").All()
		`,
		tag:    "pred",
		expect: []interface{}{newIDDocument("follows"), newIDDocument("follows"), newIDDocument("follows")},
	},
	{
		message: "use .Tag()-.Is()-.Back()",
		query: `
			g.V("<bob>").In("<follows>").Tag("foo").Out("<status>").Is("cool_person").Back("foo").All()
		`,
		expect: []interface{}{newIDDocument("dani")},
	},
	{
		message: "separate .Tag()-.Is()-.Back()",
		query: `
			x = g.V("<charlie>").Out("<follows>").Tag("foo").Out("<status>").Is("cool_person").Back("foo")
			x.In("<follows>").Is("<dani>").Back("foo").All()
		`,
		expect: []interface{}{newIDDocument("bob")},
	},
	{
		message: "do multiple .Back()s",
		query: `
			g.V("<emily>").Out("<follows>").As("f").Out("<follows>").Out("<status>").Is("cool_person").Back("f").In("<follows>").In("<follows>").As("acd").Out("<status>").Is("cool_person").Back("f").All()
		`,
		tag:    "acd",
		expect: []interface{}{newIDDocument("dani")},
	},
	{
		message: "use Except to filter out a single vertex",
		query: `
			g.V("<alice>", "<bob>").Except(g.V("<alice>")).All()
		`,
		expect: []interface{}{newIDDocument("bob")},
	},
	{
		message: "use chained Except",
		query: `
			g.V("<alice>", "<bob>", "<charlie>").Except(g.V("<bob>")).Except(g.V("<charlie>")).All()
		`,
		expect: []interface{}{newIDDocument("alice")},
	},

	{
		message: "use Unique",
		query: `
			g.V("<alice>", "<bob>", "<charlie>").Out("<follows>").Unique().All()
		`,
		expect: []interface{}{newIDDocument("bob"), newIDDocument("dani"), newIDDocument("fred")},
	},

	// Morphism tests.
	{
		message: "show simple morphism",
		query: `
			grandfollows = g.M().Out("<follows>").Out("<follows>")
			g.V("<charlie>").Follow(grandfollows).All()
		`,
		expect: []interface{}{newIDDocument("greg"), newIDDocument("fred"), newIDDocument("bob")},
	},
	{
		message: "show reverse morphism",
		query: `
			grandfollows = g.M().Out("<follows>").Out("<follows>")
			g.V("<fred>").FollowR(grandfollows).All()
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("charlie"), newIDDocument("dani")},
	},

	// Intersection tests.
	{
		message: "show simple intersection",
		query: `
			function follows(x) { return g.V(x).Out("<follows>") }
			follows("<dani>").And(follows("<charlie>")).All()
		`,
		expect: []interface{}{newIDDocument("bob")},
	},
	{
		message: "show simple morphism intersection",
		query: `
			grandfollows = g.M().Out("<follows>").Out("<follows>")
			function gfollows(x) { return g.V(x).Follow(grandfollows) }
			gfollows("<alice>").And(gfollows("<charlie>")).All()
		`,
		expect: []interface{}{newIDDocument("fred")},
	},
	{
		message: "show double morphism intersection",
		query: `
			grandfollows = g.M().Out("<follows>").Out("<follows>")
			function gfollows(x) { return g.V(x).Follow(grandfollows) }
			gfollows("<emily>").And(gfollows("<charlie>")).And(gfollows("<bob>")).All()
		`,
		expect: []interface{}{newIDDocument("greg")},
	},
	{
		message: "show reverse intersection",
		query: `
			grandfollows = g.M().Out("<follows>").Out("<follows>")
			g.V("<greg>").FollowR(grandfollows).Intersect(g.V("<fred>").FollowR(grandfollows)).All()
		`,
		expect: []interface{}{newIDDocument("charlie")},
	},
	{
		message: "show standard sort of morphism intersection, continue follow",
		query: `gfollowers = g.M().In("<follows>").In("<follows>")
			function cool(x) { return g.V(x).As("a").Out("<status>").Is("cool_person").Back("a") }
			cool("<greg>").Follow(gfollowers).Intersect(cool("<bob>").Follow(gfollowers)).All()
		`,
		expect: []interface{}{newIDDocument("charlie")},
	},
	{
		message: "test Or()",
		query: `
			g.V("<bob>").Out("<follows>").Or(g.V().Has("<status>", "cool_person")).All()
		`,
		expect: []interface{}{newIDDocument("fred"), newIDDocument("bob"), newIDDocument("greg"), newIDDocument("dani")},
	},

	// Has tests.
	{
		message: "show a simple Has",
		query: `
				g.V().Has("<status>", "cool_person").All()
		`,
		expect: []interface{}{newIDDocument("greg"), newIDDocument("dani"), newIDDocument("bob")},
	},
	{
		message: "show a simple HasR",
		query: `
				g.V().HasR("<status>", "<bob>").All()
		`,
		expect: []interface{}{"cool_person"},
	},
	{
		message: "show a double Has",
		query: `
				g.V().Has("<status>", "cool_person").Has("<follows>", "<fred>").All()
		`,
		expect: []interface{}{newIDDocument("bob")},
	},
	{
		message: "show a Has with filter",
		query: `
				g.V().Has("<follows>", gt("<f>")).All()
		`,
		expect: []interface{}{newIDDocument("bob"), newIDDocument("dani"), newIDDocument("emily"), newIDDocument("fred")},
	},

	// Skip/Limit tests.
	{
		message: "use Limit",
		query: `
				g.V().Has("<status>", "cool_person").Limit(2).All()
		`,
		expect: []interface{}{newIDDocument("bob"), newIDDocument("dani")},
	},
	{
		message: "use Skip",
		query: `
				g.V().Has("<status>", "cool_person").Skip(2).All()
		`,
		expect: []interface{}{newIDDocument("greg")},
	},
	{
		message: "use Skip and Limit",
		query: `
				g.V().Has("<status>", "cool_person").Skip(1).Limit(1).All()
		`,
		expect: []interface{}{newIDDocument("dani")},
	},

	{
		message: "show Count",
		query: `
				g.V().Has("<status>").Count()
		`,
		expect: []interface{}{"5"},
	},
	{
		message: "use Count value",
		query: `
				g.Emit(g.V().Has("<status>").Count()+1)
		`,
		expect: []interface{}{"6"},
	},

	// Tag tests.
	{
		message: "show a simple save",
		query: `
			g.V().Save("<status>", "somecool").All()
		`,
		tag:    "somecool",
		expect: []interface{}{"cool_person", "cool_person", "cool_person", "smart_person", "smart_person"},
	},
	{
		message: "show a simple save optional",
		query: `
			g.V("<bob>","<charlie>").Out("<follows>").SaveOpt("<status>", "somecool").All()
		`,
		tag:    "somecool",
		expect: []interface{}{"cool_person", "cool_person"},
	},
	{
		message: "show a simple saveR",
		query: `
			g.V("cool_person").SaveR("<status>", "who").All()
		`,
		tag:    "who",
		expect: []interface{}{newIDDocument("greg"), newIDDocument("dani"), newIDDocument("bob")},
	},
	{
		message: "show an out save",
		query: `
			g.V("<dani>").Out(null, "pred").All()
		`,
		tag:    "pred",
		expect: []interface{}{newIDDocument("follows"), newIDDocument("follows"), newIDDocument("status")},
	},
	{
		message: "show a tag list",
		query: `
			g.V("<dani>").Out(null, ["pred", "foo", "bar"]).All()
		`,
		tag:    "foo",
		expect: []interface{}{newIDDocument("follows"), newIDDocument("follows"), newIDDocument("status")},
	},
	{
		message: "show a pred list",
		query: `
			g.V("<dani>").Out(["<follows>", "<status>"]).All()
		`,
		expect: []interface{}{newIDDocument("bob"), "<greg>", "cool_person"},
	},
	{
		message: "show a predicate path",
		query: `
			g.V("<dani>").Out(g.V("<follows>"), "pred").All()
		`,
		expect: []interface{}{newIDDocument("bob"), newIDDocument("greg")},
	},
	{
		message: "list all bob's incoming predicates",
		query: `
		  g.V("<bob>").InPredicates().All()
		`,
		expect: []interface{}{newIDDocument("follows")},
	},
	{
		message: "save all bob's incoming predicates",
		query: `
		  g.V("<bob>").SaveInPredicates("pred").All()
		`,
		expect: []interface{}{newIDDocument("follows"), newIDDocument("follows"), newIDDocument("follows")},
		tag:    "pred",
	},
	{
		message: "list all labels",
		query: `
		  g.V().Labels().All()
		`,
		expect: []interface{}{newIDDocument("smart_graph")},
	},
	{
		message: "list all in predicates",
		query: `
		  g.V().InPredicates().All()
		`,
		expect: []interface{}{newIDDocument("are"), newIDDocument("follows"), newIDDocument("status")},
	},
	{
		message: "list all out predicates",
		query: `
		  g.V().OutPredicates().All()
		`,
		expect: []interface{}{newIDDocument("are"), newIDDocument("follows"), newIDDocument("status")},
	},
	{
		message: "traverse using LabelContext",
		query: `
			g.V("<greg>").LabelContext("<smart_graph>").Out("<status>").All()
		`,
		expect: []interface{}{"smart_person"},
	},
	{
		message: "open and close a LabelContext",
		query: `
			g.V().LabelContext("<smart_graph>").In("<status>").LabelContext(null).In("<follows>").All()
		`,
		expect: []interface{}{newIDDocument("dani"), newIDDocument("fred")},
	},
	{
		message: "issue #254",
		query:   `g.V({"id":"<alice>"}).All()`,
		expect:  nil, err: true,
	},
	{
		message: "roundtrip values",
		query: `
		v = g.V("<bob>").ToValue()
		s = g.V(v).Out("<status>").ToValue()
		g.V(s).All()
		`,
		expect: []interface{}{"cool_person"},
	},
	{
		message: "roundtrip values (tag map)",
		query: `
		v = g.V("<bob>").TagValue()
		s = g.V(v.id).Out("<status>").TagValue()
		g.V(s.id).All()
		`,
		expect: []interface{}{"cool_person"},
	},
	{
		message: "show ToArray",
		query: `
			arr = g.V("<bob>").In("<follows>").ToArray()
			for (i in arr) g.Emit(arr[i]);
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("charlie"), newIDDocument("dani")},
	},
	{
		message: "show ToArray with limit",
		query: `
			arr = g.V("<bob>").In("<follows>").ToArray(2)
			for (i in arr) g.Emit(arr[i]);
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("charlie")},
	},
	{
		message: "show ForEach",
		query: `
			g.V("<bob>").In("<follows>").ForEach(function(o){g.Emit(o.id)});
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("charlie"), newIDDocument("dani")},
	},
	{
		message: "show ForEach with limit",
		query: `
			g.V("<bob>").In("<follows>").ForEach(2, function(o){g.Emit(o.id)});
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("charlie")},
	},
	{
		message: "clone paths",
		query: `
			var alice = g.V('<alice>')
			g.Emit(alice.ToValue())
			var out = alice.Out('<follows>')
			g.Emit(out.ToValue())
			g.Emit(alice.ToValue())
		`,
		expect: []interface{}{newIDDocument("alice"), newIDDocument("bob"), newIDDocument("alice")},
	},
	{
		message: "default namespaces",
		query: `
			g.AddDefaultNamespaces()
			g.Emit(g.Uri('rdf:type'))
		`,
		expect: []interface{}{newIDDocument("http/)/www.w3.org/1999/02/22-rdf-syntaxnewIDDocument(-nstype")},
	},
	{
		message: "add namespace",
		query: `
			g.AddNamespace('ex','http://example.net/')
			g.Emit(g.Uri('ex:alice'))
		`,
		expect: []interface{}{newIDDocument("http/)/examplenewIDDocument(.netalice")},
	},
	{
		message: "recursive follow",
		query: `
			g.V("<charlie>").FollowRecursive("<follows>").All();
		`,
		expect: []interface{}{newIDDocument("bob"), newIDDocument("dani"), newIDDocument("fred"), newIDDocument("greg")},
	},
	{
		message: "recursive follow tag",
		query: `
			g.V("<charlie>").FollowRecursive("<follows>", "depth").All();
		`,
		tag:    "depth",
		expect: []interface{}{intVal(1), intVal(1), intVal(2), intVal(2)},
	},
	{
		message: "recursive follow path",
		query: `
			g.V("<charlie>").FollowRecursive(g.V().Out("<follows>")).All();
		`,
		expect: []interface{}{newIDDocument("bob"), newIDDocument("dani"), newIDDocument("fred"), newIDDocument("greg")},
	},
	{
		message: "find non-existent",
		query: `
			g.V('<not-existing>').ForEach(function(d){ g.Emit(d); })
		`,
		expect: nil,
	},
	{
		message: "default limit All",
		query: `
			g.V().All()
		`,
		limit:  issue718Limit,
		data:   issue718Graph(),
		expect: issue718Nodes(),
	},
	{
		message: "issue #758. Verify saveOpt respects label context",
		query: `
			g.V("<greg>").LabelContext("<smart_graph>").SaveOpt("<status>", "statusTag").All()
		`,
		tag:    "statusTag",
		file: multiGraphTestFile,
		expect: []interface{}{"smart_person"},
	},
	{
		message: "issue #758. Verify saveR respects label context.",
		query: `
			g.V("smart_person").LabelContext("<other_graph>").SaveR("<status>", "who").All()
		`,
		tag:    "who",
		file: multiGraphTestFile,
		expect: []interface{}{newIDDocument("fred")},
	},
}

func runQueryGetTag(rec func(), g []quad.Quad, qu string, tag string, limit int) ([]interface{}, error) {
	js := makeTestSession(g)
	c := make(chan query.Result, 1)
	go func() {
		defer rec()
		js.Execute(context.TODO(), qu, c, limit)
	}()

	var results []interface{}
	for res := range c {
		if err := res.Err(); err != nil {
			return results, err
		}
		data := res.(*Result)
		if data.Val == nil {
			if val := data.Tags[tag]; val != nil {
				results = append(results, quadValueToString(js.qs.NameOf(val)))
			}
		} else {
			switch v := data.Val.(type) {
			case string:
				results = append(results, v)
			default:
				results = append(results, fmt.Sprint(v))
			}
		}
	}
	return results, nil
}

func TestGizmo(t *testing.T) {

	simpleGraph := testutil.LoadGraph(t, "../../data/testdata.nq")
	multiGraph := testutil.LoadGraph(t, multiGraphTestFile)

	for _, test := range testQueries {
		test := test
		t.Run(test.message, func(t *testing.T) {
			rec := func() {
				if r := recover(); r != nil {
					t.Errorf("Unexpected panic on %s: %v", test.message, r)
				}
			}
			defer rec()
			if test.tag == "" {
				test.tag = TopResultTag
			}
			quads := simpleGraph
			if (test.file == multiGraphTestFile){
				quads = multiGraph
			}
			
			if test.data != nil {
				quads = test.data
			}
			limit := test.limit
			if limit == 0 {
				limit = -1
			}
			got, err := runQueryGetTag(rec, quads, test.query, test.tag, limit)
			if err != nil {
				if test.err {
					return //expected
				}
				t.Error(err)
			}
			assert.ElementsMatch(t, got, test.expect)
			if !reflect.DeepEqual(got, test.expect) {
				t.Errorf("got: %v expected: %v", got, test.expect)
			}
		})
	}
}

var issue160TestGraph = []quad.Quad{
	quad.MakeRaw("alice", "follows", "bob", ""),
	quad.MakeRaw("bob", "follows", "alice", ""),
	quad.MakeRaw("charlie", "follows", "bob", ""),
	quad.MakeRaw("dani", "follows", "charlie", ""),
	quad.MakeRaw("dani", "follows", "alice", ""),
	quad.MakeRaw("alice", "is", "cool", ""),
	quad.MakeRaw("bob", "is", "not cool", ""),
	quad.MakeRaw("charlie", "is", "cool", ""),
	quad.MakeRaw("danie", "is", "not cool", ""),
}

func TestIssue160(t *testing.T) {
	qu := `g.V().Tag('query').Out(raw('follows')).Out(raw('follows')).ForEach(function (item) { if (item.id !== item.query) g.Emit({ id: item.id }); })`
	expect := []interface{}{
		"****\nid : alice\n",
		"****\nid : bob\n",
		"****\nid : bob\n",
	}

	ses := makeTestSession(issue160TestGraph)
	c := make(chan query.Result, 5)
	go ses.Execute(context.TODO(), qu, c, 100)
	var got []string
	for res := range c {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()
			got = append(got, ses.FormatREPL(res))
		}()
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("Unexpected result, got: %q expected: %q", got, expect)
	}
}

func TestShapeOf(t *testing.T) {
	ses := makeTestSession(nil)
	const query = `g.V().ForEach(function(x){
g.Emit({id: x.id})
})`
	_, err := ses.ShapeOf(query)
	require.NoError(t, err)
}

const issue718Limit = 5

func issue718Graph() []quad.Quad {
	var quads []quad.Quad
	for i := 0; i < issue718Limit; i++ {
		n := fmt.Sprintf("n%d", i+1)
		quads = append(quads, quad.MakeIRI("a", "b", n, ""))
	}
	return quads
}

func issue718Nodes() []interface{} {
	var nodes []interface{}
	nodes = append(nodes, newIDDocument("a"), newIDDocument("b"))
	for i := 0; i < issue718Limit-2; i++ {
		n := fmt.Sprintf("<n%d>", i+1)
		nodes = append(nodes, n)
	}
	return nodes
}



