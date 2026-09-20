package rgw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// LockMode is an S3 Object Lock retention mode.
type LockMode string

const (
	LockModeGovernance LockMode = "GOVERNANCE"
	LockModeCompliance LockMode = "COMPLIANCE"
)

// GetBucketRequest identifies a bucket and, optionally, the RGW daemon through
// which Ceph Dashboard should retrieve it.
type GetBucketRequest struct {
	Name       string
	DaemonName string
}

// ListBucketsRequest filters buckets by owner when UID is set and selects the
// RGW daemon through which Ceph Dashboard should retrieve them.
type ListBucketsRequest struct {
	UID        string
	DaemonName string
}

// DeleteBucketRequest identifies an empty bucket to delete and, optionally,
// the RGW daemon through which Ceph Dashboard should delete it.
type DeleteBucketRequest struct {
	Name       string
	DaemonName string
}

// BucketVersioningState is an S3 bucket versioning state accepted by Ceph.
type BucketVersioningState string

const (
	BucketVersioningEnabled   BucketVersioningState = "Enabled"
	BucketVersioningSuspended BucketVersioningState = "Suspended"
	BucketVersioningOff       BucketVersioningState = "Off"
)

// BucketMFADeleteState is the MFA Delete state returned and accepted by Ceph.
type BucketMFADeleteState string

const (
	BucketMFADeleteEnabled  BucketMFADeleteState = "Enabled"
	BucketMFADeleteDisabled BucketMFADeleteState = "Disabled"
)

// BucketEncryptionStatus is the server-side encryption status returned by
// Ceph for a bucket.
type BucketEncryptionStatus string

const (
	BucketEncryptionEnabled  BucketEncryptionStatus = "Enabled"
	BucketEncryptionDisabled BucketEncryptionStatus = "Disabled"
)

// BucketEncryptionType selects the S3 server-side encryption algorithm.
type BucketEncryptionType string

const (
	BucketEncryptionAES256 BucketEncryptionType = "AES256"
	BucketEncryptionAWSKMS BucketEncryptionType = "aws:kms"
)

// BucketCannedACL is a canned S3 ACL value. The constants are the values
// exposed by Ceph Dashboard; callers can convert other RGW-supported values.
type BucketCannedACL string

const (
	BucketCannedACLPrivate           BucketCannedACL = "private"
	BucketCannedACLPublicRead        BucketCannedACL = "public-read"
	BucketCannedACLPublicReadWrite   BucketCannedACL = "public-read-write"
	BucketCannedACLAuthenticatedRead BucketCannedACL = "authenticated-read"
)

// Bucket is the bucket representation assembled by Ceph Dashboard. Fields
// whose shape is controlled by RGW configuration are retained as raw JSON.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwBucket.get)
//   - src/pybind/mgr/dashboard/frontend/src/app/ceph/rgw/models/rgw-bucket.ts
//   - src/pybind/mgr/dashboard/frontend/src/app/ceph/rgw/models/rgw-bucket-versioning.ts
//   - src/pybind/mgr/dashboard/frontend/src/app/ceph/rgw/models/rgw-bucket-mfa-delete.ts
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts
//   - src/pybind/mgr/dashboard/services/rgw_client.py
type Bucket struct {
	Name                 string                  `json:"bucket"`
	Tenant               string                  `json:"tenant"`
	BID                  string                  `json:"bid"`
	Zonegroup            string                  `json:"zonegroup"`
	PlacementRule        string                  `json:"placement_rule"`
	ExplicitPlacement    BucketExplicitPlacement `json:"explicit_placement"`
	ID                   string                  `json:"id"`
	Marker               string                  `json:"marker"`
	IndexType            string                  `json:"index_type"`
	IndexGeneration      int64                   `json:"index_generation"`
	NumShards            int64                   `json:"num_shards"`
	ReshardStatus        string                  `json:"reshard_status"`
	JudgeReshardLockTime string                  `json:"judge_reshard_lock_time"`
	ObjectLockEnabled    bool                    `json:"object_lock_enabled"`
	MFAEnabled           bool                    `json:"mfa_enabled"`
	Owner                string                  `json:"owner"`
	Version              string                  `json:"ver"`
	MasterVersion        string                  `json:"master_ver"`
	MTime                string                  `json:"mtime"`
	CreationTime         string                  `json:"creation_time"`
	MaxMarker            string                  `json:"max_marker"`
	Usage                map[string]BucketUsage  `json:"usage"`
	Quota                BucketQuota             `json:"bucket_quota"`
	ReadTracker          int64                   `json:"read_tracker"`
	Encryption           BucketEncryptionStatus  `json:"encryption"`
	Versioning           BucketVersioningState   `json:"versioning"`
	MFADelete            BucketMFADeleteState    `json:"mfa_delete"`
	BucketPolicy         json.RawMessage         `json:"bucket_policy"`
	ACL                  string                  `json:"acl"`
	Replication          BucketReplication       `json:"replication"`
	Lifecycle            json.RawMessage         `json:"lifecycle"`
	LifecycleProgress    json.RawMessage         `json:"lifecycle_progress"`
	LockEnabled          bool                    `json:"lock_enabled"`
	LockMode             LockMode                `json:"lock_mode"`
	LockRetentionDays    *int64                  `json:"lock_retention_period_days"`
	LockRetentionYears   *int64                  `json:"lock_retention_period_years"`
}

// BucketExplicitPlacement contains explicitly selected RGW pools.
type BucketExplicitPlacement struct {
	DataPool      string `json:"data_pool"`
	DataExtraPool string `json:"data_extra_pool"`
	IndexPool     string `json:"index_pool"`
}

// BucketUsage contains usage counters for one RGW storage category.
type BucketUsage struct {
	Size           int64 `json:"size"`
	SizeActual     int64 `json:"size_actual"`
	SizeUtilized   int64 `json:"size_utilized"`
	SizeKB         int64 `json:"size_kb"`
	SizeKBActual   int64 `json:"size_kb_actual"`
	SizeKBUtilized int64 `json:"size_kb_utilized"`
	NumObjects     int64 `json:"num_objects"`
}

// BucketQuota contains the bucket-level RGW quota.
type BucketQuota struct {
	Enabled    bool  `json:"enabled"`
	CheckOnRaw bool  `json:"check_on_raw"`
	MaxSize    int64 `json:"max_size"`
	MaxSizeKB  int64 `json:"max_size_kb"`
	MaxObjects int64 `json:"max_objects"`
}

// BucketReplication summarizes both S3 replication rules and RGW sync policy.
type BucketReplication struct {
	SyncPolicyActive           bool            `json:"sync_policy_active"`
	ReplicationRulesConfigured bool            `json:"replication_rules_configured"`
	Policy                     json.RawMessage `json:"policy"`
}

// ListBuckets retrieves detailed buckets through GET /api/rgw/bucket using
// Ceph Dashboard API version 1.1. The method sends stats=true so that its
// return type is consistently []Bucket rather than Ceph's alternate []string
// response.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwBucket.list)
//   - src/pybind/mgr/dashboard/controllers/_rest_controller.py
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts
//   - qa/tasks/mgr/dashboard/test_rgw.py (RgwBucketTest.test_all)
func (client *Client) ListBuckets(ctx context.Context, input ListBucketsRequest) ([]Bucket, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}

	endpoint := client.endpoint("api/rgw/bucket")
	query := url.Values{"stats": {"true"}}
	setOptional(query, "uid", input.UID)
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", mediaTypeV1_1)
	body, err := client.do(request)
	if err != nil {
		return nil, err
	}

	var buckets []Bucket
	if err := json.Unmarshal(body, &buckets); err != nil {
		return nil, fmt.Errorf("rgw: decode GET %s response: %w", request.URL.Path, err)
	}
	return buckets, nil
}

// GetBucket retrieves a bucket through GET /api/rgw/bucket/{bucket}.
func (client *Client) GetBucket(ctx context.Context, input GetBucketRequest) (Bucket, error) {
	if ctx == nil {
		return Bucket{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return Bucket{}, errors.New("rgw: bucket name must not be empty")
	}

	endpoint := client.bucketEndpoint(input.Name)
	query := url.Values{}
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Bucket{}, err
	}
	body, err := client.do(request)
	if err != nil {
		return Bucket{}, err
	}

	var bucket Bucket
	if err := json.Unmarshal(body, &bucket); err != nil {
		return Bucket{}, fmt.Errorf("rgw: decode GET %s response: %w", request.URL.Path, err)
	}
	return bucket, nil
}

// UpdateBucketRequest contains the parameters accepted by Ceph's RGW bucket
// update controller. Name, BucketID, and UID are required. UID is the desired
// owner and must have an S3 key because Ceph uses that credential for the
// follow-up S3 configuration calls. EncryptionEnabled is the desired
// encryption state, matching Ceph Dashboard's frontend.
type UpdateBucketRequest struct {
	Name     string
	BucketID string
	UID      string

	VersioningState    BucketVersioningState
	EncryptionEnabled  bool
	EncryptionType     BucketEncryptionType
	KeyID              string
	MFADelete          BucketMFADeleteState
	MFATokenSerial     string
	MFATokenPIN        string
	LockMode           LockMode
	LockRetentionDays  *int64
	LockRetentionYears *int64
	Tags               string
	BucketPolicy       string
	CannedACL          BucketCannedACL
	ReplicationEnabled *bool
	Lifecycle          string
	DaemonName         string
}

// UpdateBucket updates a bucket through PUT /api/rgw/bucket/{bucket}.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwBucket.set)
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts
//   - src/pybind/mgr/dashboard/services/rgw_client.py
//   - qa/tasks/mgr/dashboard/test_rgw.py (RgwBucketTest.test_all)
func (client *Client) UpdateBucket(ctx context.Context, input UpdateBucketRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}
	if strings.TrimSpace(input.BucketID) == "" {
		return errors.New("rgw: bucket ID must not be empty")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("rgw: bucket UID must not be empty")
	}

	endpoint := client.bucketEndpoint(input.Name)
	query := url.Values{
		"bucket_id":        {input.BucketID},
		"uid":              {input.UID},
		"encryption_state": {strconv.FormatBool(input.EncryptionEnabled)},
	}
	setOptional(query, "versioning_state", string(input.VersioningState))
	setOptional(query, "encryption_type", string(input.EncryptionType))
	setOptional(query, "key_id", input.KeyID)
	setOptional(query, "mfa_delete", string(input.MFADelete))
	setOptional(query, "mfa_token_serial", input.MFATokenSerial)
	setOptional(query, "mfa_token_pin", input.MFATokenPIN)
	setOptional(query, "lock_mode", string(input.LockMode))
	setOptionalInt(query, "lock_retention_period_days", input.LockRetentionDays)
	setOptionalInt(query, "lock_retention_period_years", input.LockRetentionYears)
	setOptional(query, "tags", input.Tags)
	setOptional(query, "bucket_policy", input.BucketPolicy)
	setOptional(query, "canned_acl", string(input.CannedACL))
	setOptionalBool(query, "replication", input.ReplicationEnabled)
	setOptional(query, "lifecycle", input.Lifecycle)
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), nil)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

// DeleteBucket deletes an empty bucket through
// DELETE /api/rgw/bucket/{bucket}.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwBucket.delete)
//   - src/pybind/mgr/dashboard/controllers/_rest_controller.py
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts
//   - src/pybind/mgr/dashboard/services/rgw_client.py (RgwClient.proxy)
func (client *Client) DeleteBucket(ctx context.Context, input DeleteBucketRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}

	endpoint := client.bucketEndpoint(input.Name)
	query := url.Values{}
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint.String(), nil)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

// CreateBucketRequest contains the arguments accepted by Ceph's RGW bucket
// create controller. Name and UID are required; all other fields are optional.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwBucket.create)
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts
//   - src/pybind/mgr/dashboard/services/rgw_client.py (create_bucket)
type CreateBucketRequest struct {
	Name string
	UID  string

	Zonegroup       string
	PlacementTarget string

	LockEnabled        bool
	LockMode           LockMode
	LockRetentionDays  *int64
	LockRetentionYears *int64
	EncryptionEnabled  bool
	EncryptionType     BucketEncryptionType
	KeyID              string
	Tags               string
	BucketPolicy       string
	CannedACL          BucketCannedACL
	ReplicationEnabled bool
	DaemonName         string
}

// CreateBucket creates a bucket through POST /api/rgw/bucket.
//
// Ceph's own v20.2.4 frontend sends these parameters in the query string with
// an empty request body, so this method intentionally does the same.
func (client *Client) CreateBucket(ctx context.Context, input CreateBucketRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("rgw: bucket UID must not be empty")
	}
	endpoint := client.endpoint("api/rgw/bucket")
	query := url.Values{
		"bucket":           {input.Name},
		"uid":              {input.UID},
		"lock_enabled":     {strconv.FormatBool(input.LockEnabled)},
		"encryption_state": {strconv.FormatBool(input.EncryptionEnabled)},
		"replication":      {strconv.FormatBool(input.ReplicationEnabled)},
	}
	setOptional(query, "zonegroup", input.Zonegroup)
	setOptional(query, "placement_target", input.PlacementTarget)
	setOptional(query, "lock_mode", string(input.LockMode))
	setOptionalInt(query, "lock_retention_period_days", input.LockRetentionDays)
	setOptionalInt(query, "lock_retention_period_years", input.LockRetentionYears)
	setOptional(query, "encryption_type", string(input.EncryptionType))
	setOptional(query, "key_id", input.KeyID)
	setOptional(query, "tags", input.Tags)
	setOptional(query, "bucket_policy", input.BucketPolicy)
	setOptional(query, "canned_acl", string(input.CannedACL))
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

func (client *Client) bucketEndpoint(name string) *url.URL {
	endpoint := client.endpoint("api/rgw/bucket")
	baseEscapedPath := strings.TrimRight(endpoint.EscapedPath(), "/")
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + name
	endpoint.RawPath = baseEscapedPath + "/" + url.PathEscape(name)
	return endpoint
}

func setOptional(values url.Values, name, value string) {
	if value != "" {
		values.Set(name, value)
	}
}

func setOptionalInt(values url.Values, name string, value *int64) {
	if value != nil {
		values.Set(name, strconv.FormatInt(*value, 10))
	}
}
