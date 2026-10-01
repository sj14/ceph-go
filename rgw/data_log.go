package rgw

import (
	"context"
	"errors"
	"net/url"
)

// DataLogEntityType identifies the entity recorded by a data-log change.
type DataLogEntityType string

const (
	DataLogEntityBucket  DataLogEntityType = "bucket"
	DataLogEntityUnknown DataLogEntityType = "unknown"
)

// DataLogInfo describes the data log as a whole.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_log.cc
//   - src/rgw/driver/rados/rgw_rest_log.h
//   - src/rgw/driver/rados/rgw_datalog.h
//   - src/rgw/driver/rados/rgw_datalog.cc
//   - src/rgw/driver/rados/rgw_data_sync.h
type DataLogInfo struct {
	NumShards int64 `json:"num_objects"`
}

type DataLogShardInfo struct {
	Marker     string `json:"marker"`
	LastUpdate string `json:"last_update"`
}

type DataLogEntryList struct {
	Marker     string         `json:"marker"`
	LastUpdate string         `json:"last_update"`
	Truncated  bool           `json:"truncated"`
	Entries    []DataLogEntry `json:"entries"`
}

// DataLogChangeList is a page of changes without per-entry log metadata.
// Marker is the opaque continuation token for the next request when Truncated
// is true; it is not a change's bucket key.
type DataLogChangeList struct {
	Marker     string          `json:"marker"`
	LastUpdate string          `json:"last_update"`
	Truncated  bool            `json:"truncated"`
	Entries    []DataLogChange `json:"entries"`
}

// DataLogEntry contains a change and its log metadata, always requested by
// ListDataLogEntries. Verified against Ceph v20.2.4's
// src/rgw/driver/rados/rgw_datalog.cc (rgw_data_change_log_entry::dump).
type DataLogEntry struct {
	LogID        string        `json:"log_id"`
	LogTimestamp string        `json:"log_timestamp"`
	Change       DataLogChange `json:"entry"`
}

type DataLogChange struct {
	EntityType DataLogEntityType `json:"entity_type"`
	Key        string            `json:"key"`
	Timestamp  string            `json:"timestamp"`
	Generation int64             `json:"gen"`
}

type DataLogSyncStatus struct {
	Info    DataLogSyncInfo              `json:"info"`
	Markers map[string]DataLogSyncMarker `json:"markers"`
}

type DataLogSyncInfo struct {
	Status     LogSyncState `json:"status"`
	NumShards  int64        `json:"num_shards"`
	InstanceID int64        `json:"instance_id"`
}

type DataLogSyncMarker struct {
	Status         LogShardSyncState `json:"status"`
	Marker         string            `json:"marker"`
	NextStepMarker string            `json:"next_step_marker"`
	TotalEntries   int64             `json:"total_entries"`
	Position       int64             `json:"pos"`
	Timestamp      string            `json:"timestamp"`
}

// GetDataLogInfo retrieves data-log information through
// GET /admin/log?type=data.
func (client *Client) GetDataLogInfo(ctx context.Context) (DataLogInfo, error) {
	if ctx == nil {
		return DataLogInfo{}, errors.New("rgw: context must not be nil")
	}
	var result DataLogInfo
	err := client.getLog(ctx, url.Values{"type": {"data"}}, &result)
	return result, err
}

type GetDataLogShardInfoRequest struct {
	ShardID int64
}

// GetDataLogShardInfo retrieves one data-log shard's information.
func (client *Client) GetDataLogShardInfo(ctx context.Context, input GetDataLogShardInfoRequest) (DataLogShardInfo, error) {
	if ctx == nil {
		return DataLogShardInfo{}, errors.New("rgw: context must not be nil")
	}
	if input.ShardID < 0 {
		return DataLogShardInfo{}, errors.New("rgw: data log shard ID must not be negative")
	}
	query := url.Values{"type": {"data"}, "info": {""}}
	setInt64(query, "id", &input.ShardID)
	var result DataLogShardInfo
	err := client.getLog(ctx, query, &result)
	return result, err
}

// ListDataLogEntriesRequest filters both detailed entry and change listings.
// Pass the previous page's Marker unchanged to continue a truncated listing.
type ListDataLogEntriesRequest struct {
	ShardID    int64
	Marker     string
	MaxEntries *int64
}

// ListDataLogEntries lists changes with their log IDs and log timestamps from
// one shard. It always sets extra-info=true. Ceph v20.2.4's
// src/rgw/driver/rados/rgw_rest_log.cc (RGWOp_DATALog_List::send_response)
// selects rgw_data_change_log_entry for this format.
func (client *Client) ListDataLogEntries(ctx context.Context, input ListDataLogEntriesRequest) (DataLogEntryList, error) {
	var result DataLogEntryList
	err := client.listDataLog(ctx, input, true, &result)
	return result, err
}

// ListDataLogChanges lists changes without per-entry log IDs or log timestamps
// from one shard. It always sets extra-info=false, selecting rgw_data_change
// in Ceph v20.2.4's RGWOp_DATALog_List::send_response.
func (client *Client) ListDataLogChanges(ctx context.Context, input ListDataLogEntriesRequest) (DataLogChangeList, error) {
	var result DataLogChangeList
	err := client.listDataLog(ctx, input, false, &result)
	return result, err
}

func (client *Client) listDataLog(ctx context.Context, input ListDataLogEntriesRequest, extraInfo bool, result any) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if input.ShardID < 0 {
		return errors.New("rgw: data log shard ID must not be negative")
	}
	query := url.Values{"type": {"data"}}
	setInt64(query, "id", &input.ShardID)
	setString(query, "marker", input.Marker)
	setInt64(query, "max-entries", input.MaxEntries)
	setBool(query, "extra-info", &extraInfo)
	return client.getLog(ctx, query, result)
}

type GetDataLogStatusRequest struct {
	SourceZone string
}

// GetDataLogStatus retrieves data synchronization status for a source zone.
func (client *Client) GetDataLogStatus(ctx context.Context, input GetDataLogStatusRequest) (DataLogSyncStatus, error) {
	if ctx == nil {
		return DataLogSyncStatus{}, errors.New("rgw: context must not be nil")
	}
	query := url.Values{"type": {"data"}, "status": {""}}
	setString(query, "source-zone", input.SourceZone)
	var result DataLogSyncStatus
	err := client.getLog(ctx, query, &result)
	return result, err
}
