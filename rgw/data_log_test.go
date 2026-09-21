package rgw

import (
	"encoding/json"
	"testing"
)

func TestDataLogEntryAcceptsBothCephFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		body         string
		wantLogID    string
		wantKey      string
		wantDetailed bool
	}{
		{
			name:    "change",
			body:    `{"entity_type":"bucket","key":"bucket:instance","timestamp":"0.000000","gen":2}`,
			wantKey: "bucket:instance",
		},
		{
			name:         "change with log metadata",
			body:         `{"log_id":"log-1","log_timestamp":"0.000000","entry":{"entity_type":"bucket","key":"bucket:instance","timestamp":"0.000000","gen":2}}`,
			wantLogID:    "log-1",
			wantKey:      "bucket:instance",
			wantDetailed: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var entry DataLogEntry
			if err := json.Unmarshal([]byte(test.body), &entry); err != nil {
				t.Fatal(err)
			}
			if entry.LogID != test.wantLogID || entry.Change.Key != test.wantKey ||
				(entry.LogTimestamp != "") != test.wantDetailed {
				t.Fatalf("data log entry = %#v", entry)
			}
		})
	}
}
