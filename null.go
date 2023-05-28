package sqlz

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Null represents a nullable type of T.
// The zero value for Null[T] is its null state (Valid is false).
type Null[T any] struct {
	Some  T    // the actual value
	Valid bool // Valid is true if this value is not null
}

// NewNull returns a new nullable type of T, initialized with the given value.
func NewNull[T any](value T) Null[T] {
	return Null[T]{
		Some:  value,
		Valid: true,
	}
}

// Set sets the value.
func (n *Null[T]) Set(value T) {
	n.Some, n.Valid = value, true
}

// Invalidate sets n to its null value.
func (n *Null[T]) Invalidate() {
	var zero T
	n.Some, n.Valid = zero, false
}

// Scan implements [sql.Scanner].
func (n *Null[T]) Scan(value any) error {
	if value == nil {
		n.Invalidate()
		return nil
	}
	var ptr any = &n.Some
	if scanner, ok := ptr.(sql.Scanner); ok {
		// *T implements Scanner, use this to scan the value.
		if err := scanner.Scan(value); err != nil {
			return err
		}
		n.Valid = true
	} else if n.Some, n.Valid = value.(T); !n.Valid {
		return fmt.Errorf("sqlz.Null: converting value type %T to %T is unsupported", value, n.Some)
	}
	return nil
}

// Value implements [driver.Valuer].
func (n Null[T]) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	var some any = n.Some
	if valuer, ok := some.(driver.Valuer); ok {
		return valuer.Value()
	}
	return some, nil
}

// Ptr returns a pointer to the value if valid, otherwise nil.
func (n Null[T]) Ptr() *T {
	if !n.Valid {
		return nil
	}
	return &n.Some
}

var nullBytes = []byte("null")

// MarshalJSON implements [json.Marshaler].
func (n Null[T]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return nullBytes, nil
	}
	return json.Marshal(n.Some)
}

// UnmarshalJSON implements [json.Unmarshaler].
func (n *Null[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, nullBytes) {
		n.Invalidate()
		return nil
	}
	if err := json.Unmarshal(data, &n.Some); err != nil {
		return fmt.Errorf("sqlz.Null: could not unmarshal type %T: %w", n.Some, err)
	}
	n.Valid = true
	return nil
}

func (n Null[T]) String() string {
	if n.Valid {
		return fmt.Sprintf("sqlz.Null[%T]{%[1]v}", n.Some)
	}
	return fmt.Sprintf("sqlz.Null[%T]{<null>}", n.Some)
}
