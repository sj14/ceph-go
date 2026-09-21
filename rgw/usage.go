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

// Usage is the direct representation produced by RGWUsage::show(). Entries or
// Summary may be absent when their corresponding request option is false.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/rgw_rest_usage.cc
//   - src/rgw/rgw_usage.cc
//   - src/rgw/rgw_usage.h
type Usage struct {
	Entries []UsageEntry   `json:"entries,omitempty"`
	Summary []UsageSummary `json:"summary,omitempty"`
}

// UsageEntry groups detailed bucket usage by user.
type UsageEntry struct {
	User    string        `json:"user"`
	Buckets []BucketUsage `json:"buckets"`
}

// BucketUsage describes usage recorded for one bucket and time interval.
type BucketUsage struct {
	Bucket     string          `json:"bucket"`
	Time       time.Time       `json:"time"`
	Epoch      int64           `json:"epoch"`
	Owner      string          `json:"owner"`
	Payer      string          `json:"payer,omitempty"`
	Categories []UsageCategory `json:"categories"`
	S3Select   S3SelectUsage   `json:"s3select"`
}

// UsageCategory contains counters for an open-ended RGW operation category.
type UsageCategory struct {
	Category      string `json:"category"`
	BytesSent     int64  `json:"bytes_sent"`
	BytesReceived int64  `json:"bytes_received"`
	Operations    int64  `json:"ops"`
	SuccessfulOps int64  `json:"successful_ops"`
}

// S3SelectUsage contains the S3 Select counters in a detailed usage entry.
type S3SelectUsage struct {
	BytesProcessed int64 `json:"bytes_processed"`
	BytesReturned  int64 `json:"bytes_returned"`
}

// UsageSummary contains aggregate counters for one user.
type UsageSummary struct {
	User       string          `json:"user"`
	Categories []UsageCategory `json:"categories"`
	Total      UsageTotal      `json:"total"`
}

// UsageTotal contains all aggregate counters returned in a usage summary.
type UsageTotal struct {
	BytesSent      int64 `json:"bytes_sent"`
	BytesReceived  int64 `json:"bytes_received"`
	Operations     int64 `json:"ops"`
	SuccessfulOps  int64 `json:"successful_ops"`
	BytesProcessed int64 `json:"bytes_processed"`
	BytesReturned  int64 `json:"bytes_returned"`
}

type GetUsageRequest struct {
	UID         string
	Bucket      string
	Tenant      string
	Start       *time.Time
	End         *time.Time
	ShowEntries *bool
	ShowSummary *bool
	Categories  []string
}

type TrimUsageRequest struct {
	UID       string
	Bucket    string
	Tenant    string
	Start     *time.Time
	End       *time.Time
	RemoveAll *bool
}

// GetUsage retrieves detailed and aggregate usage through GET /admin/usage.
func (client *Client) GetUsage(ctx context.Context, input GetUsageRequest) (Usage, error) {
	if ctx == nil {
		return Usage{}, errors.New("rgw: context must not be nil")
	}
	query := usageQuery(input.UID, input.Bucket, input.Tenant, input.Start, input.End)
	setBool(query, "show-entries", input.ShowEntries)
	setBool(query, "show-summary", input.ShowSummary)
	if len(input.Categories) > 0 {
		query.Set("categories", strings.Join(input.Categories, ","))
	}
	request, err := client.newRequest(ctx, http.MethodGet, "usage", query)
	if err != nil {
		return Usage{}, err
	}
	body, err := client.do(request)
	if err != nil {
		return Usage{}, err
	}
	var usage Usage
	if err := json.Unmarshal(body, &usage); err != nil {
		return Usage{}, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return usage, nil
}

// TrimUsage deletes matching usage records through DELETE /admin/usage. RGW
// requires RemoveAll to be true when no user, bucket, or time range is given.
func (client *Client) TrimUsage(ctx context.Context, input TrimUsageRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" && strings.TrimSpace(input.Bucket) == "" &&
		input.Start == nil && input.End == nil && (input.RemoveAll == nil || !*input.RemoveAll) {
		return errors.New("rgw: trimming all usage requires remove-all")
	}
	query := usageQuery(input.UID, input.Bucket, input.Tenant, input.Start, input.End)
	setBool(query, "remove-all", input.RemoveAll)
	request, err := client.newRequest(ctx, http.MethodDelete, "usage", query)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

func usageQuery(uid, bucket, tenant string, start, end *time.Time) url.Values {
	query := url.Values{}
	setString(query, "uid", uid)
	setString(query, "bucket", bucket)
	setString(query, "tenant", tenant)
	setTime(query, "start", start)
	setTime(query, "end", end)
	return query
}

func setTime(query url.Values, name string, value *time.Time) {
	if value != nil {
		query.Set(name, value.UTC().Format(time.RFC3339))
	}
}
