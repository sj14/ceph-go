package admin

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// BucketIndexLogStatusOption selects the bucket sync-status aggregation mode.
type BucketIndexLogStatusOption string

const (
	BucketIndexLogStatusOptionMerge BucketIndexLogStatusOption = "merge"
)

// BucketIndexLogInfo describes the bucket-index log layout and markers.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_log.cc
//   - src/rgw/driver/rados/rgw_rest_log.h
//   - src/rgw/driver/rados/rgw_data_sync.h
type BucketIndexLogInfo struct {
	BucketVersion string                     `json:"bucket_ver"`
	MasterVersion string                     `json:"master_ver"`
	MaxMarker     string                     `json:"max_marker"`
	SyncStopped   bool                       `json:"syncstopped"`
	OldestGen     int64                      `json:"oldest_gen"`
	LatestGen     int64                      `json:"latest_gen"`
	Generations   []BucketIndexLogGeneration `json:"generations"`
}

type BucketIndexLogGeneration struct {
	Generation int64 `json:"gen"`
	NumShards  int64 `json:"num_shards"`
}

type BucketIndexLogEntryList struct {
	Entries   []BucketIndexLogEntry `json:"entries"`
	Truncated bool                  `json:"truncated"`
	NextLog   *BucketIndexNextLog   `json:"next_log,omitempty"`
}

// BucketIndexNextLog identifies the layout following the listed generation.
// Ceph uses "generation" here but "gen" in BucketIndexLogGeneration.
type BucketIndexNextLog struct {
	Generation int64 `json:"generation"`
	NumShards  int64 `json:"num_shards"`
}

// BucketIndexLogEntry is open-ended because rgw_bi_log_entry serializes
// different fields for the operation stored in each entry.
type BucketIndexLogEntry map[string]any

type BucketIndexLogStatus struct {
	SyncStatus      BucketIndexFullSyncStatus    `json:"sync_status"`
	IncrementalInfo []BucketIndexShardSyncStatus `json:"inc_status"`
}

type BucketIndexFullSyncStatus struct {
	State                 LogShardSyncState           `json:"state"`
	Full                  BucketIndexFullSyncProgress `json:"full"`
	IncrementalGeneration int64                       `json:"incremental_gen"`
}

type BucketIndexFullSyncProgress struct {
	Position ObjectKey `json:"position"`
	Count    int64     `json:"count"`
}

// ObjectKey is Ceph's rgw_obj_key representation.
type ObjectKey struct {
	Name      string `json:"name"`
	Instance  string `json:"instance"`
	Namespace string `json:"ns"`
}

type BucketIndexShardSyncStatus struct {
	Status            LogShardSyncState                `json:"status"`
	IncrementalMarker BucketIndexIncrementalSyncMarker `json:"inc_marker"`
}

type BucketIndexIncrementalSyncMarker struct {
	Position  string `json:"position"`
	Timestamp string `json:"timestamp"`
}

type GetBucketIndexLogInfoRequest struct {
	Tenant         string
	Bucket         string
	BucketInstance string
}

type ListBucketIndexLogEntriesRequest struct {
	Tenant         string
	Bucket         string
	BucketInstance string
	Marker         string
	MaxEntries     *int64
	Generation     *int64
}

type GetBucketIndexLogStatusRequest struct {
	Bucket       string
	SourceZone   string
	SourceBucket string
	Option       BucketIndexLogStatusOption
}

// GetBucketIndexLogInfo retrieves bucket-index log information. Ceph v20.2.4
// unconditionally parses bucket-instance, so it is required even though the
// handler initially appears to accept bucket alone.
func (client *Client) GetBucketIndexLogInfo(ctx context.Context, input GetBucketIndexLogInfoRequest) (BucketIndexLogInfo, error) {
	if ctx == nil {
		return BucketIndexLogInfo{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.BucketInstance) == "" {
		return BucketIndexLogInfo{}, errors.New("admin: bucket instance must not be empty")
	}
	query := bucketIndexLogQuery(input.Tenant, input.Bucket, input.BucketInstance)
	query.Set("info", "")
	var result BucketIndexLogInfo
	err := client.getLog(ctx, query, &result)
	return result, err
}

// ListBucketIndexLogEntries lists bucket-index log entries using Ceph's
// version 2 response format, which includes truncation and next-log metadata.
func (client *Client) ListBucketIndexLogEntries(ctx context.Context, input ListBucketIndexLogEntriesRequest) (BucketIndexLogEntryList, error) {
	if ctx == nil {
		return BucketIndexLogEntryList{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.BucketInstance) == "" {
		return BucketIndexLogEntryList{}, errors.New("admin: bucket instance must not be empty")
	}
	query := bucketIndexLogQuery(input.Tenant, input.Bucket, input.BucketInstance)
	query.Set("format-ver", "2")
	setString(query, "marker", input.Marker)
	setInt64(query, "max-entries", input.MaxEntries)
	setInt64(query, "generation", input.Generation)
	var result BucketIndexLogEntryList
	err := client.getLog(ctx, query, &result)
	return result, err
}

// GetBucketIndexLogStatus retrieves version 2 bucket synchronization status.
func (client *Client) GetBucketIndexLogStatus(ctx context.Context, input GetBucketIndexLogStatusRequest) (BucketIndexLogStatus, error) {
	if ctx == nil {
		return BucketIndexLogStatus{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.Bucket) == "" && strings.TrimSpace(input.SourceBucket) == "" {
		return BucketIndexLogStatus{}, errors.New("admin: bucket or source bucket must not be empty")
	}
	if input.Option != "" && input.Option != BucketIndexLogStatusOptionMerge {
		return BucketIndexLogStatus{}, errors.New("admin: bucket-index log status options must be empty or merge")
	}
	query := url.Values{
		"type":    {"bucket-index"},
		"status":  {""},
		"version": {"2"},
	}
	setString(query, "bucket", input.Bucket)
	setString(query, "source-zone", input.SourceZone)
	setString(query, "source-bucket", input.SourceBucket)
	setString(query, "options", string(input.Option))
	var result BucketIndexLogStatus
	err := client.getLog(ctx, query, &result)
	return result, err
}

func bucketIndexLogQuery(tenant, bucket, bucketInstance string) url.Values {
	query := url.Values{"type": {"bucket-index"}}
	setString(query, "tenant", tenant)
	setString(query, "bucket", bucket)
	setString(query, "bucket-instance", bucketInstance)
	return query
}
