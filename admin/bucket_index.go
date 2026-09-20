package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type BucketIndexCheck struct {
	InvalidMultipartEntries []string               `json:"invalid_multipart_entries"`
	Objects                 json.RawMessage        `json:"objects,omitempty"`
	Result                  BucketIndexCheckResult `json:"check_result"`
}

type BucketIndexCheckResult struct {
	Existing   BucketIndexHeader `json:"existing_header"`
	Calculated BucketIndexHeader `json:"calculated_header"`
}

type BucketIndexHeader struct {
	Usage map[string]StorageStats `json:"usage"`
}

type CheckBucketIndexRequest struct {
	Name         string
	Fix          *bool
	CheckObjects *bool
}

// CheckBucketIndex checks and optionally repairs a bucket index through
// GET /admin/bucket?index. Ceph requires Fix when CheckObjects is true.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_bucket.cc (RGWOp_Check_Bucket_Index)
//   - src/rgw/driver/rados/rgw_bucket.cc (RGWBucketAdminOp::check_index)
func (client *Client) CheckBucketIndex(ctx context.Context, input CheckBucketIndexRequest) (BucketIndexCheck, error) {
	if ctx == nil {
		return BucketIndexCheck{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return BucketIndexCheck{}, errors.New("admin: bucket name must not be empty")
	}
	query := url.Values{
		"bucket": {input.Name},
		"index":  {""},
	}
	setBool(query, "fix", input.Fix)
	setBool(query, "check-objects", input.CheckObjects)
	body, request, err := client.bucketRequest(ctx, http.MethodGet, query)
	if err != nil {
		return BucketIndexCheck{}, err
	}
	var result BucketIndexCheck
	if err := json.Unmarshal(body, &result); err != nil {
		return BucketIndexCheck{}, fmt.Errorf("admin: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return result, nil
}
