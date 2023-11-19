package sqlz_test

import (
	"context"
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
			t.Error("erecordpected embedded pointer panic")
		}
	}()

	err := sqlz.Scan(context.Background(), rows, &record)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
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
	for range records {
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
