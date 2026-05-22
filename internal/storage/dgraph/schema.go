package dgraph

import (
	"context"
	"fmt"

	"github.com/dgraph-io/dgo/v240/protos/api"
)

// InitializeSchema injects SchemaDQL (generated from internal/schema/schema.yaml)
// into the Dgraph cluster via the Alter API. Idempotent — safe to call on every boot.
func (c *Client) InitializeSchema(ctx context.Context) error {
	op := &api.Operation{Schema: SchemaDQL}
	if err := c.dg.Alter(ctx, op); err != nil {
		return fmt.Errorf("dgraph.InitializeSchema: %w", err)
	}
	return nil
}

// DropAll wipes the entire Dgraph dataset (including schema). DEV ONLY.
func (c *Client) DropAll(ctx context.Context) error {
	op := &api.Operation{DropAll: true}
	if err := c.dg.Alter(ctx, op); err != nil {
		return fmt.Errorf("dgraph.DropAll: %w", err)
	}
	return nil
}
