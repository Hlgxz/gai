package orm

// Scope is a reusable query constraint that can be applied to a QueryBuilder.
// Usage: Query[User](db).Scope(ActiveUsers).Get()
type Scope func(q *QueryBuilder) *QueryBuilder

// ActiveScope filters soft-deleted records. It is a no-op when the
// QueryBuilder already has automatic soft-delete filtering enabled
// (which is the default for models with a DeletedAt field).
// Use this scope only on builders created via Table() or after
// calling WithTrashed().
func ActiveScope(q *QueryBuilder) *QueryBuilder {
	if q.softDelete {
		return q
	}
	return q.WhereNull("deleted_at")
}
