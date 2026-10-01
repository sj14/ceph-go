package rgw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BucketVersioningState is the versioning state returned by Admin Ops.
type BucketVersioningState string

const (
	BucketVersioningOff       BucketVersioningState = "off"
	BucketVersioningEnabled   BucketVersioningState = "enabled"
	BucketVersioningSuspended BucketVersioningState = "suspended"
)

// BucketIndexType identifies the bucket index layout returned by Admin Ops.
type BucketIndexType string

const (
	BucketIndexNormal    BucketIndexType = "Normal"
	BucketIndexIndexless BucketIndexType = "Indexless"
)

// BucketReshardState identifies the current bucket resharding state.
type BucketReshardState string

const (
	BucketReshardNone        BucketReshardState = "None"
	BucketReshardInProgress  BucketReshardState = "InProgress"
	BucketReshardInLogrecord BucketReshardState = "InLogrecord"
)

// Bucket is the direct JSON representation produced by RGW's bucket_stats().
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_bucket.cc
//   - src/rgw/driver/rados/rgw_bucket.cc (bucket_stats and RGWBucketAdminOp::info)
//   - src/rgw/rgw_bucket_layout.h
//   - src/rgw/rgw_bucket_layout.cc
type Bucket struct {
	Name                 string                  `json:"bucket"`
	Tenant               string                  `json:"tenant"`
	Versioning           BucketVersioningState   `json:"versioning"`
	Zonegroup            string                  `json:"zonegroup"`
	PlacementRule        string                  `json:"placement_rule"`
	ExplicitPlacement    BucketExplicitPlacement `json:"explicit_placement"`
	ID                   string                  `json:"id"`
	Marker               string                  `json:"marker"`
	IndexType            BucketIndexType         `json:"index_type"`
	IndexGeneration      int64                   `json:"index_generation"`
	ReshardStatus        BucketReshardState      `json:"reshard_status"`
	JudgeReshardLockTime string                  `json:"judge_reshard_lock_time"`
	ObjectLockEnabled    bool                    `json:"object_lock_enabled"`
	MFAEnabled           bool                    `json:"mfa_enabled"`
	Owner                string                  `json:"owner"`
	NumShards            int64                   `json:"num_shards"`
	Version              string                  `json:"ver"`
	MasterVersion        string                  `json:"master_ver"`
	MaxMarker            string                  `json:"max_marker"`
	Usage                map[string]StorageStats `json:"usage"`
	ModificationTime     time.Time               `json:"mtime"`
	CreationTime         time.Time               `json:"creation_time"`
	Quota                Quota                   `json:"bucket_quota"`
	Tags                 map[string]string       `json:"tagset,omitempty"`
	ReadTracker          int64                   `json:"read_tracker"`
}

type BucketExplicitPlacement struct {
	DataPool      string `json:"data_pool"`
	DataExtraPool string `json:"data_extra_pool"`
	IndexPool     string `json:"index_pool"`
}

// ListBucketsRequest filters both bucket-name and detailed bucket listings by
// user UID. An empty UID lists all buckets. For a user belonging to an account,
// Ceph lists that account's buckets.
//
// Ceph v20.2.4's src/rgw/driver/rados/rgw_rest_bucket.cc
// (RGWOp_Bucket_Info::execute) does not accept an account-id listing filter;
// rgw_bucket.cc (RGWBucketAdminOp::info) resolves account membership from UID.
type ListBucketsRequest struct {
	UID string
}

// ListBuckets lists detailed buckets through GET /admin/bucket?stats=true.
// Use ListBucketNames when only names are needed to avoid fetching statistics.
func (client *Client) ListBuckets(ctx context.Context, input ListBucketsRequest) ([]Bucket, error) {
	body, request, err := client.listBuckets(ctx, input, true)
	if err != nil {
		return nil, err
	}
	var result []Bucket
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return result, nil
}

// ListBucketNames lists bucket names through GET /admin/bucket?stats=false.
// Both listing methods accept the same owner filters in ListBucketsRequest.
// Ceph v20.2.4's src/rgw/driver/rados/rgw_rest_bucket.cc selects stats; its
// rgw_bucket.cc emits strings without stats and bucket_stats objects with stats.
func (client *Client) ListBucketNames(ctx context.Context, input ListBucketsRequest) ([]string, error) {
	body, request, err := client.listBuckets(ctx, input, false)
	if err != nil {
		return nil, err
	}
	var result []string
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return result, nil
}

func (client *Client) listBuckets(ctx context.Context, input ListBucketsRequest, stats bool) ([]byte, *http.Request, error) {
	if ctx == nil {
		return nil, nil, errors.New("rgw: context must not be nil")
	}
	query := url.Values{}
	setString(query, "uid", input.UID)
	setBool(query, "stats", &stats)
	return client.bucketRequest(ctx, http.MethodGet, query)
}

type GetBucketRequest struct {
	Name string
	UID  string
}

// GetBucket retrieves bucket information directly through GET /admin/bucket.
func (client *Client) GetBucket(ctx context.Context, input GetBucketRequest) (Bucket, error) {
	if ctx == nil {
		return Bucket{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return Bucket{}, errors.New("rgw: bucket name must not be empty")
	}
	query := url.Values{"bucket": {input.Name}}
	setString(query, "uid", input.UID)
	body, request, err := client.bucketRequest(ctx, http.MethodGet, query)
	if err != nil {
		return Bucket{}, err
	}
	var bucket Bucket
	if err := json.Unmarshal(body, &bucket); err != nil {
		return Bucket{}, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return bucket, nil
}

type LinkBucketRequest struct {
	Name      string
	UID       string
	AccountID string
	BucketID  string
	NewName   string
}

// LinkBucket links an existing bucket to a user or account through
// PUT /admin/bucket. It does not create a bucket.
func (client *Client) LinkBucket(ctx context.Context, input LinkBucketRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}
	if strings.TrimSpace(input.UID) == "" && strings.TrimSpace(input.AccountID) == "" {
		return errors.New("rgw: user UID or account ID must not be empty")
	}
	query := url.Values{"bucket": {input.Name}}
	setString(query, "uid", input.UID)
	setString(query, "account-id", input.AccountID)
	setString(query, "bucket-id", input.BucketID)
	setString(query, "new-bucket-name", input.NewName)
	return client.bucketAction(ctx, http.MethodPut, query)
}

type UnlinkBucketRequest struct {
	Name      string
	UID       string
	AccountID string
}

// UnlinkBucket removes a bucket from a user or account's bucket list through
// POST /admin/bucket. It does not delete the bucket or its objects.
func (client *Client) UnlinkBucket(ctx context.Context, input UnlinkBucketRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}
	if strings.TrimSpace(input.UID) == "" && strings.TrimSpace(input.AccountID) == "" {
		return errors.New("rgw: user UID or account ID must not be empty")
	}
	query := url.Values{"bucket": {input.Name}}
	setString(query, "uid", input.UID)
	setString(query, "account-id", input.AccountID)
	return client.bucketAction(ctx, http.MethodPost, query)
}

type DeleteBucketRequest struct {
	Name         string
	Tenant       string
	PurgeObjects *bool
	BypassGC     *bool
}

// DeleteBucket deletes a bucket directly through DELETE /admin/bucket.
func (client *Client) DeleteBucket(ctx context.Context, input DeleteBucketRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}
	query := url.Values{"bucket": {input.Name}}
	setString(query, "tenant", input.Tenant)
	setBool(query, "purge-objects", input.PurgeObjects)
	setBool(query, "bypass-gc", input.BypassGC)
	return client.bucketAction(ctx, http.MethodDelete, query)
}

func (client *Client) bucketRequest(ctx context.Context, method string, query url.Values) ([]byte, *http.Request, error) {
	request, err := client.newRequest(ctx, method, "bucket", query)
	if err != nil {
		return nil, nil, err
	}
	body, err := client.do(request)
	return body, request, err
}

func (client *Client) bucketAction(ctx context.Context, method string, query url.Values) error {
	request, err := client.newRequest(ctx, method, "bucket", query)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}
