// Package pgvector wraps PostgreSQL + pgvector extension for semantic vector search.
// Replaces VikingDB in the design doc (面试时讲 VikingDB 架构，实际跑 pgvector)。
package pgvector

// Client wraps pgxpool + pgvector helpers.
type Client struct {
	dsn string
	// pool *pgxpool.Pool  // TODO Phase 1
}

// NewClient connects to Postgres at dsn.
func NewClient(dsn string) (*Client, error) {
	// TODO Phase 1: pgxpool.New
	return &Client{dsn: dsn}, nil
}
