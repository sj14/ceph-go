package rgw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// HealthStatus is Ceph's overall health state. The values come from Ceph
// v20.2.4's src/include/health.h (health_status_t).
type HealthStatus string

const (
	HealthStatusOK   HealthStatus = "HEALTH_OK"
	HealthStatusWarn HealthStatus = "HEALTH_WARN"
	HealthStatusErr  HealthStatus = "HEALTH_ERR"
)

// ScrubStatus describes whether cluster scrubbing is active or disabled. The
// values come from Ceph v20.2.4's dashboard CephService.
type ScrubStatus string

const (
	ScrubStatusDisabled ScrubStatus = "Disabled"
	ScrubStatusActive   ScrubStatus = "Active"
	ScrubStatusInactive ScrubStatus = "Inactive"
)

// HealthCheckSummary summarizes one Ceph health check.
type HealthCheckSummary struct {
	Message string `json:"message"`
	Count   int64  `json:"count"`
}

// HealthCheckDetail contains one detailed health-check message.
type HealthCheckDetail struct {
	Message string `json:"message"`
}

// HealthCheck contains one warning or error reported by Ceph. Type is filled
// by the full and minimal endpoints; the snapshot endpoint uses it as the map
// key instead.
type HealthCheck struct {
	Type     string              `json:"type"`
	Severity HealthStatus        `json:"severity"`
	Summary  HealthCheckSummary  `json:"summary"`
	Detail   []HealthCheckDetail `json:"detail"`
	Muted    bool                `json:"muted"`
}

// HealthReportStatus is the health section returned by the full and minimal
// endpoints. Mute entries are retained as JSON because their shape is managed
// by Ceph's monitor and is not defined by the Dashboard controller.
type HealthReportStatus struct {
	Status HealthStatus      `json:"status"`
	Checks []HealthCheck     `json:"checks"`
	Mutes  []json.RawMessage `json:"mutes"`
}

// ClientPerformance contains aggregate client and recovery rates.
type ClientPerformance struct {
	ReadBytesPerSecond       int64 `json:"read_bytes_sec"`
	ReadOperationsPerSecond  int64 `json:"read_op_per_sec"`
	WriteBytesPerSecond      int64 `json:"write_bytes_sec"`
	WriteOperationsPerSecond int64 `json:"write_op_per_sec"`
	RecoveringBytesPerSecond int64 `json:"recovering_bytes_per_sec"`
}

// ClusterCapacity contains the raw capacity totals returned by Ceph.
type ClusterCapacity struct {
	TotalAvailableBytes int64 `json:"total_avail_bytes"`
	TotalBytes          int64 `json:"total_bytes"`
	TotalUsedRawBytes   int64 `json:"total_used_raw_bytes"`
}

// HealthDataFrame is the stable portion of the df section shared by full and
// minimal health reports. The full endpoint may return additional fields.
type HealthDataFrame struct {
	Stats ClusterCapacity `json:"stats"`
}

// PlacementGroupObjectStats contains cluster-wide object counters.
type PlacementGroupObjectStats struct {
	NumObjects          int64 `json:"num_objects"`
	NumObjectCopies     int64 `json:"num_object_copies"`
	NumObjectsDegraded  int64 `json:"num_objects_degraded"`
	NumObjectsMisplaced int64 `json:"num_objects_misplaced"`
	NumObjectsUnfound   int64 `json:"num_objects_unfound"`
}

// PlacementGroupInfo contains placement-group counts and states.
type PlacementGroupInfo struct {
	ObjectStats PlacementGroupObjectStats `json:"object_stats"`
	PGsPerOSD   float64                   `json:"pgs_per_osd"`
	Statuses    map[string]int64          `json:"statuses"`
}

// GatewayStatus contains available and unavailable gateway counts.
type GatewayStatus struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// HealthReport is returned by both full and minimal health endpoints.
// Permission- and module-dependent sections are absent when the authenticated
// Dashboard user cannot access them. The large Ceph maps remain raw so the
// client does not discard version-specific data.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/health.py (HealthData and Health)
//   - src/pybind/mgr/dashboard/services/ceph_service.py
//   - src/pybind/mgr/dashboard/services/cluster.py (ClusterCapacity)
//   - src/pybind/mgr/dashboard/controllers/_endpoint.py
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/health.service.ts
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/models/health.interface.ts
//   - qa/tasks/mgr/dashboard/test_health.py
type HealthReport struct {
	Health        HealthReportStatus  `json:"health"`
	ClientPerf    *ClientPerformance  `json:"client_perf,omitempty"`
	DF            *HealthDataFrame    `json:"df,omitempty"`
	FSMap         json.RawMessage     `json:"fs_map,omitempty"`
	ManagerMap    json.RawMessage     `json:"mgr_map,omitempty"`
	MonitorStatus json.RawMessage     `json:"mon_status,omitempty"`
	OSDMap        json.RawMessage     `json:"osd_map,omitempty"`
	PGInfo        *PlacementGroupInfo `json:"pg_info,omitempty"`
	Pools         []json.RawMessage   `json:"pools,omitempty"`
	Hosts         *int64              `json:"hosts,omitempty"`
	RGW           *int64              `json:"rgw,omitempty"`
	ISCSIDaemons  *GatewayStatus      `json:"iscsi_daemons,omitempty"`
	ScrubStatus   *ScrubStatus        `json:"scrub_status,omitempty"`
}

// HealthSnapshotStatus is the health section of a cluster snapshot.
type HealthSnapshotStatus struct {
	Status HealthStatus           `json:"status"`
	Checks map[string]HealthCheck `json:"checks"`
	Mutes  []json.RawMessage      `json:"mutes"`
}

// HealthSnapshotMonitorMap summarizes monitor quorum.
type HealthSnapshotMonitorMap struct {
	NumMonitors int64   `json:"num_mons"`
	Quorum      []int64 `json:"quorum"`
}

// HealthSnapshotOSDMap summarizes OSD availability.
type HealthSnapshotOSDMap struct {
	In      int64 `json:"in"`
	Up      int64 `json:"up"`
	NumOSDs int64 `json:"num_osds"`
}

// PlacementGroupStateCount contains the number of placement groups in a
// particular state.
type PlacementGroupStateCount struct {
	State string `json:"state_name"`
	Count int64  `json:"count"`
}

// HealthSnapshotPGMap summarizes placement-group and capacity state.
type HealthSnapshotPGMap struct {
	PGsByState               []PlacementGroupStateCount `json:"pgs_by_state"`
	NumPools                 int64                      `json:"num_pools"`
	NumPGs                   int64                      `json:"num_pgs"`
	BytesUsed                int64                      `json:"bytes_used"`
	BytesTotal               int64                      `json:"bytes_total"`
	WriteBytesPerSecond      int64                      `json:"write_bytes_sec"`
	ReadBytesPerSecond       int64                      `json:"read_bytes_sec"`
	RecoveringBytesPerSecond int64                      `json:"recovering_bytes_per_sec"`
}

// HealthSnapshotServiceMap contains active and standby daemon counts.
type HealthSnapshotServiceMap struct {
	NumActive   int64 `json:"num_active"`
	NumStandbys int64 `json:"num_standbys"`
}

// HealthSnapshot is the compact cluster overview returned by Ceph Dashboard.
// Sections guarded by Dashboard permissions are pointers so absence remains
// distinguishable from zero counts.
type HealthSnapshot struct {
	FSID              string                    `json:"fsid"`
	Health            HealthSnapshotStatus      `json:"health"`
	MonitorMap        *HealthSnapshotMonitorMap `json:"monmap,omitempty"`
	OSDMap            *HealthSnapshotOSDMap     `json:"osdmap,omitempty"`
	PGMap             *HealthSnapshotPGMap      `json:"pgmap,omitempty"`
	ManagerMap        *HealthSnapshotServiceMap `json:"mgrmap,omitempty"`
	FileSystemMap     *HealthSnapshotServiceMap `json:"fsmap,omitempty"`
	NumRGWGateways    *int64                    `json:"num_rgw_gateways,omitempty"`
	NumISCSIGateways  *GatewayStatus            `json:"num_iscsi_gateways,omitempty"`
	NumHosts          *int64                    `json:"num_hosts,omitempty"`
	NumAvailableHosts *int64                    `json:"num_hosts_available,omitempty"`
}

// GetFullHealth retrieves the detailed permission-filtered cluster health
// report through GET /api/health/full.
func (client *Client) GetFullHealth(ctx context.Context) (HealthReport, error) {
	if ctx == nil {
		return HealthReport{}, errors.New("rgw: context must not be nil")
	}

	var report HealthReport
	err := client.getHealth(ctx, "api/health/full", &report)
	return report, err
}

// GetMinimalHealth retrieves the reduced permission-filtered cluster health
// report through GET /api/health/minimal.
func (client *Client) GetMinimalHealth(ctx context.Context) (HealthReport, error) {
	if ctx == nil {
		return HealthReport{}, errors.New("rgw: context must not be nil")
	}

	var report HealthReport
	err := client.getHealth(ctx, "api/health/minimal", &report)
	return report, err
}

// GetClusterCapacity retrieves cluster capacity through
// GET /api/health/get_cluster_capacity.
func (client *Client) GetClusterCapacity(ctx context.Context) (ClusterCapacity, error) {
	if ctx == nil {
		return ClusterCapacity{}, errors.New("rgw: context must not be nil")
	}

	var capacity ClusterCapacity
	err := client.getHealth(ctx, "api/health/get_cluster_capacity", &capacity)
	return capacity, err
}

// GetClusterFSID retrieves the cluster FSID through
// GET /api/health/get_cluster_fsid.
func (client *Client) GetClusterFSID(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errors.New("rgw: context must not be nil")
	}

	var fsid string
	err := client.getHealth(ctx, "api/health/get_cluster_fsid", &fsid)
	return fsid, err
}

// GetTelemetryStatus reports whether Ceph telemetry is enabled through
// GET /api/health/get_telemetry_status.
func (client *Client) GetTelemetryStatus(ctx context.Context) (bool, error) {
	if ctx == nil {
		return false, errors.New("rgw: context must not be nil")
	}

	var enabled bool
	err := client.getHealth(ctx, "api/health/get_telemetry_status", &enabled)
	return enabled, err
}

// GetHealthSnapshot retrieves a compact cluster overview through
// GET /api/health/snapshot.
func (client *Client) GetHealthSnapshot(ctx context.Context) (HealthSnapshot, error) {
	if ctx == nil {
		return HealthSnapshot{}, errors.New("rgw: context must not be nil")
	}

	var snapshot HealthSnapshot
	err := client.getHealth(ctx, "api/health/snapshot", &snapshot)
	return snapshot, err
}

func (client *Client) getHealth(ctx context.Context, path string, result any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.endpoint(path).String(), nil)
	if err != nil {
		return err
	}
	body, err := client.do(request)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("rgw: decode GET %s response: %w", request.URL.Path, err)
	}
	return nil
}
