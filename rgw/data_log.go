package rgw

import (
	"context"
	"encoding/json"
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

// DataLogEntry normalizes Ceph's two list formats. LogID and LogTimestamp are
// populated only when ListDataLogEntriesRequest.ExtraInfo is true.
type DataLogEntry struct {
	LogID        string
	LogTimestamp string
	Change       DataLogChange
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

type GetDataLogShardInfoRequest struct {
	ShardID int64
}

type ListDataLogEntriesRequest struct {
	ShardID    int64
	Marker     string
	MaxEntries *int64
	ExtraInfo  *bool
}

type GetDataLogStatusRequest struct {
	SourceZone string
}

// UnmarshalJSON accepts both rgw_data_change and rgw_data_change_log_entry.
func (entry *DataLogEntry) UnmarshalJSON(data []byte) error {
	*entry = DataLogEntry{}
	var detailed struct {
		LogID        string          `json:"log_id"`
		LogTimestamp string          `json:"log_timestamp"`
		Change       json.RawMessage `json:"entry"`
	}
	if err := json.Unmarshal(data, &detailed); err != nil {
		return err
	}
	if detailed.Change != nil {
		entry.LogID = detailed.LogID
		entry.LogTimestamp = detailed.LogTimestamp
		return json.Unmarshal(detailed.Change, &entry.Change)
	}
	return json.Unmarshal(data, &entry.Change)
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

// ListDataLogEntries lists entries from one data-log shard.
func (client *Client) ListDataLogEntries(ctx context.Context, input ListDataLogEntriesRequest) (DataLogEntryList, error) {
	if ctx == nil {
		return DataLogEntryList{}, errors.New("rgw: context must not be nil")
	}
	if input.ShardID < 0 {
		return DataLogEntryList{}, errors.New("rgw: data log shard ID must not be negative")
	}
	query := url.Values{"type": {"data"}}
	setInt64(query, "id", &input.ShardID)
	setString(query, "marker", input.Marker)
	setInt64(query, "max-entries", input.MaxEntries)
	setBool(query, "extra-info", input.ExtraInfo)
	var result DataLogEntryList
	err := client.getLog(ctx, query, &result)
	return result, err
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
