package rgw

import (
	"context"
	"errors"
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

// CreateBucketRequest contains the arguments accepted by Ceph's RGW bucket
// create controller. Name and UID are required; all other fields are optional.
//
// Verified against Ceph v21.3.0 (tag commit cc6b5e2da077):
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
// Ceph's own v21.3.0 frontend sends these parameters in the query string with
// an empty request body, so this method intentionally does the same.
func (client *Client) CreateBucket(ctx context.Context, input CreateBucketRequest) (*Response, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("rgw: bucket name must not be empty")
	}
	if strings.TrimSpace(input.UID) == "" {
		return nil, errors.New("rgw: bucket UID must not be empty")
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
		return nil, err
	}
	return client.do(request)
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
