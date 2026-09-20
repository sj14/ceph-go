package rgw

import (
	"context"
	"errors"
	"strings"
)

// UserRateLimitConfiguration contains a user's rate limit.
type UserRateLimitConfiguration struct {
	User RateLimit `json:"user_ratelimit"`
}

// GetUserRateLimitRequest identifies the RGW user whose rate limit should be
// read.
type GetUserRateLimitRequest struct {
	UID string
}

// UpdateUserRateLimitRequest contains the user and rate-limit values to set.
// Zero limits mean unlimited.
type UpdateUserRateLimitRequest struct {
	UID string
	RateLimit
}

// GetGlobalUserRateLimit retrieves the global RGW rate-limit configuration
// through GET /api/rgw/user/ratelimit.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py
//     (RgwUser.get_global_rate_limit)
//   - src/pybind/mgr/dashboard/services/rgw_client.py (RgwRateLimit)
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-user.service.ts
func (client *Client) GetGlobalUserRateLimit(ctx context.Context) (GlobalRateLimitConfiguration, error) {
	if ctx == nil {
		return GlobalRateLimitConfiguration{}, errors.New("rgw: context must not be nil")
	}

	var configuration GlobalRateLimitConfiguration
	err := client.getRateLimit(ctx, client.endpoint("api/rgw/user/ratelimit"), &configuration)
	return configuration, err
}

// GetUserRateLimit retrieves a user's rate limit through
// GET /api/rgw/user/{uid}/ratelimit.
func (client *Client) GetUserRateLimit(ctx context.Context, input GetUserRateLimitRequest) (UserRateLimitConfiguration, error) {
	if ctx == nil {
		return UserRateLimitConfiguration{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return UserRateLimitConfiguration{}, errors.New("rgw: user UID must not be empty")
	}

	var configuration UserRateLimitConfiguration
	err := client.getRateLimit(ctx, client.userResourceEndpoint(input.UID, "ratelimit"), &configuration)
	return configuration, err
}

// UpdateUserRateLimit updates a user's rate limit through
// PUT /api/rgw/user/{uid}/ratelimit.
func (client *Client) UpdateUserRateLimit(ctx context.Context, input UpdateUserRateLimitRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	return client.updateRateLimit(ctx, client.userResourceEndpoint(input.UID, "ratelimit"), input.RateLimit)
}
