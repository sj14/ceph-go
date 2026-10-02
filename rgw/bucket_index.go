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
	// Objects is present only when CheckObjects is enabled. Ceph
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
	// Fix removes invalid multipart index entries and rebuilds index statistics.
	// False reports inconsistencies without repairing them.
	Fix bool
	// CheckObjects also reconciles object index entries against stored objects.
	// It requires Fix=true; false skips that scan.
	CheckObjects bool
}

// CheckBucketIndex checks a bucket index through GET /admin/bucket?index.
// Fix opts into removing invalid multipart index entries and rebuilding index
// statistics. CheckObjects additionally reconciles object entries and
// requires Fix. Both default to false. The report's existing/calculated
// headers are captured before rebuilding.
// Ceph requires the buckets=write capability even without repair.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_bucket.cc (RGWOp_Check_Bucket_Index)
//   - src/rgw/driver/rados/rgw_bucket.cc (RGWBucketAdminOp::check_index,
//     RGWBucket::check_bad_index_multipart/check_object_index/check_index)
func (client *Client) CheckBucketIndex(ctx context.Context, input CheckBucketIndexRequest) (BucketIndexCheck, error) {
	if ctx == nil {
		return BucketIndexCheck{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return BucketIndexCheck{}, errors.New("rgw: bucket name must not be empty")
	}
	if input.CheckObjects && !input.Fix {
		return BucketIndexCheck{}, errors.New("rgw: object reconciliation requires index repair")
	}
	query := url.Values{
		"bucket": {input.Name},
		"index":  {""},
	}
	setBool(query, "fix", &input.Fix)
	setBool(query, "check-objects", &input.CheckObjects)
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
