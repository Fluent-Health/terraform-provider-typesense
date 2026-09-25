package typesense

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// SchemaChange is one in-progress collection alter, as reported by
// GET /operations/schema_changes. The server lists only collections whose
// alter is still running.
type SchemaChange struct {
	Collection    string `json:"collection"`
	ValidatedDocs int64  `json:"validated_docs"`
	AlteredDocs   int64  `json:"altered_docs"`
}

// GetSchemaChanges lists the collection alters currently running.
// GET /operations/schema_changes — https://typesense.org/docs/30.2/api/collections.html#update-or-alter-a-collection
func (c *Client) GetSchemaChanges(ctx context.Context) ([]SchemaChange, error) {
	var out []SchemaChange
	if err := c.do(ctx, "GET", "/operations/schema_changes", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// requiredIdlePolls is how many consecutive polls must report no alter for a
// collection before WaitForSchemaChange returns. In a multi-node cluster behind
// a load balancer each poll can land on a different node, and a node that has
// finished applying the alter reports nothing while another is still running.
const requiredIdlePolls = 3

// WaitForSchemaChange blocks until no alter for the named collection is
// reported for requiredIdlePolls consecutive polls, the client's schema-change
// timeout elapses, or ctx is done. Poll errors are tolerated (the server can be
// busy or briefly unreachable mid-alter) until the deadline.
func (c *Client) WaitForSchemaChange(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, c.schemaChangeTimeout)
	defer cancel()

	idle := 0
	var lastErr error
	for {
		changes, err := c.GetSchemaChanges(ctx)
		switch {
		case err != nil && ctx.Err() != nil:
			// The deadline cut this poll short; report on the state seen so far.
		case err != nil:
			lastErr = err
			idle = 0
		case hasSchemaChange(changes, name):
			lastErr = nil
			idle = 0
		default:
			lastErr = nil
			idle++
			if idle >= requiredIdlePolls {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("schema change on collection %q still unconfirmed after %s (last poll error: %v): %w", name, c.schemaChangeTimeout, lastErr, ctx.Err())
			}
			return fmt.Errorf("schema change on collection %q still running after %s: %w", name, c.schemaChangeTimeout, ctx.Err())
		case <-time.After(c.pollInterval):
		}
	}
}

func hasSchemaChange(changes []SchemaChange, name string) bool {
	for _, ch := range changes {
		if ch.Collection == name {
			return true
		}
	}
	return false
}

// IsRequestOutlived reports whether err means the HTTP request ended before the
// server answered: a gateway or proxy timeout (504, 408), a proxy that lost the
// backend connection (502), a client-side timeout, or a dropped connection. For
// a collection PATCH this does NOT mean the alter failed — Typesense keeps
// running it after the connection is gone.
func IsRequestOutlived(err error) bool {
	if err == nil {
		return false
	}
	var ae *APIError
	if errors.As(err, &ae) {
		switch ae.StatusCode {
		case http.StatusGatewayTimeout, http.StatusRequestTimeout, http.StatusBadGateway:
			return true
		}
		return false
	}
	// The caller's own context ending is a cancellation, not an outlived request.
	if errors.Is(err, context.Canceled) {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	return errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.EOF)
}

// IsAlterInProgress reports whether err is Typesense's 422 for a PATCH that
// arrived while another alter on the same collection was still running.
func IsAlterInProgress(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.StatusCode == http.StatusUnprocessableEntity &&
		strings.Contains(ae.Message, "Another collection update operation is in progress")
}
