package sqlz

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

// A Scanner is for scanning result sets from rows into a destination structure.
// It maintains an internal type cache for mapping struct fields to database columns.
// It's safe for concurrent use by multiple goroutines. The zero value is ready to use.
type Scanner struct {
	tc cache

	// IgnoreUnknownColumns controls whether Scan will return an error if a column in the result set has no corresponding struct field.
	// Default is false (return an error).
	IgnoreUnknownColumns bool
}

// Scan is for scanning the result set from rows into a destination structure.
// It supports scanning into a struct, a slice of structs, or a channel that emits structs.
//
// The destination (dest) must be a pointer to a struct, a pointer to a slice of structs, or a channel of structs.
// If the destination is a channel, Scan will send a struct for each row in the result set until the context is canceled
// or the result set is exhausted.
//
// The structure of the destination struct must match the structure of the result set. The field name or its `db` tag must match the column name.
// The field order does not need to match the column order. If a column has no corresponding struct field, Scan returns an error.
//
// Scan blocks until the context is canceled, the result set is exhausted, or an error occurs.
func (s *Scanner) Scan(ctx context.Context, rows Rows, dest any) error {
	destValue := reflect.ValueOf(dest)
	if kind := destValue.Kind(); kind == reflect.Chan {
		return s.scanChan(ctx, destValue, rows)
	} else if kind != reflect.Pointer {
		panic("dest must be a pointer or chan")
	}
	elemValue := destValue.Elem()
	switch elemValue.Kind() {
	case reflect.Struct:
		destValues, err := s.mapFieldDest(elemValue, rows)
		if err != nil {
			return err
		}
		if !rows.Next() {
			if err = rows.Err(); err != nil {
				return err
			}
			return sql.ErrNoRows
		}
		return rows.Scan(destValues...)

	case reflect.Slice:
		return s.scanSlice(destValue, rows)

	default:
		panic("dest must point to a struct or slice")
	}
}

func (s *Scanner) scanSlice(slicePtr reflect.Value, rows Rows) error {
	slice := slicePtr.Elem()
	elemType := slice.Type().Elem()
	isPtrElem := elemType.Kind() == reflect.Pointer
	if isPtrElem {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		panic("dest slice of non-struct elements")
	}
	elem := reflect.New(elemType).Elem()
	destValues, err := s.mapFieldDest(elem, rows)
	if err != nil {
		return err
	}
	for rows.Next() {
		if err := rows.Scan(destValues...); err != nil {
			return err
		}
		newElem := elem
		if isPtrElem {
			newElem = reflect.New(elemType)
			newElem.Elem().Set(elem)
		}
		slice = reflect.Append(slice, newElem)
		elem.SetZero()
	}
	if err = rows.Err(); err != nil {
		return err
	}
	slicePtr.Elem().Set(slice)
	return nil
}

func (s *Scanner) scanChan(ctx context.Context, dest reflect.Value, rows Rows) error {
	elemType := dest.Type().Elem()
	isPtrElem := elemType.Kind() == reflect.Pointer
	if isPtrElem {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		panic("dest chan of non-struct elements")
	}
	elem := reflect.New(elemType).Elem()
	destValues, err := s.mapFieldDest(elem, rows)
	if err != nil {
		return err
	}
	selectOps := []reflect.SelectCase{
		{
			Dir:  reflect.SelectSend,
			Chan: dest,
		},
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ctx.Done()),
		},
	}
	for rows.Next() {
		if err := rows.Scan(destValues...); err != nil {
			return err
		}
		newElem := elem
		if isPtrElem {
			newElem = reflect.New(elemType)
			newElem.Elem().Set(elem)
		}
		selectOps[0].Send = newElem
		if chosen, _, _ := reflect.Select(selectOps); chosen == 1 {
			// select on ctx.Done()
			return ctx.Err()
		}
		elem.SetZero()
	}
	return rows.Err()
}

func (s *Scanner) mapFieldDest(dest reflect.Value, rows Rows) ([]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	fieldIndex := s.tc.getStructFieldIndex(dest.Type())
	scanValues := make([]any, len(columns))
	placeholder := new(any)
	for i, column := range columns {
		x, ok := fieldIndex[column]
		if ok {
			scanValues[i] = fieldByIndex(dest, x).Addr().Interface()
		} else if !s.IgnoreUnknownColumns {
			return nil, fmt.Errorf("sqlz: missing field mapping for column %q", column)
		} else {
			scanValues[i] = placeholder
		}
	}
	return scanValues, nil
}

// PurgeCache purges the internal type cache.
func (s *Scanner) PurgeCache() {
	s.tc.purge()
}

// fieldByIndex has the same functionality as [reflect.Value.FieldByIndex] but uses uint16's as indexes.
func fieldByIndex(v reflect.Value, index []uint16) reflect.Value {
	for _, i := range index {
		v = v.Field(int(i))
	}
	return v
}
