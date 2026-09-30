package rgw

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// The log mutations below are verified against Ceph v20.2.4:
//   - src/rgw/driver/rados/rgw_rest_log.cc and .h (dispatch, parsing, caps)
//   - src/rgw/driver/rados/rgw_metadata.cc (metadata-log locks and trimming)
//   - src/rgw/services/svc_cls.cc (lock cookies/tags)
//   - src/rgw/services/svc_bilog_rados.cc (bucket-index trimming)
//   - src/rgw/driver/rados/rgw_datalog.cc and rgw_datalog_notify.cc
//   - src/common/ceph_json.h (map/set request encoding)

type LockMetadataLogRequest struct {
	ShardID       int64
	Period        string
	LockerID      string
	ZoneID        string
	LengthSeconds int64
}

// LockMetadataLog obtains or renews an exclusive metadata-log shard lock
// through POST /admin/log?type=metadata&lock. Period defaults to the current
// period. LockerID is the lock cookie and ZoneID is its tag. Requires mdlog=write.
func (client *Client) LockMetadataLog(ctx context.Context, input LockMetadataLogRequest) error {
	query, err := metadataLogLockQuery(input.ShardID, input.Period, input.LockerID, input.ZoneID)
	if err != nil {
		return err
	}
	if input.LengthSeconds <= 0 || input.LengthSeconds > 1<<32-1 {
		return errors.New("rgw: metadata log lock length must be between 1 and 4294967295 seconds")
	}
	query.Set("lock", "")
	setInt64(query, "length", &input.LengthSeconds)
	return client.logAction(ctx, http.MethodPost, query, nil)
}

type UnlockMetadataLogRequest struct {
	ShardID  int64
	Period   string
	LockerID string
	ZoneID   string
}

// UnlockMetadataLog releases a lock using its cookie and tag through
// POST /admin/log?type=metadata&unlock. Requires mdlog=write.
func (client *Client) UnlockMetadataLog(ctx context.Context, input UnlockMetadataLogRequest) error {
	query, err := metadataLogLockQuery(input.ShardID, input.Period, input.LockerID, input.ZoneID)
	if err != nil {
		return err
	}
	query.Set("unlock", "")
	return client.logAction(ctx, http.MethodPost, query, nil)
}

func metadataLogLockQuery(shard int64, period, locker, zone string) (url.Values, error) {
	query, err := logShardQuery("metadata", shard)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(locker) == "" || strings.TrimSpace(zone) == "" {
		return nil, errors.New("rgw: metadata log locker ID and zone ID must not be empty")
	}
	setString(query, "period", period)
	query.Set("locker-id", locker)
	query.Set("zone-id", zone)
	return query, nil
}

type NotifyMetadataLogRequest struct {
	UpdatedShards []int64
}

// NotifyMetadataLog wakes metadata-sync workers through
// POST /admin/log?type=metadata&notify, with a JSON array of shard IDs.
// Requires mdlog=write.
func (client *Client) NotifyMetadataLog(ctx context.Context, input NotifyMetadataLogRequest) error {
	shards := input.UpdatedShards
	if shards == nil {
		shards = []int64{}
	}
	for _, shard := range shards {
		if shard < 0 || shard > 1<<31-1 {
			return errors.New("rgw: notification shard IDs must be between 0 and 2147483647")
		}
	}
	return client.logAction(ctx, http.MethodPost, url.Values{"type": {"metadata"}, "notify": {""}}, shards)
}

type TrimMetadataLogRequest struct {
	ShardID int64
	Period  string
	Marker  string
}

// TrimMetadataLog deletes entries through Marker in one metadata-log shard
// using DELETE /admin/log?type=metadata. Period defaults to the current period.
// Requires mdlog=write. Marker is the bounding end marker.
func (client *Client) TrimMetadataLog(ctx context.Context, input TrimMetadataLogRequest) error {
	query, err := logTrimQuery("metadata", input.ShardID, input.Marker)
	if err != nil {
		return err
	}
	setString(query, "period", input.Period)
	return client.logAction(ctx, http.MethodDelete, query, nil)
}

type TrimBucketIndexLogRequest struct {
	Tenant         string
	Bucket         string
	BucketInstance string
	StartMarker    string
	EndMarker      string
	Generation     *int64
}

// TrimBucketIndexLog deletes a range of bucket-index log entries through
// DELETE /admin/log?type=bucket-index. EndMarker is required; StartMarker
// defaults to the beginning and Generation to zero. As with bucket-index log
// reads, Ceph v20.2.4 requires BucketInstance. Requires bilog=write.
func (client *Client) TrimBucketIndexLog(ctx context.Context, input TrimBucketIndexLogRequest) error {
	if strings.TrimSpace(input.BucketInstance) == "" || strings.TrimSpace(input.EndMarker) == "" {
		return errors.New("rgw: bucket instance and end marker must not be empty")
	}
	if input.Generation != nil && *input.Generation < 0 {
		return errors.New("rgw: bucket-index log generation must not be negative")
	}
	query := bucketIndexLogQuery(input.Tenant, input.Bucket, input.BucketInstance)
	setString(query, "start-marker", input.StartMarker)
	query.Set("end-marker", input.EndMarker)
	setInt64(query, "generation", input.Generation)
	return client.logAction(ctx, http.MethodDelete, query, nil)
}

// DataLogShardNotification uses the original notify API's key-only entries.
// Ceph treats these entries as generation zero.
type DataLogShardNotification struct {
	ShardID int64    `json:"key"`
	Keys    []string `json:"val"`
}

type NotifyDataLogRequest struct {
	SourceZone string
	Shards     []DataLogShardNotification
}

// NotifyDataLog wakes data-sync workers through
// POST /admin/log?type=data&notify. Requires datalog=write.
func (client *Client) NotifyDataLog(ctx context.Context, input NotifyDataLogRequest) error {
	shards := input.Shards
	if shards == nil {
		shards = []DataLogShardNotification{}
	}
	for _, shard := range shards {
		if shard.ShardID < 0 || shard.ShardID > 1<<31-1 {
			return errors.New("rgw: notification shard IDs must be between 0 and 2147483647")
		}
	}
	query := url.Values{"type": {"data"}, "notify": {""}}
	setString(query, "source-zone", input.SourceZone)
	return client.logAction(ctx, http.MethodPost, query, shards)
}

// DataLogNotificationEntry is rgw_data_notify_entry. It identifies a bucket
// shard and its log generation; it is distinct from the data-log change model.
type DataLogNotificationEntry struct {
	Key        string `json:"key"`
	Generation uint64 `json:"gen"`
}

type DataLogShardNotificationV2 struct {
	ShardID int64                      `json:"key"`
	Entries []DataLogNotificationEntry `json:"val"`
}

type NotifyDataLogV2Request struct {
	SourceZone string
	Shards     []DataLogShardNotificationV2
}

// NotifyDataLogV2 wakes data-sync workers through
// POST /admin/log?type=data&notify2, preserving each entry's generation.
// Requires datalog=write.
func (client *Client) NotifyDataLogV2(ctx context.Context, input NotifyDataLogV2Request) error {
	shards := input.Shards
	if shards == nil {
		shards = []DataLogShardNotificationV2{}
	}
	for _, shard := range shards {
		if shard.ShardID < 0 || shard.ShardID > 1<<31-1 {
			return errors.New("rgw: notification shard IDs must be between 0 and 2147483647")
		}
	}
	query := url.Values{"type": {"data"}, "notify2": {""}}
	setString(query, "source-zone", input.SourceZone)
	return client.logAction(ctx, http.MethodPost, query, shards)
}

type TrimDataLogRequest struct {
	ShardID int64
	Marker  string
}

// TrimDataLog deletes entries through Marker in one data-log shard using
// DELETE /admin/log?type=data. Requires datalog=write.
func (client *Client) TrimDataLog(ctx context.Context, input TrimDataLogRequest) error {
	query, err := logTrimQuery("data", input.ShardID, input.Marker)
	if err != nil {
		return err
	}
	return client.logAction(ctx, http.MethodDelete, query, nil)
}

func logShardQuery(logType string, shard int64) (url.Values, error) {
	if shard < 0 || shard > 1<<31-1 {
		return nil, errors.New("rgw: log shard ID must be between 0 and 2147483647")
	}
	query := url.Values{"type": {logType}}
	setInt64(query, "id", &shard)
	return query, nil
}

func logTrimQuery(logType string, shard int64, marker string) (url.Values, error) {
	query, err := logShardQuery(logType, shard)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(marker) == "" {
		return nil, errors.New("rgw: log trim marker must not be empty")
	}
	query.Set("marker", marker)
	return query, nil
}

func (client *Client) logAction(ctx context.Context, method string, query url.Values, input any) error {
	var request *http.Request
	var err error
	if input == nil {
		request, err = client.newRequest(ctx, method, "log", query)
	} else {
		request, err = client.newJSONRequest(ctx, method, "log", query, input)
	}
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}
