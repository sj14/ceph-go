package rgw

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// SetBucketSyncRequest identifies a bucket and requires an explicit sync state.
// Sync must not be nil.
type SetBucketSyncRequest struct {
	Name   string
	Tenant string
	Sync   *bool
}

// SetBucketSync enables or disables bucket synchronization through
// PUT /admin/bucket?sync. Sync must be provided; omission returns a validation
// error before any request is sent.
//
// Verified against Ceph v20.2.4:
//   - src/rgw/driver/rados/rgw_rest_bucket.cc (RGWOp_Sync_Bucket and op_put)
//   - src/rgw/driver/rados/rgw_bucket.cc (RGWBucket::sync)
func (client *Client) SetBucketSync(ctx context.Context, input SetBucketSyncRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}
	if input.Sync == nil {
		return errors.New("rgw: bucket sync must be provided")
	}
	query := url.Values{
		"bucket": {input.Name},
		"sync":   {""},
	}
	setString(query, "tenant", input.Tenant)
	setBool(query, "sync", input.Sync)
	return client.bucketAction(ctx, http.MethodPut, query)
}
