package integration

import "testing"

const testClusterFSID = "00000000-0000-0000-0000-000000000001"

func TestGetFullHealth(t *testing.T) {
	t.Parallel()

	report, err := integrationClient(t).GetFullHealth(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if report.Health.Status == "" || report.DF == nil || report.PGInfo == nil ||
		report.RGW == nil || *report.RGW != 1 {
		t.Fatalf("full health report = %#v", report)
	}
}

func TestGetMinimalHealth(t *testing.T) {
	t.Parallel()

	report, err := integrationClient(t).GetMinimalHealth(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if report.Health.Status == "" || report.ClientPerf == nil || report.DF == nil ||
		report.RGW == nil || *report.RGW != 1 {
		t.Fatalf("minimal health report = %#v", report)
	}
}

func TestGetClusterCapacity(t *testing.T) {
	t.Parallel()

	capacity, err := integrationClient(t).GetClusterCapacity(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if capacity.TotalBytes <= 0 || capacity.TotalAvailableBytes < 0 ||
		capacity.TotalUsedRawBytes < 0 {
		t.Fatalf("cluster capacity = %#v", capacity)
	}
}

func TestGetClusterFSID(t *testing.T) {
	t.Parallel()

	fsid, err := integrationClient(t).GetClusterFSID(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if fsid != testClusterFSID {
		t.Fatalf("cluster FSID = %q, want %q", fsid, testClusterFSID)
	}
}

func TestGetTelemetryStatus(t *testing.T) {
	t.Parallel()

	enabled, err := integrationClient(t).GetTelemetryStatus(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("telemetry is enabled, want disabled test-cluster default")
	}
}

func TestGetHealthSnapshot(t *testing.T) {
	t.Parallel()

	snapshot, err := integrationClient(t).GetHealthSnapshot(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.FSID != testClusterFSID || snapshot.Health.Status == "" ||
		snapshot.OSDMap == nil || snapshot.OSDMap.NumOSDs != 1 ||
		snapshot.NumRGWGateways == nil || *snapshot.NumRGWGateways != 1 {
		t.Fatalf("health snapshot = %#v", snapshot)
	}
}
