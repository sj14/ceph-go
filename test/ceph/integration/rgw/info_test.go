package integration

import "testing"

func TestRGWGetGatewayInfo(t *testing.T) {
	t.Parallel()

	info, err := rgwIntegrationClient(t).GetGatewayInfo(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(info.StorageBackends) != 1 || info.StorageBackends[0].Name != "rados" ||
		info.StorageBackends[0].ClusterID == "" {
		t.Fatalf("Admin Ops gateway info = %#v", info)
	}
}
