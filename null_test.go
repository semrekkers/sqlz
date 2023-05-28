package sqlz_test

//go:generate go run github.com/golang/mock/mockgen --package mocks --destination internal/mocks/scanner_mock.go database/sql Scanner

import (
	"fmt"
	"testing"

	"github.com/semrekkers/sqlz"
	"github.com/semrekkers/sqlz/internal/mocks"

	"github.com/golang/mock/gomock"
)

const testValue = "This is a test value"

func TestNewNull(t *testing.T) {
	v := sqlz.NewNull(testValue)

	if !v.Valid {
		t.Error("v.Valid != true")
	}
	if v.Some != testValue {
		t.Error("v.Some != <testValue>")
	}
}

func TestNullSet(t *testing.T) {
	var v sqlz.Null[string]

	v.Set(testValue)

	if !v.Valid {
		t.Error("v.Valid != true")
	}
	if v.Some != testValue {
		t.Error("v.Some != <testValue>")
	}
}

func TestNullInvalidate(t *testing.T) {
	v := sqlz.NewNull(testValue)

	v.Invalidate()

	if v.Valid {
		t.Error("v.Valid != false")
	}
	if v.Some != "" {
		t.Error("v.Some != <zero>")
	}
}

func TestNullPtr(t *testing.T) {
	var v sqlz.Null[string]

	invalidPtr := v.Ptr()
	v.Set(testValue)
	ptr := v.Ptr()

	if invalidPtr != nil {
		t.Error("invalidPtr != nil")
	}
	if *ptr != testValue {
		t.Error("*v.Ptr() != <testValue>")
	}
}

func TestNullScan(t *testing.T) {
	var v sqlz.Null[string]

	err := v.Scan(testValue)

	if err != nil {
		t.Error("v.Scan(...):", err)
	}
	if !v.Valid {
		t.Error("v.Valid != true")
	}
	if v.Some != testValue {
		t.Error("v.Some != <testValue>")
	}
}

func TestNullScanScanner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	scanner := mocks.NewMockScanner(ctrl)
	scanner.EXPECT().
		Scan(testValue).
		Times(1).
		Return(nil)
	v := sqlz.NewNull(struct {
		*mocks.MockScanner
	}{scanner})

	err := v.Scan(testValue)

	if err != nil {
		t.Error("v.Scan(...):", err)
	}
	if !v.Valid {
		t.Error("v.Valid != true")
	}
}

func TestNullValue(t *testing.T) {
	v := sqlz.NewNull(testValue)

	x, err := v.Value()

	if err != nil {
		t.Error("v.Value():", err)
	}
	if x != testValue {
		t.Error("x != <testValue>")
	}
}

func TestNullMarshalJSON(t *testing.T) {
	tests := []struct {
		input  sqlz.Null[string]
		expect string
	}{
		{sqlz.Null[string]{}, `null`},
		{sqlz.NewNull(""), `""`},
		{sqlz.NewNull("Test value"), `"Test value"`},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("Case_%d", i), func(t *testing.T) {
			out, err := tc.input.MarshalJSON()

			if err != nil {
				t.Error("input.MarshalJSON():", err)
			}
			if got := string(out); got != tc.expect {
				t.Errorf("expected: %q, got: %q", tc.expect, got)
			}
		})
	}
}

func TestNullUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input  string
		expect sqlz.Null[string]
	}{
		{`null`, sqlz.Null[string]{}},
		{`""`, sqlz.NewNull("")},
		{`"Test value"`, sqlz.NewNull("Test value")},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("Case_%d", i), func(t *testing.T) {
			var v sqlz.Null[string]

			err := v.UnmarshalJSON([]byte(tc.input))

			if err != nil {
				t.Error("v.UnmarshalJSON(...):", err)
			}
			if v != tc.expect {
				t.Errorf("expected: %v, got: %v", tc.expect, v)
			}
		})
	}
}

func ExampleNull() {
	var x sqlz.Null[string]
	fmt.Println("Zero value is null:", x)
	x.Set("My value")
	fmt.Println("Now x is set to the value:", x)
	fmt.Println("And x.Valid is true:", x.Valid)
	// Output:
	// Zero value is null: sqlz.Null[string]{<null>}
	// Now x is set to the value: sqlz.Null[string]{My value}
	// And x.Valid is true: true
}
