package rgw

import (
	"context"
	"strings"
	"testing"
)

func TestTrimUsageRequiresExplicitRemoveAll(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://rgw.example", "access-key", "secret-key")
	if err != nil {
		t.Fatal(err)
	}
	err = client.TrimUsage(context.Background(), TrimUsageRequest{})
	if err == nil || !strings.Contains(err.Error(), "remove-all") {
		t.Fatalf("error = %v, want remove-all validation error", err)
	}
}
