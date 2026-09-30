package rgw

import "testing"

func TestParseMetadataUpdateVersion(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		header string
		want   ObjectVersion
	}{
		{"ver:0,tag:", ObjectVersion{}},
		{"ver:42,tag:version-tag", ObjectVersion{Version: 42, Tag: "version-tag"}},
		{"ver:7,tag:tag,with:punctuation", ObjectVersion{Version: 7, Tag: "tag,with:punctuation"}},
	} {
		got, err := parseMetadataUpdateVersion(test.header)
		if err != nil || got != test.want {
			t.Errorf("parse %q = %#v, %v; want %#v", test.header, got, err, test.want)
		}
	}
	for _, header := range []string{"", "ver:1", "version:1,tag:t", "ver:-1,tag:t", "ver:x,tag:t", "ver:9223372036854775808,tag:t"} {
		if _, err := parseMetadataUpdateVersion(header); err == nil {
			t.Errorf("parse %q succeeded", header)
		}
	}
}
