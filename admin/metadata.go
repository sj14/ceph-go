package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ObjectVersion is Ceph's obj_version representation shared by metadata and
// metadata-log responses.
type ObjectVersion struct {
	Tag     string `json:"tag"`
	Version int64  `json:"ver"`
}

// MetadataKeyList normalizes both metadata list response formats. Ceph returns
// a bare array unless MaxEntries is present and a paginated object otherwise.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/rgw_rest_metadata.cc
//   - src/rgw/rgw_rest_metadata.h
//   - src/rgw/rgw_metadata.cc
//   - src/rgw/driver/rados/rgw_metadata.h
type MetadataKeyList struct {
	Keys      []string `json:"keys"`
	Truncated bool     `json:"truncated"`
	Count     int64    `json:"count"`
	Marker    string   `json:"marker,omitempty"`
}

type ListMetadataKeysRequest struct {
	Section    string
	Marker     string
	MaxEntries *int64
}

// Metadata contains the common envelope around section-specific metadata.
// Data is an object whose fields depend on the requested metadata section.
type Metadata struct {
	Key              string         `json:"key"`
	Version          ObjectVersion  `json:"ver"`
	ModificationTime *time.Time     `json:"mtime,omitempty"`
	Data             map[string]any `json:"data"`
}

type GetMetadataRequest struct {
	Section string
	Key     string
}

type GetLocalMetadataRequest struct {
	Section string
}

// ListMetadataKeys lists metadata sections or keys within a section through
// GET /admin/metadata[/<section>].
func (client *Client) ListMetadataKeys(ctx context.Context, input ListMetadataKeysRequest) (MetadataKeyList, error) {
	if ctx == nil {
		return MetadataKeyList{}, errors.New("admin: context must not be nil")
	}
	query := url.Values{}
	setString(query, "marker", input.Marker)
	setInt64(query, "max-entries", input.MaxEntries)
	body, request, err := client.metadataRequest(ctx, input.Section, query)
	if err != nil {
		return MetadataKeyList{}, err
	}
	var result MetadataKeyList
	if input.MaxEntries == nil {
		if err := json.Unmarshal(body, &result.Keys); err != nil {
			return MetadataKeyList{}, fmt.Errorf("admin: decode %s %s response: %w", request.Method, request.URL.Path, err)
		}
		result.Count = int64(len(result.Keys))
		return result, nil
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return MetadataKeyList{}, fmt.Errorf("admin: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return result, nil
}

// GetMetadata retrieves one metadata object through
// GET /admin/metadata[/<section>]?key=....
func (client *Client) GetMetadata(ctx context.Context, input GetMetadataRequest) (Metadata, error) {
	if ctx == nil {
		return Metadata{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.Key) == "" {
		return Metadata{}, errors.New("admin: metadata key must not be empty")
	}
	return client.getMetadata(ctx, input.Section, url.Values{"key": {input.Key}})
}

// GetLocalMetadata retrieves the authenticated administrative user's metadata
// in a section through GET /admin/metadata/<section>?myself.
func (client *Client) GetLocalMetadata(ctx context.Context, input GetLocalMetadataRequest) (Metadata, error) {
	if ctx == nil {
		return Metadata{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.Section) == "" {
		return Metadata{}, errors.New("admin: metadata section must not be empty")
	}
	return client.getMetadata(ctx, input.Section, url.Values{"myself": {""}})
}

func (client *Client) getMetadata(ctx context.Context, section string, query url.Values) (Metadata, error) {
	body, request, err := client.metadataRequest(ctx, section, query)
	if err != nil {
		return Metadata{}, err
	}
	var result Metadata
	if err := json.Unmarshal(body, &result); err != nil {
		return Metadata{}, fmt.Errorf("admin: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return result, nil
}

func (client *Client) metadataRequest(ctx context.Context, section string, query url.Values) ([]byte, *http.Request, error) {
	section = strings.TrimSpace(section)
	if section == "." || section == ".." || strings.ContainsAny(section, "/?#") {
		return nil, nil, errors.New("admin: metadata section must not be a path segment, query, or fragment")
	}
	resource := "metadata"
	if section != "" {
		resource += "/" + section
	}
	request, err := client.newRequest(ctx, http.MethodGet, resource, query)
	if err != nil {
		return nil, nil, err
	}
	body, err := client.do(request)
	return body, request, err
}
