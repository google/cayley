package linkedql

import (
	"errors"
	"regexp"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/iterator"
	"github.com/cayleygraph/cayley/query"
	"github.com/cayleygraph/cayley/query/path"
	"github.com/cayleygraph/cayley/query/shape"
	"github.com/cayleygraph/quad"
)

func init() {
	Register(&Vertex{})
	Register(&Out{})
	Register(&As{})
	Register(&Intersect{})
	Register(&Is{})
	Register(&Back{})
	Register(&Both{})
	Register(&Count{})
	Register(&Except{})
	Register(&Filter{})
	Register(&Follow{})
	Register(&FollowReverse{})
	Register(&Has{})
	Register(&HasReverse{})
	Register(&In{})
	Register(&InPredicates{})
	Register(&LabelContext{})
	Register(&Labels{})
	Register(&Limit{})
	Register(&OutPredicates{})
	Register(&Save{})
	Register(&SaveInPredicates{})
	Register(&SaveOptional{})
	Register(&SaveOptionalReverse{})
	Register(&SaveOutPredicates{})
	Register(&SaveReverse{})
	Register(&Skip{})
	Register(&Union{})
	Register(&Unique{})
	Register(&Order{})
}

// Step is the tree representation of a call in a Path context.
// For example:
// g.V(g.IRI("alice")) is represented as &V{ values: quad.Value[]{quad.IRI("alice")} }
// g.V().out(g.IRI("likes")) is represented as &Out{ Via: quad.Value[]{quad.IRI("likes")}, From: &V{} }
type Step interface {
	RegistryItem
	BuildIterator(qs graph.QuadStore) (query.Iterator, error)
}

// ValueStep is a ValueStep that can build a ValueIterator
type ValueStep interface {

	// BuildValueIterator implements ValueStepStep
	BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error)
}

// func buildValueIterator(s Step, qs graph.QuadStore, property string) (*ValueIterator, error) {
// 	it, err := s.BuildIterator(qs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	valueIt, ok := it.(*ValueIterator)
// 	if !ok {
// 		return nil, errors.New(s.Type().String() + " must be called with ValueIterator " + property)
// 	}
// 	return valueIt, err
// }

// Vertex corresponds to g.Vertex() and g.V()
type Vertex struct {
	Values []quad.Value `json:"values"`
}

// Type implements Step
func (s *Vertex) Type() quad.IRI {
	return prefix + "Vertex"
}

// BuildIterator implements Step
func (s *Vertex) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Vertex) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	path := path.StartPath(qs, s.Values...)
	return NewValueIterator(path, qs), nil
}

// Out corresponds to .out()
type Out struct {
	From ValueStep `json:"from"`
	Via  ValueStep `json:"via"`
	Tags []string  `json:"tags"`
}

// Type implements Step
func (s *Out) Type() quad.IRI {
	return prefix + "Out"
}

// BuildIterator implements Step
func (s *Out) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Out) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	viaIt, err := s.Via.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	path := fromIt.path.OutWithTags(s.Tags, viaIt.path)
	return NewValueIterator(path, qs), nil
}

// As corresponds to .tag()
type As struct {
	From ValueStep `json:"from"`
	Tags []string  `json:"tags"`
}

// Type implements Step
func (s *As) Type() quad.IRI {
	return prefix + "As"
}

// BuildIterator implements Step
func (s *As) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *As) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	path := fromIt.path.Tag(s.Tags...)
	return NewValueIterator(path, qs), nil
}

// BuildValueIterator implements ValueStep
func (s *Value) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	// TODO(iddan): support non iterators for query result
	return fromIt, nil
}

// Intersect represents .intersect() and .and()
type Intersect struct {
	From        ValueStep `json:"from"`
	Intersectee ValueStep `json:"intersectee"`
}

// Type implements Step
func (s *Intersect) Type() quad.IRI {
	return prefix + "Intersect"
}

// BuildIterator implements Step
func (s *Intersect) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Intersect) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	intersecteeIt, err := s.Intersectee.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.And(intersecteeIt.path), qs), nil
}

// Is corresponds to .back()
type Is struct {
	From   ValueStep    `json:"from"`
	Values []quad.Value `json:"values"`
}

// Type implements Step
func (s *Is) Type() quad.IRI {
	return prefix + "Is"
}

// BuildIterator implements Step
func (s *Is) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Is) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Is(s.Values...), qs), nil
}

// Back corresponds to .back()
type Back struct {
	From ValueStep `json:"from"`
	Tag  string    `json:"tag"`
}

// Type implements Step
func (s *Back) Type() quad.IRI {
	return prefix + "Back"
}

// BuildIterator implements Step
func (s *Back) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Back) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Back(s.Tag), qs), nil
}

// Both corresponds to .both()
type Both struct {
	From ValueStep `json:"from"`
	Via  ValueStep `json:"via"`
	Tags []string  `json:"tags"`
}

// Type implements Step
func (s *Both) Type() quad.IRI {
	return prefix + "Both"
}

// BuildIterator implements Step
func (s *Both) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Both) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	viaIt, err := s.Via.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.BothWithTags(s.Tags, viaIt.path), qs), nil
}

// Count corresponds to .count()
type Count struct {
	From ValueStep `json:"from"`
}

// Type implements Step
func (s *Count) Type() quad.IRI {
	return prefix + "Count"
}

// BuildIterator implements Step
func (s *Count) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Count) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Count(), qs), nil
}

// Except corresponds to .except() and .difference()
type Except struct {
	From     ValueStep `json:"from"`
	Excepted ValueStep `json:"excepted"`
}

// Type implements Step
func (s *Except) Type() quad.IRI {
	return prefix + "Except"
}

// BuildIterator implements Step
func (s *Except) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Except) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	exceptedIt, err := s.Excepted.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Except(exceptedIt.path), qs), nil
}

// Filter corresponds to filter()
type Filter struct {
	From   ValueStep `json:"from"`
	Filter Operator  `json:"filter"`
}

// Type implements Step
func (s *Filter) Type() quad.IRI {
	return prefix + "Filter"
}

// BuildIterator implements Step
func (s *Filter) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Filter) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	switch filter := s.Filter.(type) {
	case *LessThan:
		return NewValueIterator(fromIt.path.Filter(iterator.Operator(iterator.CompareLT), filter.Value), qs), nil
	case *LessThanEquals:
		return NewValueIterator(fromIt.path.Filter(iterator.Operator(iterator.CompareLTE), filter.Value), qs), nil
	case *GreaterThan:
		return NewValueIterator(fromIt.path.Filter(iterator.Operator(iterator.CompareGT), filter.Value), qs), nil
	case *GreaterThanEquals:
		return NewValueIterator(fromIt.path.Filter(iterator.Operator(iterator.CompareGTE), filter.Value), qs), nil
	case *RegExp:
		expression, err := regexp.Compile(string(filter.Expression))
		if err != nil {
			return nil, errors.New("Invalid RegExp")
		}
		if filter.IncludeIRIs {
			return NewValueIterator(fromIt.path.RegexWithRefs(expression), qs), nil
		}
		return NewValueIterator(fromIt.path.RegexWithRefs(expression), qs), nil
	case *Like:
		return NewValueIterator(fromIt.path.Filters(shape.Wildcard{Pattern: filter.Pattern}), qs), nil
	default:
		return nil, errors.New("Filter is not recognized")
	}
}

// Follow corresponds to .follow()
type Follow struct {
	From     ValueStep `json:"from"`
	Followed ValueStep `json:"followed"`
}

// Type implements Step
func (s *Follow) Type() quad.IRI {
	return prefix + "Follow"
}

// BuildIterator implements Step
func (s *Follow) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Follow) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	followedIt, err := s.Followed.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Follow(followedIt.path), qs), nil
}

// FollowReverse corresponds to .followR()
type FollowReverse struct {
	From     ValueStep `json:"from"`
	Followed ValueStep `json:"followed"`
}

// Type implements Step
func (s *FollowReverse) Type() quad.IRI {
	return prefix + "FollowReverse"
}

// BuildIterator implements Step
func (s *FollowReverse) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *FollowReverse) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	followedIt, err := s.Followed.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.FollowReverse(followedIt.path), qs), nil
}

// Has corresponds to .has()
type Has struct {
	From   ValueStep    `json:"from"`
	Via    ValueStep    `json:"via"`
	Values []quad.Value `json:"values"`
}

// Type implements Step
func (s *Has) Type() quad.IRI {
	return prefix + "Has"
}

// BuildIterator implements Step
func (s *Has) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Has) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	viaIt, err := s.Via.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Has(viaIt.path, s.Values...), qs), nil
}

// HasReverse corresponds to .hasR()
type HasReverse struct {
	From   ValueStep    `json:"from"`
	Via    ValueStep    `json:"via"`
	Values []quad.Value `json:"values"`
}

// Type implements Step
func (s *HasReverse) Type() quad.IRI {
	return prefix + "HasReverse"
}

// BuildIterator implements Step
func (s *HasReverse) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *HasReverse) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	viaIt, err := s.Via.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.HasReverse(viaIt.path, s.Values...), qs), nil
}

// In corresponds to .in()
type In struct {
	From ValueStep `json:"from"`
	Via  ValueStep `json:"via"`
	Tags []string  `json:"tags"`
}

// Type implements Step
func (s *In) Type() quad.IRI {
	return prefix + "In"
}

// BuildIterator implements Step
func (s *In) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *In) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	viaIt, err := s.Via.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.InWithTags(s.Tags, viaIt.path), qs), nil
}

// InPredicates corresponds to .inPredicates()
type InPredicates struct {
	From ValueStep `json:"from"`
}

// Type implements Step
func (s *InPredicates) Type() quad.IRI {
	return prefix + "InPredicates"
}

// BuildIterator implements Step
func (s *InPredicates) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *InPredicates) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.InPredicates(), qs), nil
}

// LabelContext corresponds to .labelContext()
type LabelContext struct {
	From ValueStep `json:"from"`
	// TODO(iddan): Via
}

// Type implements Step
func (s *LabelContext) Type() quad.IRI {
	return prefix + "LabelContext"
}

// BuildIterator implements Step
func (s *LabelContext) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *LabelContext) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.LabelContext(), qs), nil
}

// Labels corresponds to .labels()
type Labels struct {
	From ValueStep `json:"from"`
}

// Type implements Step
func (s *Labels) Type() quad.IRI {
	return prefix + "Labels"
}

// BuildIterator implements Step
func (s *Labels) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Labels) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Labels(), qs), nil
}

// Limit corresponds to .limit()
type Limit struct {
	From  ValueStep `json:"from"`
	Limit int64     `json:"limit"`
}

// Type implements Step
func (s *Limit) Type() quad.IRI {
	return prefix + "Limit"
}

// BuildIterator implements Step
func (s *Limit) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Limit) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Limit(s.Limit), qs), nil
}

// OutPredicates corresponds to .outPredicates()
type OutPredicates struct {
	From ValueStep `json:"from"`
}

// Type implements Step
func (s *OutPredicates) Type() quad.IRI {
	return prefix + "OutPredicates"
}

// BuildIterator implements Step
func (s *OutPredicates) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *OutPredicates) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.OutPredicates(), qs), nil
}

// Save corresponds to .save()
type Save struct {
	From ValueStep `json:"from"`
	Via  ValueStep `json:"via"`
	Tag  string    `json:"tag"`
}

// Type implements Step
func (s *Save) Type() quad.IRI {
	return prefix + "Save"
}

// BuildIterator implements Step
// TODO(iddan): Default tag to Via
func (s *Save) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Save) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	viaIt, err := s.Via.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Save(viaIt.path, s.Tag), qs), nil
}

// SaveInPredicates corresponds to .saveInPredicates()
type SaveInPredicates struct {
	From ValueStep `json:"from"`
	Tag  string    `json:"tag"`
}

// Type implements Step
func (s *SaveInPredicates) Type() quad.IRI {
	return prefix + "SaveInPredicates"
}

// BuildIterator implements Step
func (s *SaveInPredicates) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *SaveInPredicates) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.SavePredicates(true, s.Tag), qs), nil
}

// SaveOptional corresponds to .saveOpt()
type SaveOptional struct {
	From ValueStep `json:"from"`
	Via  ValueStep `json:"via"`
	Tag  string    `json:"tag"`
}

// Type implements Step
func (s *SaveOptional) Type() quad.IRI {
	return prefix + "SaveOptional"
}

// BuildIterator implements Step
func (s *SaveOptional) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *SaveOptional) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	viaIt, err := s.Via.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.SaveOptional(viaIt.path, s.Tag), qs), nil
}

// SaveOptionalReverse corresponds to .saveOptR()
type SaveOptionalReverse struct {
	From ValueStep `json:"from"`
	Via  ValueStep `json:"via"`
	Tag  string    `json:"tag"`
}

// Type implements Step
func (s *SaveOptionalReverse) Type() quad.IRI {
	return prefix + "SaveOptionalReverse"
}

// BuildIterator implements Step
func (s *SaveOptionalReverse) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *SaveOptionalReverse) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	viaIt, err := s.Via.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.SaveOptionalReverse(viaIt.path, s.Tag), qs), nil
}

// SaveOutPredicates corresponds to .saveOutPredicates()
type SaveOutPredicates struct {
	From ValueStep `json:"from"`
	Tag  string    `json:"tag"`
}

// Type implements Step
func (s *SaveOutPredicates) Type() quad.IRI {
	return prefix + "SaveOutPredicates"
}

// BuildIterator implements Step
func (s *SaveOutPredicates) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *SaveOutPredicates) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.SavePredicates(false, s.Tag), qs), nil
}

// SaveReverse corresponds to .saveR()
type SaveReverse struct {
	From ValueStep `json:"from"`
	Via  ValueStep `json:"via"`
	Tag  string    `json:"tag"`
}

// Type implements Step
func (s *SaveReverse) Type() quad.IRI {
	return prefix + "SaveReverse"
}

// BuildIterator implements Step
func (s *SaveReverse) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *SaveReverse) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	viaIt, err := s.Via.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.SaveReverse(viaIt.path, s.Tag), qs), nil
}

// Skip corresponds to .skip()
type Skip struct {
	From   ValueStep `json:"from"`
	Offset int64     `json:"offset"`
}

// Type implements Step
func (s *Skip) Type() quad.IRI {
	return prefix + "Skip"
}

// BuildIterator implements Step
func (s *Skip) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Skip) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Skip(s.Offset), qs), nil
}

// Union corresponds to .union() and .or()
type Union struct {
	From      ValueStep `json:"from"`
	Unionized ValueStep `json:"unionized"`
}

// Type implements Step
func (s *Union) Type() quad.IRI {
	return prefix + "Union"
}

// BuildIterator implements Step
func (s *Union) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Union) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	unionizedIt, err := s.Unionized.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Or(unionizedIt.path), qs), nil
}

// Unique corresponds to .unique()
type Unique struct {
	From ValueStep `json:"from"`
}

// Type implements Step
func (s *Unique) Type() quad.IRI {
	return prefix + "Unique"
}

// BuildIterator implements Step
func (s *Unique) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Unique) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Unique(), qs), nil
}

// Order corresponds to .order()
type Order struct {
	From ValueStep `json:"from"`
}

// Type implements Step
func (s *Order) Type() quad.IRI {
	return prefix + "Order"
}

// BuildIterator implements Step
func (s *Order) BuildIterator(qs graph.QuadStore) (query.Iterator, error) {
	return s.BuildValueIterator(qs)
}

// BuildValueIterator implements ValueStep
func (s *Order) BuildValueIterator(qs graph.QuadStore) (*ValueIterator, error) {
	fromIt, err := s.From.BuildValueIterator(qs)
	if err != nil {
		return nil, err
	}
	return NewValueIterator(fromIt.path.Order(), qs), nil
}
