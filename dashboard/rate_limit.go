package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// RateLimit contains the per-minute operation and byte limits for an RGW
// scope. A zero limit means unlimited.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py
//     (RgwBucket and RgwUser rate-limit endpoints)
//   - src/pybind/mgr/dashboard/controllers/_endpoint.py
//   - src/pybind/mgr/dashboard/controllers/_base_controller.py
//   - src/pybind/mgr/dashboard/services/rgw_client.py (RgwRateLimit)
//   - src/pybind/mgr/dashboard/frontend/src/app/ceph/rgw/models/rgw-rate-limit.ts
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-user.service.ts
//   - src/rgw/rgw_rest_ratelimit.cc
type RateLimit struct {
	MaxReadOps    int64 `json:"max_read_ops"`
	MaxWriteOps   int64 `json:"max_write_ops"`
	MaxReadBytes  int64 `json:"max_read_bytes"`
	MaxWriteBytes int64 `json:"max_write_bytes"`
	Enabled       bool  `json:"enabled"`
}

// GlobalRateLimitConfiguration contains all global RGW rate-limit scopes.
type GlobalRateLimitConfiguration struct {
	Bucket    RateLimit `json:"bucket_ratelimit"`
	User      RateLimit `json:"user_ratelimit"`
	Anonymous RateLimit `json:"anonymous_ratelimit"`
}

func (client *Client) getRateLimit(ctx context.Context, endpoint *url.URL, configuration any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	body, err := client.do(request)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, configuration); err != nil {
		return fmt.Errorf("rgw: decode GET %s response: %w", request.URL.Path, err)
	}
	return nil
}

func (client *Client) updateRateLimit(ctx context.Context, endpoint *url.URL, rateLimit RateLimit) error {
	body, err := json.Marshal(rateLimit)
	if err != nil {
		return fmt.Errorf("rgw: encode rate limit: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	_, err = client.do(request)
	return err
}
