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

// RateLimitScope identifies the object whose rate limit is read or changed.
type RateLimitScope string

const (
	RateLimitScopeUser      RateLimitScope = "user"
	RateLimitScopeBucket    RateLimitScope = "bucket"
	RateLimitScopeAnonymous RateLimitScope = "anon"
)

// RateLimit contains the per-minute operation and byte limits for an RGW
// scope. A zero limit means unlimited.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/rgw_rest_ratelimit.cc
//   - src/rgw/rgw_rest_ratelimit.h
//   - src/rgw/rgw_common.h (RGWRateLimitInfo)
//   - src/rgw/rgw_common.cc (RGWRateLimitInfo::dump)
//   - src/rgw/driver/rados/rgw_sal_rados.cc (register_admin_apis)
type RateLimit struct {
	MaxReadOps    int64 `json:"max_read_ops"`
	MaxWriteOps   int64 `json:"max_write_ops"`
	MaxReadBytes  int64 `json:"max_read_bytes"`
	MaxWriteBytes int64 `json:"max_write_bytes"`
	Enabled       bool  `json:"enabled"`
}

// RateLimitConfiguration contains the sections returned by RGW. A scoped
// request sets only User or Bucket; a global request sets all three fields.
type RateLimitConfiguration struct {
	Bucket    *RateLimit `json:"bucket_ratelimit,omitempty"`
	User      *RateLimit `json:"user_ratelimit,omitempty"`
	Anonymous *RateLimit `json:"anonymous_ratelimit,omitempty"`
}

type GetRateLimitRequest struct {
	Scope  RateLimitScope
	UID    string
	Bucket string
	Tenant string
	Global bool
}

type SetRateLimitRequest struct {
	Scope         RateLimitScope
	UID           string
	Bucket        string
	Tenant        string
	Global        bool
	MaxReadOps    *int64
	MaxWriteOps   *int64
	MaxReadBytes  *int64
	MaxWriteBytes *int64
	Enabled       *bool
}

// GetRateLimit retrieves a user, bucket, or complete global configuration
// through GET /admin/ratelimit. Global requests return the bucket, user, and
// anonymous defaults and do not require a scope.
func (client *Client) GetRateLimit(ctx context.Context, input GetRateLimitRequest) (RateLimitConfiguration, error) {
	if ctx == nil {
		return RateLimitConfiguration{}, errors.New("rgw: context must not be nil")
	}
	query := url.Values{}
	if input.Global {
		query.Set("global", "true")
	} else {
		if err := setRateLimitTarget(query, input.Scope, input.UID, input.Bucket, input.Tenant, false); err != nil {
			return RateLimitConfiguration{}, err
		}
	}
	request, err := client.newRequest(ctx, http.MethodGet, "ratelimit", query)
	if err != nil {
		return RateLimitConfiguration{}, err
	}
	body, err := client.do(request)
	if err != nil {
		return RateLimitConfiguration{}, err
	}
	var configuration RateLimitConfiguration
	if err := json.Unmarshal(body, &configuration); err != nil {
		return RateLimitConfiguration{}, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return configuration, nil
}

// SetRateLimit changes a user, bucket, or one global rate-limit scope through
// POST /admin/ratelimit. At least one limit or Enabled must be provided.
func (client *Client) SetRateLimit(ctx context.Context, input SetRateLimitRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if input.MaxReadOps == nil && input.MaxWriteOps == nil && input.MaxReadBytes == nil &&
		input.MaxWriteBytes == nil && input.Enabled == nil {
		return errors.New("rgw: at least one rate-limit value must be provided")
	}
	for _, limit := range []struct {
		name  string
		value *int64
	}{
		{"max read operations", input.MaxReadOps},
		{"max write operations", input.MaxWriteOps},
		{"max read bytes", input.MaxReadBytes},
		{"max write bytes", input.MaxWriteBytes},
	} {
		if limit.value != nil && *limit.value < 0 {
			return fmt.Errorf("rgw: %s must not be negative", limit.name)
		}
	}
	query := url.Values{}
	if err := setRateLimitTarget(query, input.Scope, input.UID, input.Bucket, input.Tenant, input.Global); err != nil {
		return err
	}
	setInt64(query, "max-read-ops", input.MaxReadOps)
	setInt64(query, "max-write-ops", input.MaxWriteOps)
	setInt64(query, "max-read-bytes", input.MaxReadBytes)
	setInt64(query, "max-write-bytes", input.MaxWriteBytes)
	setBool(query, "enabled", input.Enabled)
	request, err := client.newRequest(ctx, http.MethodPost, "ratelimit", query)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

func setRateLimitTarget(query url.Values, scope RateLimitScope, uid, bucket, tenant string, global bool) error {
	if global {
		if scope != RateLimitScopeUser && scope != RateLimitScopeBucket && scope != RateLimitScopeAnonymous {
			return errors.New("rgw: global rate-limit scope must be user, bucket, or anon")
		}
		query.Set("global", "true")
		query.Set("ratelimit-scope", string(scope))
		return nil
	}
	switch scope {
	case RateLimitScopeUser:
		if strings.TrimSpace(uid) == "" {
			return errors.New("rgw: user UID must not be empty")
		}
		query.Set("uid", uid)
	case RateLimitScopeBucket:
		if strings.TrimSpace(bucket) == "" {
			return errors.New("rgw: bucket name must not be empty")
		}
		query.Set("bucket", bucket)
		setString(query, "tenant", tenant)
	case RateLimitScopeAnonymous:
		return errors.New("rgw: anonymous rate limits must be global")
	default:
		return errors.New("rgw: rate-limit scope must be user or bucket")
	}
	query.Set("ratelimit-scope", string(scope))
	return nil
}
