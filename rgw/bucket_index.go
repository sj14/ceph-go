package rgw

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
	InvalidMultipartEntries []string `json:"invalid_multipart_entries"`
	// Objects is present only when RepairBucketIndex checks objects. Ceph
	// emits an object containing repeated "object" keys, so retain its JSON
	// rather than decoding into a map that would discard entries.
	Objects json.RawMessage        `json:"objects,omitempty"`
	Result  BucketIndexCheckResult `json:"check_result"`
}

type BucketIndexCheckResult struct {
	Existing   BucketIndexHeader `json:"existing_header"`
	Calculated BucketIndexHeader `json:"calculated_header"`
}

type BucketIndexHeader struct {
	Usage map[string]StorageStats `json:"usage"`
}

type CheckBucketIndexRequest struct {
	Name string
}

// CheckBucketIndex checks a bucket index without repairing it through
// GET /admin/bucket?index&fix=false&check-objects=false.
// Ceph requires the buckets=write capability even for this check.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_bucket.cc (RGWOp_Check_Bucket_Index)
//   - src/rgw/driver/rados/rgw_bucket.cc (RGWBucketAdminOp::check_index)
func (client *Client) CheckBucketIndex(ctx context.Context, input CheckBucketIndexRequest) (BucketIndexCheck, error) {
	return client.checkBucketIndex(ctx, input.Name, false, false)
}

// RepairBucketIndexRequest selects a bucket to repair. CheckObjects also
// reconciles object index entries against stored objects; false skips that scan.
type RepairBucketIndexRequest struct {
	Name         string
	CheckObjects bool
}

// RepairBucketIndex checks and repairs a bucket index through
// GET /admin/bucket?index&fix=true. It removes invalid multipart index entries
// and rebuilds index statistics, optionally reconciling object entries.
// It requires the buckets=write capability and returns Ceph's check report.
// The report's existing/calculated headers are captured before rebuilding.
func (client *Client) RepairBucketIndex(ctx context.Context, input RepairBucketIndexRequest) (BucketIndexCheck, error) {
	return client.checkBucketIndex(ctx, input.Name, true, input.CheckObjects)
}

func (client *Client) checkBucketIndex(ctx context.Context, name string, fix, checkObjects bool) (BucketIndexCheck, error) {
	if ctx == nil {
		return BucketIndexCheck{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(name) == "" {
		return BucketIndexCheck{}, errors.New("rgw: bucket name must not be empty")
	}
	query := url.Values{
		"bucket": {name},
		"index":  {""},
	}
	setBool(query, "fix", &fix)
	setBool(query, "check-objects", &checkObjects)
	body, request, err := client.bucketRequest(ctx, http.MethodGet, query)
	if err != nil {
		return BucketIndexCheck{}, err
	}
	var result BucketIndexCheck
	if err := json.Unmarshal(body, &result); err != nil {
		return BucketIndexCheck{}, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return result, nil
}
