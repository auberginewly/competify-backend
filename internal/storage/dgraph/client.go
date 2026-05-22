// Package dgraph wraps the Dgraph gRPC client (dgo v240).
// All ontology entities (Competitor / Product / Feature / ...) persist here.
package dgraph

import (
	"context"
	"fmt"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps dgo.Dgraph with a kept grpc.ClientConn so callers can Close.
type Client struct {
	addr string
	conn *grpc.ClientConn
	dg   *dgo.Dgraph
}

// NewClient dials Dgraph Alpha at addr (e.g. "localhost:9080" for the gRPC port).
// The connection is plaintext (insecure); production should use TLS.
func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dgraph.NewClient: dial %s: %w", addr, err)
	}
	return &Client{
		addr: addr,
		conn: conn,
		dg:   dgo.NewDgraphClient(api.NewDgraphClient(conn)),
	}, nil
}

// Close releases the gRPC connection. Idempotent.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

// HealthCheck pings Alpha by issuing a trivial query.
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.dg.NewReadOnlyTxn().Query(ctx, `{ q(func: type(Competitor), first: 1) { uid } }`)
	if err != nil {
		return fmt.Errorf("dgraph.HealthCheck: %w", err)
	}
	return nil
}

// Dgraph returns the underlying dgo client for advanced use (queries/mutations).
// Most callers should use the typed methods in query.go / schema.go instead.
func (c *Client) Dgraph() *dgo.Dgraph { return c.dg }
