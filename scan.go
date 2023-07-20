// Package sqlz provides a set of helper functions and types to simplify operations with SQL databases in Go.
// It provides a more flexible and intuitive interface for scanning SQL query results directly into Go structs,
// slices of structs, or channels of structs.
package sqlz

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Rows represents the result set of a database query.
// It's implemented by [sql.Rows].
type Rows interface {
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(dest ...any) error
}

// Scan is for scanning the result set from a SQL query into a destination structure.
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
func Scan(ctx context.Context, rows Rows, dest any) error {
	destValue := reflect.ValueOf(dest)
	if kind := destValue.Kind(); kind == reflect.Chan {
		return scanChan(ctx, destValue, rows)
	} else if kind != reflect.Pointer {
		panic("dest must be a pointer or chan")
	}
	elemValue := destValue.Elem()
	switch elemValue.Kind() {
	case reflect.Struct:
		scanValues, err := mapToScanValues(elemValue, rows)
		if err != nil {
			return err
		}
		if !rows.Next() {
			if err = rows.Err(); err != nil {
				return err
			}
			return sql.ErrNoRows
		}
		return rows.Scan(scanValues...)

	case reflect.Slice:
		return scanSlice(destValue, rows)

	default:
		panic("dest must point to a struct or slice")
	}
}

func scanSlice(slicePtr reflect.Value, rows Rows) error {
	slice := slicePtr.Elem()
	elemType := slice.Type().Elem()
	isPtrElem := elemType.Kind() == reflect.Pointer
	if isPtrElem {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		panic("dest slice of non-struct elements")
	}
	scratch := reflect.New(elemType).Elem()
	scanValues, err := mapToScanValues(scratch, rows)
	if err != nil {
		return err
	}
	for rows.Next() {
		if err := rows.Scan(scanValues...); err != nil {
			return err
		}
		newElem := scratch
		if isPtrElem {
			newElem = reflect.New(elemType)
			newElem.Elem().Set(scratch)
		}
		slice = reflect.Append(slice, newElem)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	slicePtr.Elem().Set(slice)
	return nil
}

func scanChan(ctx context.Context, dest reflect.Value, rows Rows) error {
	elemType := dest.Type().Elem()
	isPtrElem := elemType.Kind() == reflect.Pointer
	if isPtrElem {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		panic("dest chan of non-struct elements")
	}
	scratch := reflect.New(elemType).Elem()
	scanValues, err := mapToScanValues(scratch, rows)
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
		if err := rows.Scan(scanValues...); err != nil {
			return err
		}
		newElem := scratch
		if isPtrElem {
			newElem = reflect.New(elemType)
			newElem.Elem().Set(scratch)
		}
		selectOps[0].Send = newElem
		if chosen, _, _ := reflect.Select(selectOps); chosen == 1 {
			// select on ctx.Done()
			return ctx.Err()
		}
	}
	return rows.Err()
}

func mapToScanValues(dest reflect.Value, rows Rows) ([]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	fieldIndex := getFieldIndex(dest.Type())
	scanValues := make([]any, len(columns))
	for i, column := range columns {
		x, ok := fieldIndex[column]
		if !ok {
			return nil, fmt.Errorf("sqlz: missing field mapping for column %q", column)
		}
		scanValues[i] = fieldByIndex(dest, x).Addr().Interface()
	}
	return scanValues, nil
}

// fieldByIndex has the same functionality as [reflect.Value.FieldByIndex] but uses uint16's as indexes.
func fieldByIndex(v reflect.Value, index []uint16) reflect.Value {
	if len(index) == 1 {
		return v.Field(int(index[0]))
	}
	for _, x := range index {
		v = v.Field(int(x))
	}
	return v
}

type typeFieldIndex = map[string][]uint16

var (
	fieldIndexMu    sync.Mutex
	fieldIndexCache = map[reflect.Type]typeFieldIndex{}
)

func getFieldIndex(t reflect.Type) typeFieldIndex {
	fieldIndexMu.Lock()
	defer fieldIndexMu.Unlock()
	fieldIndex, ok := fieldIndexCache[t]
	if !ok {
		fieldIndex = make(typeFieldIndex, t.NumField())
		fieldIndexFromStruct(t, nil, "", fieldIndex)
		fieldIndexCache[t] = fieldIndex
	}
	return fieldIndex
}

func fieldIndexFromStruct(t reflect.Type, cursor []uint16, prefix string, fieldIndex typeFieldIndex) {
	numField := t.NumField()
	for i := 0; i < numField; i++ {
		field := t.Field(i)
		fieldName := field.Tag.Get("db")
		if fieldName == "-" {
			continue // skip
		}
		if field.Anonymous {
			if kind := field.Type.Kind(); kind == reflect.Pointer {
				panic("cannot use embedded pointer in struct")
			} else if kind == reflect.Struct {
				// traverse embedded struct field
				fieldIndexFromStruct(field.Type, append(cursor, uint16(i)), fieldName, fieldIndex)
			}
			continue // next
		}
		if !field.IsExported() {
			continue // skip
		}
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}
		index := make([]uint16, len(cursor)+1)
		copy(index, cursor)
		index[len(cursor)] = uint16(i) // it's unlikely that a struct has more than 65536 fields.
		fieldIndex[prefix+fieldName] = index
	}
}

// PurgeCache purges internal caches.
func PurgeCache() {
	fieldIndexMu.Lock()
	fieldIndexCache = map[reflect.Type]typeFieldIndex{}
	fieldIndexMu.Unlock()
}
