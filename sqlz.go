// Package sqlz provides a set of helper functions and types to simplify operations with SQL databases in Go.
// It provides a more flexible and intuitive interface for scanning SQL query results directly into Go structs,
// slices of structs, or channels of structs.
package sqlz

import "context"

// Rows represents the result set of a database query.
// It's implemented by [sql.Rows].
type Rows interface {
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(dest ...any) error
}

var global Scanner

// Scan is for scanning the result set from rows into a destination structure.
// It uses the global Scanner. See [Scanner.Scan] for more details.
func Scan(ctx context.Context, rows Rows, dest any) error {
	return global.Scan(ctx, rows, dest)
}

// PurgeCache purges the internal type cache of the global Scanner.
//
// Deprecated: This is a no-op, use a dedicated [Scanner] instead.
func PurgeCache() {
	// no-op
}
