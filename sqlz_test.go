package sqlz_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/semrekkers/sqlz"
	"github.com/semrekkers/sqlz/internal/scantest"
)

type testStructBase struct {
	ID          int
	Username    string
	DisplayName string `db:"display_name"`
	Email       string
	Age         int
	Password    []byte `db:"-"`
	IsAdmin     bool   `db:"is_admin"`

	sessionKey []byte
}

type testStruct struct {
	testStructBase
	CreatedAt time.Time `db:"created_at"`
}

var fixedTestStruct = testStruct{
	testStructBase{
		ID:          1146,
		Username:    "john_doe",
		DisplayName: "John Doe",
		Email:       "john@example.com",
		Age:         42,
		IsAdmin:     false,
	},
	time.Date(2023, 10, 10, 13, 14, 21, 0, time.UTC),
}

func TestScanStruct(t *testing.T) {
	var (
		rows   = scantest.NewRows(1)
		record testStruct
	)

	err := sqlz.Scan(context.Background(), rows, &record)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
	if !reflect.DeepEqual(record, fixedTestStruct) {
		t.Errorf("record %v != fixedTestStruct", record)
	}
}

func TestScanMissingField(t *testing.T) {
	var (
		rows   = scantest.NewRows(1)
		record testStructBase
	)

	err := sqlz.Scan(context.Background(), rows, &record)

	if err.Error() != `sqlz: missing field mapping for column "created_at"` {
		t.Errorf("err{%s} != `sqlz: missing field mapping ...`", err)
	}
}

func TestScanIgnoreUnknownColumns(t *testing.T) {
	var (
		sc     = sqlz.Scanner{IgnoreUnknownColumns: true}
		rows   = scantest.NewRows(1)
		record testStructBase
	)

	err := sc.Scan(context.Background(), rows, &record)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
}

func TestEmbeddedPointerField(t *testing.T) {
	var (
		rows   = scantest.NewRows(1)
		record struct {
			*testStruct
		}
	)

	defer func() {
		msg := recover().(string)
		if msg != "cannot use embedded pointer in struct" {
			t.Error("expected embedded pointer panic")
		}
	}()

	err := sqlz.Scan(context.Background(), rows, &record)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
}

func TestPointerField(t *testing.T) {
	var (
		sc     = sqlz.Scanner{IgnoreUnknownColumns: true}
		rows   = scantest.NewRows(1)
		record struct {
			Row *testStruct
		}
	)

	// The Row field of record should be silently ignored.
	err := sc.Scan(context.Background(), rows, &record)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
}

func TestScanSmallStruct(t *testing.T) {
	var (
		sc     = sqlz.Scanner{IgnoreUnknownColumns: true}
		rows   = scantest.NewRows(1)
		record struct {
			Age int
		}
	)

	err := sc.Scan(context.Background(), rows, &record)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
	if record.Age != 42 {
		t.Errorf("record.Age{%d} != 42", record.Age)
	}
}

func TestEmbeddedInterfaceField(t *testing.T) {
	var (
		rows   = scantest.NewRows(1)
		record struct {
			testStruct
			sqlz.Rows // any interface will do
		}
	)

	err := sqlz.Scan(context.Background(), rows, &record)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
	if !reflect.DeepEqual(record.testStruct, fixedTestStruct) {
		t.Errorf("record %v != fixedTestStruct", record)
	}
}

func TestScanSlice(t *testing.T) {
	var (
		rows    = scantest.NewRows(4)
		records []*testStruct
	)

	err := sqlz.Scan(context.Background(), rows, &records)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
	if len(records) != 4 {
		t.Errorf("len(records){%d} != 4", len(records))
	}
	for i, rec := range records {
		if !reflect.DeepEqual(*rec, fixedTestStruct) {
			t.Errorf("record[%d] %v != fixedTestStruct", i, *rec)
		}
	}
}

func TestScanChan(t *testing.T) {
	var (
		rows    = scantest.NewRows(4)
		records = make(chan *testStruct, 50)
	)

	err := sqlz.Scan(context.Background(), rows, records)
	close(records)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
	var recordCount int
	for v := range records {
		if !reflect.DeepEqual(*v, fixedTestStruct) {
			t.Errorf("record[%d] %v != fixedTestStruct", recordCount, *v)
		}
		recordCount++
	}
	if recordCount != 4 {
		t.Errorf("recordCount{%d} != 4", recordCount)
	}
}

func TestScanChanCanceled(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		rows        = scantest.NewRows(1)
		records     = make(chan *testStruct)
	)
	cancel()

	err := sqlz.Scan(ctx, rows, records)

	if err != context.Canceled {
		t.Errorf("err{%s} != context.Canceled", err)
	}
}

func BenchmarkScanStruct(b *testing.B) {
	var (
		sc sqlz.Scanner
	)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(p *testing.PB) {
		rows := scantest.NewRows(1_000_000_000)
		for p.Next() {
			var record testStruct
			if err := sc.Scan(context.Background(), rows, &record); err != nil {
				b.Error(err)
			}
		}
	})
}

func BenchmarkScanSlice(b *testing.B) {
	var (
		sc sqlz.Scanner
	)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(p *testing.PB) {
		records := make([]testStruct, 0, 30)
		for p.Next() {
			rows := scantest.NewRows(30)
			if err := sc.Scan(context.Background(), rows, &records); err != nil {
				b.Error(err)
			}
			records = records[:0]
		}
	})
}

func BenchmarkScanChan(b *testing.B) {
	var (
		sc sqlz.Scanner
		ch = make(chan *testStruct, 30)
	)
	go func() {
		for range ch {
			// drain
		}
	}()
	b.Cleanup(func() {
		close(ch)
	})
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			rows := scantest.NewRows(30)
			if err := sc.Scan(context.Background(), rows, ch); err != nil {
				b.Error(err)
			}
		}
	})
}
