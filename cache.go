package sqlz

import (
	"reflect"
	"strings"
	"sync"
)

type structFieldIndex map[string][]uint16

type cache struct {
	types map[reflect.Type]structFieldIndex
	mu    sync.Mutex
}

func (c *cache) getStructFieldIndex(t reflect.Type) structFieldIndex {
	var (
		x  structFieldIndex
		ok bool
	)
	c.mu.Lock()
	defer c.mu.Unlock()
	if x, ok = c.types[t]; !ok {
		if c.types == nil {
			c.types = make(map[reflect.Type]structFieldIndex)
		}
		x = make(structFieldIndex, t.NumField())
		fillStructFieldIndex(x, t, nil, "")
		c.types[t] = x
	}
	return x
}

func (c *cache) purge() {
	c.mu.Lock()
	c.types = nil
	c.mu.Unlock()
}

func fillStructFieldIndex(dest structFieldIndex, t reflect.Type, cursor []uint16, prefix string) {
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
				fillStructFieldIndex(dest, field.Type, append(cursor, uint16(i)), fieldName)
			}
			continue // next
		}
		if !field.IsExported() {
			continue // skip
		}
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}
		p := make([]uint16, len(cursor)+1)
		copy(p, cursor)
		p[len(cursor)] = uint16(i) // it's unlikely that a struct has more than 65536 fields.
		dest[prefix+fieldName] = p
	}
}
