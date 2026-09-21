package rgw

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type SetBucketQuotaRequest struct {
	UID        string
	Name       string
	MaxObjects *int64
	MaxSize    *int64
	MaxSizeKB  *int64
	Enabled    *bool
}

// SetBucketQuota updates a bucket quota through PUT /admin/bucket?quota.
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
	setInt64(query, "max-size-kb", input.MaxSizeKB)
	setBool(query, "enabled", input.Enabled)
	return client.bucketAction(ctx, http.MethodPut, query)
}
