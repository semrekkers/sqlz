package scantest

import (
	"errors"
	"fmt"
	"time"
)

type Rows struct {
	i, n int
	err  error
}

func NewRows(n int) *Rows {
	return &Rows{
		n: n,
	}
}

var (
	errRowsClosed = errors.New("scantest: Rows are closed")
	errNoRows     = errors.New("scantest: no Rows available")
)

var columnNames = []string{
	"id",           // 1146
	"username",     // "john_doe"
	"display_name", // "John Doe"
	"email",        // "john@example.com"
	"age",          // 42
	"is_admin",     // false
	"created_at",   // fixedTimestamp
}

func (r *Rows) Columns() ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.n == 0 {
		return nil, errNoRows
	}
	return columnNames, nil
}

func (r *Rows) Err() error {
	return r.err
}

func (r *Rows) SetErr(v error) {
	r.err = v
}

func (r *Rows) Close() error {
	if r.err == errRowsClosed {
		return nil
	} else if r.err != nil {
		return r.err
	}
	r.err = errRowsClosed
	return nil
}

func (r *Rows) Next() bool {
	if r.err != nil {
		return false
	}
	if r.i > r.n {
		r.err = errNoRows
		return false
	}
	r.i++
	return r.i <= r.n
}

var fixedTimestamp = time.Date(2023, 10, 10, 13, 14, 21, 0, time.UTC)

func (r *Rows) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if r.i == 0 {
		return errors.New("scantest: Scan called without calling Next")
	}
	if len(dest) != 7 {
		return errors.New("scantest: invalid Scan, need 7 arguments")
	}
	if err := scanv("id", dest[0], 1146); err != nil {
		return err
	}
	if err := scanv("username", dest[1], "john_doe"); err != nil {
		return err
	}
	if err := scanv("display_name", dest[2], "John Doe"); err != nil {
		return err
	}
	if err := scanv("email", dest[3], "john@example.com"); err != nil {
		return err
	}
	if err := scanv("age", dest[4], 42); err != nil {
		return err
	}
	if err := scanv("is_admin", dest[5], false); err != nil {
		return err
	}
	if err := scanv("created_at", dest[6], fixedTimestamp); err != nil {
		return err
	}
	return nil
}

func scanv[T any](name string, dest any, v T) error {
	x, ok := dest.(*T)
	if !ok {
		return fmt.Errorf("scantest: scan %s: cannot convert %T to %T", name, v, dest)
	}
	*x = v
	return nil
}
