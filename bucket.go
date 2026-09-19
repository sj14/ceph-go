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

// Bucket is the bucket representation assembled by Ceph Dashboard. Fields
// whose shape is controlled by RGW configuration are retained as raw JSON.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwBucket.get)
//   - src/pybind/mgr/dashboard/frontend/src/app/ceph/rgw/models/rgw-bucket.ts
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
	Encryption           string                  `json:"encryption"`
	Versioning           string                  `json:"versioning"`
	MFADelete            string                  `json:"mfa_delete"`
	BucketPolicy         json.RawMessage         `json:"bucket_policy"`
	ACL                  string                  `json:"acl"`
	Replication          BucketReplication       `json:"replication"`
	Lifecycle            json.RawMessage         `json:"lifecycle"`
	LifecycleProgress    json.RawMessage         `json:"lifecycle_progress"`
	LockEnabled          bool                    `json:"lock_enabled"`
	LockMode             string                  `json:"lock_mode"`
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

// GetBucket retrieves a bucket through GET /api/rgw/bucket/{bucket}.
func (client *Client) GetBucket(ctx context.Context, input GetBucketRequest) (Bucket, error) {
	if ctx == nil {
		return Bucket{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return Bucket{}, errors.New("rgw: bucket name must not be empty")
	}

	endpoint := client.endpoint("api/rgw/bucket")
	baseEscapedPath := strings.TrimRight(endpoint.EscapedPath(), "/")
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + input.Name
	endpoint.RawPath = baseEscapedPath + "/" + url.PathEscape(input.Name)
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
	EncryptionType     string
	KeyID              string
	Tags               string
	BucketPolicy       string
	CannedACL          string
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
	setOptional(query, "encryption_type", input.EncryptionType)
	setOptional(query, "key_id", input.KeyID)
	setOptional(query, "tags", input.Tags)
	setOptional(query, "bucket_policy", input.BucketPolicy)
	setOptional(query, "canned_acl", input.CannedACL)
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
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
