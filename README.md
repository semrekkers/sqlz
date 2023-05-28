# sqlz

[![Go Reference](https://pkg.go.dev/badge/github.com/semrekkers/sqlz.svg)](https://pkg.go.dev/github.com/semrekkers/sqlz)
[![build](https://github.com/semrekkers/sqlz/actions/workflows/build.yml/badge.svg)](https://github.com/semrekkers/sqlz/actions/workflows/build.yml)

This package provides a set of helper functions and types to simplify operations with SQL databases in Go. It provides a more flexible and intuitive interface for scanning SQL query results directly into Go structs, slices of structs, or channels of structs. It was inspired by [jmoiron/sqlx](https://github.com/jmoiron/sqlx) but it's more lightweight. The core implementation is just the `Scan()` function.

## Examples

Use sqlz to scan a slice of `User` structs from a query result set.

```go
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
```

It also supports channels:

```go
records := make(chan *User, 8)

go func() {
    defer close(records)
    // ... perform query
    if err = sqlz.Scan(ctx, rows, records); err != nil {
        log.Fatal(err)
    }
}()

for user := range records {
    // Receive all users from the concurrent query.
    fmt.Println("found user:", user.ID)
}
```

### License

MIT License
