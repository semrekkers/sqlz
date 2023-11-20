package sqlz

import (
	"maps"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

type structFieldIndex map[string][]uint16

type cache struct {
	types atomic.Pointer[map[reflect.Type]structFieldIndex]
	mu    sync.Mutex
}

func (c *cache) load() (x map[reflect.Type]structFieldIndex) {
	if ptr := c.types.Load(); ptr != nil {
		x = *ptr
	}
	return
}

func (c *cache) getStructFieldIndex(t reflect.Type) structFieldIndex {
	if x, ok := c.load()[t]; ok {
		return x // fast path
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	types := c.load()
	if x, ok := types[t]; ok {
		return x
	} else if types != nil {
		types = maps.Clone(types)
	} else {
		types = make(map[reflect.Type]structFieldIndex, 1)
	}
	x := make(structFieldIndex, t.NumField())
	fillStructFieldIndex(x, t, nil, "")
	types[t] = x
	c.types.Store(&types)
	return x
}

func (c *cache) purge() {
	c.mu.Lock()
	c.types.Store(nil)
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
