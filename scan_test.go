package sqlz_test

//go:generate go run github.com/golang/mock/mockgen --source scan.go --package mocks --destination internal/mocks/mocks.go

import (
	"context"
	"testing"
	"time"

	"github.com/semrekkers/sqlz"
	"github.com/semrekkers/sqlz/internal/mocks"

	"github.com/golang/mock/gomock"
)

type testStruct struct {
	ID        int
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string
	Password  []byte `db:"-"`

	sessionKey []byte
}

type testStructRecord struct {
	testStruct
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (r *testStructRecord) fields() []any {
	return []any{
		&r.ID,
		&r.FirstName,
		&r.LastName,
		&r.Email,
		&r.CreatedAt,
		&r.UpdatedAt,
	}
}

var testStructRecordFields = []string{"id", "first_name", "last_name", "email", "created_at", "updated_at"}

func TestScanStruct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var record testStructRecord

	rows := mocks.NewMockRows(ctrl)
	rows.EXPECT().
		Columns().
		Times(1).
		Return(testStructRecordFields, nil)

	rows.EXPECT().
		Next().
		Times(1).
		Return(true)

	rows.EXPECT().
		Scan(record.fields()...).
		Times(1).
		Return(nil)

	err := sqlz.Scan(context.Background(), rows, &record)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
}

func TestScanMissingField(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var record testStructRecord

	rows := mocks.NewMockRows(ctrl)
	rows.EXPECT().
		Columns().
		Times(1).
		Return([]string{"order_id"}, nil)

	err := sqlz.Scan(context.Background(), rows, &record)

	if err.Error() != `sqlz: missing field mapping for column "order_id"` {
		t.Errorf("err{%s} != `sqlz: missing field mapping ...`", err)
	}
}

func TestScanSlice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		rowsLeft = 4
		autoIncr = 0
		records  []*testStructRecord
	)

	rows := mocks.NewMockRows(ctrl)
	rows.EXPECT().
		Columns().
		Times(1).
		Return(testStructRecordFields, nil)

	rows.EXPECT().
		Next().
		Times(rowsLeft + 1).
		DoAndReturn(func() (b bool) {
			b = rowsLeft > 0
			rowsLeft--
			return
		})

	rows.EXPECT().
		Scan(gomock.Any()).
		Times(rowsLeft).
		DoAndReturn(func(v []any) error {
			autoIncr++
			idPtr := v[0].(*int)
			*idPtr = autoIncr
			return nil
		})

	rows.EXPECT().
		Err().
		Times(1).
		Return(nil)

	err := sqlz.Scan(context.Background(), rows, &records)

	if err != nil {
		t.Error("sqlz.Scan(...):", err)
	}
	if len(records) != 4 {
		t.Errorf("len(records){%d} != 4", len(records))
	}
	for i, v := range records {
		if expect := i + 1; v.ID != expect {
			t.Errorf("records[%d].ID{%d} != %d", i, v.ID, expect)
		}
	}
}

func TestScanChan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		rowsLeft = 4
		records  = make(chan *testStructRecord, rowsLeft+2)
	)

	rows := mocks.NewMockRows(ctrl)
	rows.EXPECT().
		Columns().
		Times(1).
		Return(testStructRecordFields, nil)

	rows.EXPECT().
		Next().
		Times(rowsLeft + 1).
		DoAndReturn(func() (b bool) {
			b = rowsLeft > 0
			rowsLeft--
			return
		})

	rows.EXPECT().
		Scan(gomock.Any()).
		Times(rowsLeft).
		Return(nil)

	rows.EXPECT().
		Err().
		Times(1).
		Return(nil)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		ctx, cancel = context.WithCancel(context.Background())
		records     = make(chan *testStructRecord)
	)
	cancel()

	rows := mocks.NewMockRows(ctrl)
	rows.EXPECT().
		Columns().
		Times(1).
		Return(testStructRecordFields, nil)

	rows.EXPECT().
		Next().
		Times(1).
		Return(true)

	rows.EXPECT().
		Scan(gomock.Any()).
		Times(1).
		Return(nil)

	err := sqlz.Scan(ctx, rows, records)

	if err != context.Canceled {
		t.Errorf("err{%s} != context.Canceled", err)
	}
}
