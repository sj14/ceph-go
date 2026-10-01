package rgw

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// MetadataUpdateType selects the update policy passed to the metadata handler.
// The default is always. Ceph v20.2.4's user handler ignores this policy.
type MetadataUpdateType string

const (
	MetadataUpdateAlways      MetadataUpdateType = "always"
	MetadataUpdateByVersion   MetadataUpdateType = "update-by-version"
	MetadataUpdateByTimestamp MetadataUpdateType = "update-by-timestamp"
)

type MetadataUpdateStatus string

const (
	MetadataUpdateApplied MetadataUpdateStatus = "applied"
	MetadataUpdateSkipped MetadataUpdateStatus = "skipped"
)

// MetadataUpdate contains Ceph's RGWX_UPDATE_STATUS and RGWX_UPDATE_VERSION.
// Version describes the existing on-disk version, not necessarily the incoming
// version. Status can be empty for handlers that do not report an apply result.
//
// Verified against Ceph v20.2.4: src/rgw/rgw_rest_metadata.cc and .h,
// src/rgw/rgw_metadata.cc (RGWMetadataManager::put), and
// src/rgw/driver/rados/rgw_user.cc (user metadata handler).
type MetadataUpdate struct {
	Status  MetadataUpdateStatus
	Version ObjectVersion
}

type PutMetadataRequest struct {
	Section    MetadataSection
	Key        string
	Metadata   Metadata
	UpdateType MetadataUpdateType
}

// PutMetadata writes section-specific metadata through
// PUT /admin/metadata[/<section>]?key=.... Requires metadata=write. The query
// selects the destination; Metadata supplies the key/ver/mtime/data envelope.
func (client *Client) PutMetadata(ctx context.Context, input PutMetadataRequest) (MetadataUpdate, error) {
	if ctx == nil {
		return MetadataUpdate{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Key) == "" {
		return MetadataUpdate{}, errors.New("rgw: metadata key must not be empty")
	}
	resource, err := metadataResource(input.Section)
	if err != nil {
		return MetadataUpdate{}, err
	}
	query := url.Values{"key": {input.Key}}
	setString(query, "update-type", string(input.UpdateType))
	request, err := client.newJSONRequest(ctx, http.MethodPut, resource, query, input.Metadata)
	if err != nil {
		return MetadataUpdate{}, err
	}
	_, headers, err := client.doWithHeaders(request)
	if err != nil {
		return MetadataUpdate{}, err
	}
	version, err := parseMetadataUpdateVersion(headers.Get("RGWX_UPDATE_VERSION"))
	if err != nil {
		return MetadataUpdate{}, fmt.Errorf("rgw: decode metadata update version: %w", err)
	}
	return MetadataUpdate{
		Status: MetadataUpdateStatus(headers.Get("RGWX_UPDATE_STATUS")), Version: version,
	}, nil
}

type DeleteMetadataRequest struct {
	Section MetadataSection
	Key     string
}

// DeleteMetadata removes metadata through
// DELETE /admin/metadata[/<section>]?key=.... Requires metadata=write.
func (client *Client) DeleteMetadata(ctx context.Context, input DeleteMetadataRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Key) == "" {
		return errors.New("rgw: metadata key must not be empty")
	}
	resource, err := metadataResource(input.Section)
	if err != nil {
		return err
	}
	request, err := client.newRequest(ctx, http.MethodDelete, resource, url.Values{"key": {input.Key}})
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

func parseMetadataUpdateVersion(header string) (ObjectVersion, error) {
	number, tag, ok := strings.Cut(header, ",tag:")
	if !ok || !strings.HasPrefix(number, "ver:") {
		return ObjectVersion{}, fmt.Errorf("invalid RGWX_UPDATE_VERSION %q", header)
	}
	version, err := strconv.ParseInt(strings.TrimPrefix(number, "ver:"), 10, 64)
	if err != nil || version < 0 {
		return ObjectVersion{}, fmt.Errorf("invalid RGWX_UPDATE_VERSION %q", header)
	}
	return ObjectVersion{Version: version, Tag: tag}, nil
}
