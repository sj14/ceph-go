package rgw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataLogInfoRequestsCloseConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(*Client) error
	}{
		{
			name: "log info",
			call: func(client *Client) error {
				_, err := client.GetMetadataLogInfo(context.Background())
				return err
			},
		},
		{
			name: "shard info",
			call: func(client *Client) error {
				_, err := client.GetMetadataLogShardInfo(
					context.Background(), GetMetadataLogShardInfoRequest{ShardID: 0},
				)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if !request.Close {
					t.Error("request permits connection reuse")
				}
				_, _ = writer.Write([]byte(`{"num_objects":64,"marker":"","last_update":"0.000000"}`))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "access-key", "secret-key")
			if err != nil {
				t.Fatal(err)
			}
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}
