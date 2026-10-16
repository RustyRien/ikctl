package sdk

import "maps"

// Filter is a list filter. Keys are snake_case column names with an optional
// operator suffix: field, field__eq, field__not_eq, field__in, field__like,
// field__not_like, field__is_none, field__any, field__contains_all.
// Relation fields use double underscores, e.g. template__name__like.
// The special key "or" holds a list of filters of which any must match.
type Filter map[string]any

// And returns a new filter containing the keys of f and all others.
// Later filters overwrite earlier keys.
func (f Filter) And(others ...Filter) Filter {
	out := Filter{}
	maps.Copy(out, f)
	for _, o := range others {
		maps.Copy(out, o)
	}
	return out
}

// Eq matches field == value (a list value matches any of the values).
func Eq(field string, value any) Filter { return Filter{field: value} }

// NotEq matches field != value.
func NotEq(field string, value any) Filter { return Filter{field + "__not_eq": value} }

// In matches field in values.
func In(field string, values ...any) Filter { return Filter{field + "__in": values} }

// Like matches a case-insensitive substring.
func Like(field, substr string) Filter { return Filter{field + "__like": substr} }

// NotLike excludes a case-insensitive substring.
func NotLike(field, substr string) Filter { return Filter{field + "__not_like": substr} }

// IsNone matches field IS NULL (true) or IS NOT NULL (false).
func IsNone(field string, isNone bool) Filter { return Filter{field + "__is_none": isNone} }

// Any matches a to-many relation containing any of the ids.
func Any(field string, ids ...string) Filter { return Filter{field + "__any": ids} }

// Labels matches objects that have all the given labels.
func Labels(labels ...string) Filter { return Filter{"labels__contains_all": labels} }

// Or matches objects satisfying any of the filters.
func Or(filters ...Filter) Filter { return Filter{"or": filters} }
