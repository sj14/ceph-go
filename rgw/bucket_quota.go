package rgw

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// SetBucketQuotaRequest updates a named bucket's quota. MaxSize is in bytes;
// nil settings preserve stored values, and negative limits are unlimited.
type SetBucketQuotaRequest struct {
	UID        string
	Name       string
	MaxObjects *int64
	MaxSize    *int64
	Enabled    *bool
}

// SetBucketQuota updates a bucket quota through PUT /admin/bucket?quota.
// Verified against Ceph v20.2.4's src/rgw/driver/rados/rgw_rest_bucket.cc
// (RGWOp_Set_Bucket_Quota): max-size is parsed as an int64 byte count.
func (client *Client) SetBucketQuota(ctx context.Context, input SetBucketQuotaRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}
	query := url.Values{
		"bucket": {input.Name},
		"quota":  {""},
		"uid":    {input.UID},
	}
	setInt64(query, "max-objects", input.MaxObjects)
	setInt64(query, "max-size", input.MaxSize)
	setBool(query, "enabled", input.Enabled)
	return client.bucketAction(ctx, http.MethodPut, query)
}
