package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	dbPingMaxRetries = 5
	dbPingRetryDelay = 3 * time.Second
)

// Connection represents a database connection
type Connection struct {
	DB *sql.DB
}

// NewConnection creates a new database connection
//
// Usa el driver pgx en modo "simple protocol" (sin prepared statements). Esto es
// imprescindible cuando se conecta a través del pooler de Supabase en modo
// transacción (puerto 6543): ese modo multiplexa conexiones de servidor por
// transacción, lo que parte el Parse/Bind del protocolo extendido entre conexiones
// distintas y produce errores "bind message supplies N parameters, but prepared
// statement \"\" requires M". El simple protocol envía la query como texto en un
// único mensaje, eliminando ese fallo.
func NewConnection(databaseURL string, maxConns, maxIdleConns int) (*Connection, error) {
	connConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database url: %w", err)
	}
	connConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	db := stdlib.OpenDB(*connConfig)

	// Set connection pool settings
	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection with retries to handle transient pool exhaustion during rolling deploys
	var pingErr error
	for i := 0; i < dbPingMaxRetries; i++ {
		if pingErr = db.Ping(); pingErr == nil {
			break
		}
		if i < dbPingMaxRetries-1 {
			time.Sleep(dbPingRetryDelay)
		}
	}
	if pingErr != nil {
		return nil, fmt.Errorf("failed to ping database: %w", pingErr)
	}

	return &Connection{DB: db}, nil
}

// Close closes the database connection
func (c *Connection) Close() error {
	return c.DB.Close()
}

// Health checks the database health
func (c *Connection) Health() error {
	return c.DB.Ping()
}
