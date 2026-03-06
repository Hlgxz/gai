package orm

// Scope is a reusable query constraint that can be applied to a QueryBuilder.
// Usage: Query[User](db).Scope(ActiveUsers).Get()
type Scope func(q *QueryBuilder) *QueryBuilder

// ActiveScope is a built-in scope that filters soft-deleted records.
func ActiveScope(q *QueryBuilder) *QueryBuilder {
	return q.WhereNull("deleted_at")
}
