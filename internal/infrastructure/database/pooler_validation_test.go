package database

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
)

// TestPoolerSimpleProtocol valida contra el pooler transaccional real que:
//  1. los arrays text[] hacen round-trip con simple protocol
//  2. bajo concurrencia NO aparecen errores "bind message supplies N parameters"
//
// Solo corre si DATABASE_URL_TEST está definido.
func TestPoolerSimpleProtocol(t *testing.T) {
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST no definido")
	}

	conn, err := NewConnection(url, 5, 2)
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}
	defer conn.Close()

	// 1) round-trip de text[] usando pq.Array/pq.StringArray (sql.Scanner/driver.Valuer
	//    agnósticos del driver, que parsean el formato texto que devuelve pgx)
	var out pq.StringArray
	in := []string{"rock", "jazz", "80s"}
	if err := conn.DB.QueryRowContext(context.Background(),
		`SELECT $1::text[]`, pq.Array(in)).Scan(&out); err != nil {
		t.Fatalf("array round-trip: %v", err)
	}
	if len(out) != 3 || out[0] != "rock" || out[2] != "80s" {
		t.Fatalf("array mismatch: %v", out)
	}
	t.Logf("array round-trip OK: %v", out)

	// 2) concurrencia con queries de distinta aridad (lo que rompía con lib/pq)
	const goroutines = 40
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			switch n % 3 {
			case 0: // 1 parámetro
				var v int
				if e := conn.DB.QueryRowContext(ctx, `SELECT $1::int`, n).Scan(&v); e != nil {
					errCh <- e
				}
			case 1: // 6 parámetros
				var v int
				if e := conn.DB.QueryRowContext(ctx,
					`SELECT $1::int + $2::int + $3::int + $4::int + $5::int + $6::int`,
					1, 2, 3, 4, 5, n).Scan(&v); e != nil {
					errCh <- e
				}
			case 2: // array + texto
				var a pq.StringArray
				if e := conn.DB.QueryRowContext(ctx,
					`SELECT ARRAY[$1::text, $2::text]`, "a", "b").Scan(&a); e != nil {
					errCh <- e
				}
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	// 3) los errores de Postgres llegan como *pgconn.PgError (no *pq.Error),
	//    de lo que dependen las detecciones de violación de unicidad (23505).
	_, dbErr := conn.DB.ExecContext(context.Background(), `SELECT 1/0`)
	pgErr, ok := dbErr.(*pgconn.PgError) // mismo patrón que favorite/translation repos
	if !ok {
		t.Fatalf("se esperaba *pgconn.PgError (type assertion directo), se obtuvo %T (%v)", dbErr, dbErr)
	}
	t.Logf("error tipado OK: *pgconn.PgError code=%s", pgErr.Code)
	failed := false
	for e := range errCh {
		failed = true
		t.Errorf("query concurrente falló: %v", e)
	}
	if !failed {
		t.Logf("%d queries concurrentes de aridad mixta OK, sin errores bind", goroutines)
	}
}
