package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
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
func NewConnection(databaseURL string, maxConns, maxIdleConns int) (*Connection, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

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
