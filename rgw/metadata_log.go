package rgw

import (
	"context"
	"errors"
	"net/url"
)

// MetadataLogOperationStatus is the operation state stored in a metadata-log
// entry.
type MetadataLogOperationStatus string

const (
	MetadataLogOperationWrite    MetadataLogOperationStatus = "write"
	MetadataLogOperationSetAttrs MetadataLogOperationStatus = "set_attrs"
	MetadataLogOperationRemove   MetadataLogOperationStatus = "remove"
	MetadataLogOperationComplete MetadataLogOperationStatus = "complete"
	MetadataLogOperationAbort    MetadataLogOperationStatus = "abort"
	MetadataLogOperationUnknown  MetadataLogOperationStatus = "unknown"
)

// MetadataLogSyncMarkerState is the integer state emitted for a metadata sync
// marker. Unlike data-log markers, this field is not serialized as a string.
type MetadataLogSyncMarkerState int64

const (
	MetadataLogSyncMarkerFull        MetadataLogSyncMarkerState = 0
	MetadataLogSyncMarkerIncremental MetadataLogSyncMarkerState = 1
)

// MetadataLogInfo describes the metadata log as a whole.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_log.cc
//   - src/rgw/driver/rados/rgw_rest_log.h
//   - src/rgw/rgw_metadata.cc
//   - src/rgw/rgw_meta_sync_status.h
//   - src/rgw/driver/rados/rgw_sync.cc
type MetadataLogInfo struct {
	NumShards  int64  `json:"num_objects"`
	Period     string `json:"period,omitempty"`
	RealmEpoch int64  `json:"realm_epoch,omitempty"`
}

type MetadataLogShardInfo struct {
	Marker     string `json:"marker"`
	LastUpdate string `json:"last_update"`
}

type MetadataLogEntryList struct {
	Marker    string             `json:"marker"`
	Truncated bool               `json:"truncated"`
	Entries   []MetadataLogEntry `json:"entries"`
}

type MetadataLogEntry struct {
	ID        string          `json:"id"`
	Section   string          `json:"section"`
	Name      string          `json:"name"`
	Timestamp string          `json:"timestamp"`
	Data      MetadataLogData `json:"data"`
}

type MetadataLogData struct {
	ReadVersion  ObjectVersion             `json:"read_version"`
	WriteVersion ObjectVersion             `json:"write_version"`
	Status       MetadataLogOperationState `json:"status"`
}

type MetadataLogOperationState struct {
	Status MetadataLogOperationStatus `json:"status"`
}

type MetadataLogSyncStatus struct {
	Info    MetadataLogSyncInfo              `json:"info"`
	Markers map[string]MetadataLogSyncMarker `json:"markers"`
}

type MetadataLogSyncInfo struct {
	Status     LogSyncState `json:"status"`
	NumShards  int64        `json:"num_shards"`
	Period     string       `json:"period"`
	RealmEpoch int64        `json:"realm_epoch"`
}

type MetadataLogSyncMarker struct {
	State          MetadataLogSyncMarkerState `json:"state"`
	Marker         string                     `json:"marker"`
	NextStepMarker string                     `json:"next_step_marker"`
	TotalEntries   int64                      `json:"total_entries"`
	Position       int64                      `json:"pos"`
	Timestamp      string                     `json:"timestamp"`
	RealmEpoch     int64                      `json:"realm_epoch"`
}

// GetMetadataLogInfo retrieves metadata-log information through
// GET /admin/log?type=metadata.
//
// Ceph v20.2.4's RGWOp_MDLog_Info writes its normal JSON output after an error
// body in single-site deployments. The connection is therefore closed after
// this request so those extra bytes cannot corrupt a subsequent response.
func (client *Client) GetMetadataLogInfo(ctx context.Context) (MetadataLogInfo, error) {
	if ctx == nil {
		return MetadataLogInfo{}, errors.New("rgw: context must not be nil")
	}
	var result MetadataLogInfo
	err := client.getLogClosingConnection(ctx, url.Values{"type": {"metadata"}}, &result)
	return result, err
}

type GetMetadataLogShardInfoRequest struct {
	ShardID int64
	Period  string
}

// GetMetadataLogShardInfo retrieves one metadata-log shard's information.
// Ceph v20.2.4's RGWOp_MDLog_ShardInfo has the same malformed error-response
// behavior as RGWOp_MDLog_Info, so this request also closes its connection.
func (client *Client) GetMetadataLogShardInfo(ctx context.Context, input GetMetadataLogShardInfoRequest) (MetadataLogShardInfo, error) {
	if ctx == nil {
		return MetadataLogShardInfo{}, errors.New("rgw: context must not be nil")
	}
	if input.ShardID < 0 {
		return MetadataLogShardInfo{}, errors.New("rgw: metadata log shard ID must not be negative")
	}
	query := url.Values{"type": {"metadata"}, "info": {""}}
	setInt64(query, "id", &input.ShardID)
	setString(query, "period", input.Period)
	var result MetadataLogShardInfo
	err := client.getLogClosingConnection(ctx, query, &result)
	return result, err
}

type ListMetadataLogEntriesRequest struct {
	ShardID    int64
	Period     string
	Marker     string
	MaxEntries *int64
}

// ListMetadataLogEntries lists entries from one metadata-log shard.
func (client *Client) ListMetadataLogEntries(ctx context.Context, input ListMetadataLogEntriesRequest) (MetadataLogEntryList, error) {
	if ctx == nil {
		return MetadataLogEntryList{}, errors.New("rgw: context must not be nil")
	}
	if input.ShardID < 0 {
		return MetadataLogEntryList{}, errors.New("rgw: metadata log shard ID must not be negative")
	}
	query := url.Values{"type": {"metadata"}}
	setInt64(query, "id", &input.ShardID)
	setString(query, "period", input.Period)
	setString(query, "marker", input.Marker)
	setInt64(query, "max-entries", input.MaxEntries)
	var result MetadataLogEntryList
	err := client.getLog(ctx, query, &result)
	return result, err
}

// GetMetadataLogStatus retrieves metadata synchronization status.
func (client *Client) GetMetadataLogStatus(ctx context.Context) (MetadataLogSyncStatus, error) {
	if ctx == nil {
		return MetadataLogSyncStatus{}, errors.New("rgw: context must not be nil")
	}
	query := url.Values{"type": {"metadata"}, "status": {""}}
	var result MetadataLogSyncStatus
	err := client.getLog(ctx, query, &result)
	return result, err
}
