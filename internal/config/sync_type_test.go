package config

import (
	"reflect"
	"testing"
)

func TestParseSyncTypes(t *testing.T) {
	cases := []struct {
		name     string
		nested   map[string]interface{}
		want     []string
		wantWarn bool
	}{
		{
			name:   "absent yields nil",
			nested: map[string]interface{}{},
			want:   nil,
		},
		{
			name:   "empty list yields empty",
			nested: map[string]interface{}{"sync_type": []interface{}{}},
			want:   nil,
		},
		{
			name:   "single type",
			nested: map[string]interface{}{"sync_type": []interface{}{"ap"}},
			want:   []string{"ap"},
		},
		{
			name:   "normalizes case and whitespace",
			nested: map[string]interface{}{"sync_type": []interface{}{"AP", " switch "}},
			want:   []string{"ap", "switch"},
		},
		{
			name:   "dedupes",
			nested: map[string]interface{}{"sync_type": []interface{}{"ap", "ap", "gateway"}},
			want:   []string{"ap", "gateway"},
		},
		{
			name:     "unknown type warns and is skipped",
			nested:   map[string]interface{}{"sync_type": []interface{}{"bogus", "ap"}},
			want:     []string{"ap"},
			wantWarn: true,
		},
		{
			name:     "wrong type warns",
			nested:   map[string]interface{}{"sync_type": "ap"},
			want:     nil,
			wantWarn: true,
		},
		{
			name:   "tolerates []string",
			nested: map[string]interface{}{"sync_type": []string{"gateway"}},
			want:   []string{"gateway"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, warnings := parseSyncTypes("test-api", c.nested)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
			if (len(warnings) > 0) != c.wantWarn {
				t.Errorf("warnings = %v, wantWarn = %v", warnings, c.wantWarn)
			}
		})
	}
}
