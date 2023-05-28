package sqlz_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/semrekkers/sqlz"
)

var (
	ctx context.Context
	db  *sql.DB
)

type User struct {
	ID        int
	Name      string
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Age       int
	DeletedAt sqlz.Null[time.Time] `db:"deleted_at"`

	AppValue []byte `db:"-"`
}

func ExampleScan_struct() {
	row, err := db.QueryContext(ctx, "SELECT * FROM users WHERE id = 123")
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()

	var record User
	if err = sqlz.Scan(ctx, row, &record); err != nil {
		log.Fatal(err)
	}
	log.Println(record)
}

func ExampleScan_slice() {
	rows, err := db.QueryContext(ctx, "SELECT * FROM users ORDER BY id LIMIT 10")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var records []*User
	if err = sqlz.Scan(ctx, rows, &records); err != nil {
		log.Fatal(err)
	}
	log.Println(records)
}

func ExampleScan_chan() {
	records := make(chan *User, 8)

	go func() {
		defer close(records)

		rows, err := db.QueryContext(ctx, "SELECT id FROM users WHERE deleted_at IS NULL ORDER BY id")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		if err = sqlz.Scan(ctx, rows, records); err != nil {
			log.Fatal(err)
		}
	}()

	for user := range records {
		fmt.Println("found active user:", user.ID)
	}
}
