package mgr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/sj14/rgw-go/rgw"
)

// GlobalRateLimitConfiguration contains all global RGW rate-limit scopes.
type GlobalRateLimitConfiguration struct {
	Bucket    rgw.RateLimit `json:"bucket_ratelimit"`
	User      rgw.RateLimit `json:"user_ratelimit"`
	Anonymous rgw.RateLimit `json:"anonymous_ratelimit"`
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
		return fmt.Errorf("mgr: decode GET %s response: %w", request.URL.Path, err)
	}
	return nil
}

func (client *Client) updateRateLimit(ctx context.Context, endpoint *url.URL, rateLimit rgw.RateLimit) error {
	body, err := json.Marshal(rateLimit)
	if err != nil {
		return fmt.Errorf("mgr: encode rate limit: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	_, err = client.do(request)
	return err
}
