package dashboard

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// BucketRateLimitConfiguration contains a bucket's rate limit.
type BucketRateLimitConfiguration struct {
	Bucket RateLimit `json:"bucket_ratelimit"`
}

// GetBucketRateLimitRequest identifies the bucket whose rate limit should be
// read.
type GetBucketRateLimitRequest struct {
	Name string
}

// UpdateBucketRateLimitRequest contains the bucket and rate-limit values to
// set. Zero limits mean unlimited.
type UpdateBucketRateLimitRequest struct {
	Name string
	RateLimit
}

// GetGlobalBucketRateLimit retrieves the global RGW rate-limit configuration
// through GET /api/rgw/bucket/ratelimit.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py
//     (RgwBucket.get_global_rate_limit)
//   - src/pybind/mgr/dashboard/services/rgw_client.py (RgwRateLimit)
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts
func (client *Client) GetGlobalBucketRateLimit(ctx context.Context) (GlobalRateLimitConfiguration, error) {
	if ctx == nil {
		return GlobalRateLimitConfiguration{}, errors.New("rgw: context must not be nil")
	}

	var configuration GlobalRateLimitConfiguration
	err := client.getRateLimit(ctx, client.endpoint("api/rgw/bucket/ratelimit"), &configuration)
	return configuration, err
}

// GetBucketRateLimit retrieves a bucket's rate limit through
// GET /api/rgw/bucket/{bucket}/ratelimit.
func (client *Client) GetBucketRateLimit(ctx context.Context, input GetBucketRateLimitRequest) (BucketRateLimitConfiguration, error) {
	if ctx == nil {
		return BucketRateLimitConfiguration{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return BucketRateLimitConfiguration{}, errors.New("rgw: bucket name must not be empty")
	}

	var configuration BucketRateLimitConfiguration
	err := client.getRateLimit(ctx, client.bucketRateLimitEndpoint(input.Name), &configuration)
	return configuration, err
}

// UpdateBucketRateLimit updates a bucket's rate limit through
// PUT /api/rgw/bucket/{bucket}/ratelimit.
func (client *Client) UpdateBucketRateLimit(ctx context.Context, input UpdateBucketRateLimitRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}
	return client.updateRateLimit(ctx, client.bucketRateLimitEndpoint(input.Name), input.RateLimit)
}

func (client *Client) bucketRateLimitEndpoint(name string) *url.URL {
	endpoint := client.bucketEndpoint(name)
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/ratelimit"
	endpoint.RawPath = strings.TrimRight(endpoint.RawPath, "/") + "/ratelimit"
	return endpoint
}
