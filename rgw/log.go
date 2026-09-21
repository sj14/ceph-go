package rgw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// LogSyncState is the status of metadata or data synchronization.
type LogSyncState string

const (
	LogSyncStateInit                 LogSyncState = "init"
	LogSyncStateBuildingFullSyncMaps LogSyncState = "building-full-sync-maps"
	LogSyncStateSync                 LogSyncState = "sync"
	LogSyncStateUnknown              LogSyncState = "unknown"
)

// LogShardSyncState is the status of an individual data or bucket-index log
// synchronization shard.
type LogShardSyncState string

const (
	LogShardSyncStateInit        LogShardSyncState = "init"
	LogShardSyncStateFull        LogShardSyncState = "full-sync"
	LogShardSyncStateIncremental LogShardSyncState = "incremental-sync"
	LogShardSyncStateStopped     LogShardSyncState = "stopped"
	LogShardSyncStateUnknown     LogShardSyncState = "unknown"
)

func (client *Client) getLog(ctx context.Context, query url.Values, output any) error {
	request, err := client.newRequest(ctx, http.MethodGet, "log", query)
	if err != nil {
		return err
	}
	body, err := client.do(request)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, output); err != nil {
		return fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return nil
}
