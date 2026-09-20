package admin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type SetBucketSyncRequest struct {
	Name    string
	Tenant  string
	Enabled *bool
}

// SetBucketSync enables or disables bucket synchronization through
// PUT /admin/bucket?sync. Ceph defaults Enabled to true when omitted.
func (client *Client) SetBucketSync(ctx context.Context, input SetBucketSyncRequest) error {
	if ctx == nil {
		return errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("admin: bucket name must not be empty")
	}
	query := url.Values{
		"bucket": {input.Name},
		"sync":   {""},
	}
	setString(query, "tenant", input.Tenant)
	setBool(query, "sync", input.Enabled)
	return client.bucketAction(ctx, http.MethodPut, query)
}
