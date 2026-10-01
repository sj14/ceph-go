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

// RateLimitScope identifies the global rate-limit default to change.
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

// GlobalRateLimitConfiguration contains all global RGW rate-limit defaults.
// All three scopes are returned by Ceph v20.2.4's src/rgw/rgw_rest_ratelimit.cc
// (RGWOp_Ratelimit_Info::execute).
type GlobalRateLimitConfiguration struct {
	Bucket    RateLimit `json:"bucket_ratelimit"`
	User      RateLimit `json:"user_ratelimit"`
	Anonymous RateLimit `json:"anonymous_ratelimit"`
}

type GetUserRateLimitRequest struct {
	UID string
}

// GetUserRateLimit retrieves the user's stored rate-limit configuration
// through GET /admin/ratelimit?ratelimit-scope=user. It does not merge global
// defaults; a user without a stored configuration has zero limits and is disabled.
func (client *Client) GetUserRateLimit(ctx context.Context, input GetUserRateLimitRequest) (RateLimit, error) {
	if ctx == nil {
		return RateLimit{}, errors.New("rgw: context must not be nil")
	}
	query := url.Values{}
	if err := setRateLimitTarget(query, RateLimitScopeUser, input.UID, "", "", false); err != nil {
		return RateLimit{}, err
	}
	var response struct {
		User RateLimit `json:"user_ratelimit"`
	}
	if err := client.getRateLimit(ctx, query, &response); err != nil {
		return RateLimit{}, err
	}
	return response.User, nil
}

type GetBucketRateLimitRequest struct {
	Bucket string
	Tenant string
}

// GetBucketRateLimit retrieves the bucket's stored rate-limit configuration
// through GET /admin/ratelimit?ratelimit-scope=bucket. It does not merge global
// defaults; a bucket without a stored configuration has zero limits and is disabled.
func (client *Client) GetBucketRateLimit(ctx context.Context, input GetBucketRateLimitRequest) (RateLimit, error) {
	if ctx == nil {
		return RateLimit{}, errors.New("rgw: context must not be nil")
	}
	query := url.Values{}
	if err := setRateLimitTarget(query, RateLimitScopeBucket, "", input.Bucket, input.Tenant, false); err != nil {
		return RateLimit{}, err
	}
	var response struct {
		Bucket RateLimit `json:"bucket_ratelimit"`
	}
	if err := client.getRateLimit(ctx, query, &response); err != nil {
		return RateLimit{}, err
	}
	return response.Bucket, nil
}

// GetGlobalRateLimits retrieves the bucket, user, and anonymous defaults
// through GET /admin/ratelimit?global=true. These are global defaults, rather
// than the stored configuration of an individual user or bucket.
func (client *Client) GetGlobalRateLimits(ctx context.Context) (GlobalRateLimitConfiguration, error) {
	var configuration GlobalRateLimitConfiguration
	if err := client.getRateLimit(ctx, url.Values{"global": {"true"}}, &configuration); err != nil {
		return GlobalRateLimitConfiguration{}, err
	}
	return configuration, nil
}

func (client *Client) getRateLimit(ctx context.Context, query url.Values, result any) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	request, err := client.newRequest(ctx, http.MethodGet, "ratelimit", query)
	if err != nil {
		return err
	}
	body, err := client.do(request)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return nil
}

// RateLimitUpdate contains the changes to a stored rate limit. Nil fields keep
// their current values; zero limits mean unlimited. At least one field must be
// provided, and limits must not be negative.
type RateLimitUpdate struct {
	MaxReadOps    *int64
	MaxWriteOps   *int64
	MaxReadBytes  *int64
	MaxWriteBytes *int64
	Enabled       *bool
}

type SetUserRateLimitRequest struct {
	UID string
	RateLimitUpdate
}

// SetUserRateLimit changes one user's stored rate limit through
// POST /admin/ratelimit?ratelimit-scope=user.
func (client *Client) SetUserRateLimit(ctx context.Context, input SetUserRateLimitRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	query := url.Values{}
	if err := setRateLimitTarget(query, RateLimitScopeUser, input.UID, "", "", false); err != nil {
		return err
	}
	return client.setRateLimit(ctx, query, input.RateLimitUpdate)
}

type SetBucketRateLimitRequest struct {
	Bucket string
	Tenant string
	RateLimitUpdate
}

// SetBucketRateLimit changes one bucket's stored rate limit through
// POST /admin/ratelimit?ratelimit-scope=bucket.
func (client *Client) SetBucketRateLimit(ctx context.Context, input SetBucketRateLimitRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	query := url.Values{}
	if err := setRateLimitTarget(query, RateLimitScopeBucket, "", input.Bucket, input.Tenant, false); err != nil {
		return err
	}
	return client.setRateLimit(ctx, query, input.RateLimitUpdate)
}

type SetGlobalRateLimitRequest struct {
	Scope RateLimitScope
	RateLimitUpdate
}

// SetGlobalRateLimit changes one global default through
// POST /admin/ratelimit?global=true. Scope must be user, bucket, or anon.
// It does not modify the stored configuration of an individual user or bucket.
func (client *Client) SetGlobalRateLimit(ctx context.Context, input SetGlobalRateLimitRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	query := url.Values{}
	if err := setRateLimitTarget(query, input.Scope, "", "", "", true); err != nil {
		return err
	}
	return client.setRateLimit(ctx, query, input.RateLimitUpdate)
}

func (client *Client) setRateLimit(ctx context.Context, query url.Values, input RateLimitUpdate) error {
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
