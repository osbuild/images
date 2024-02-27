package osbuild

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/container"
)

func TestSource_UnmarshalJSON(t *testing.T) {
	type fields struct {
		Type   string
		Source Source
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "invalid json",
			args: args{
				data: []byte(`{"name":"org.osbuild.foo","options":{"bar":null}`),
			},
			wantErr: true,
		},
		{
			name: "unknown source",
			args: args{
				data: []byte(`{"name":"org.osbuild.foo","options":{"bar":null}}`),
			},
			wantErr: true,
		},
		{
			name: "missing options",
			args: args{
				data: []byte(`{"name":"org.osbuild.curl"}`),
			},
			wantErr: true,
		},
		{
			name: "missing name",
			args: args{
				data: []byte(`{"foo":null,"options":{"bar":null}}`),
			},
			wantErr: true,
		},
		{
			name: "curl-empty",
			fields: fields{
				Type:   "org.osbuild.curl",
				Source: &CurlSource{Items: map[string]CurlSourceItem{}},
			},
			args: args{
				data: []byte(`{"org.osbuild.curl":{"items":{}}}`),
			},
		},
		{
			name: "curl-with-secrets",
			fields: fields{
				Type: "org.osbuild.curl",
				Source: &CurlSource{
					Items: map[string]CurlSourceItem{
						"checksum1": CurlSourceOptions{URL: "url1", Secrets: &URLSecrets{Name: "org.osbuild.rhsm"}},
						"checksum2": CurlSourceOptions{URL: "url2", Secrets: &URLSecrets{Name: "whatever"}},
					}},
			},
			args: args{
				data: []byte(`{"org.osbuild.curl":{"items":{"checksum1":{"url":"url1","secrets":{"name":"org.osbuild.rhsm"}},"checksum2":{"url":"url2","secrets":{"name":"whatever"}}}}}`),
			},
		},
		{
			name: "curl-url-only",
			fields: fields{
				Type: "org.osbuild.curl",
				Source: &CurlSource{
					Items: map[string]CurlSourceItem{
						"checksum1": URL("url1"),
						"checksum2": URL("url2"),
					}},
			},
			args: args{
				data: []byte(`{"org.osbuild.curl":{"items":{"checksum1":"url1","checksum2":"url2"}}}`),
			},
		},
	}
	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources := &Sources{
				tt.fields.Type: tt.fields.Source,
			}
			var gotSources Sources
			if err := gotSources.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("Sources.UnmarshalJSON() error = %v, wantErr %v [idx: %d]", err, tt.wantErr, idx)
			}
			if tt.wantErr {
				return
			}
			gotBytes, err := json.Marshal(sources)
			if err != nil {
				t.Errorf("Could not marshal source: %v [idx: %d]", err, idx)
			}
			if !bytes.Equal(gotBytes, tt.args.data) {
				t.Errorf("Expected '%v', got '%v' [idx: %d]", string(tt.args.data), string(gotBytes), idx)
			}
			if !reflect.DeepEqual(&gotSources, sources) {
				t.Errorf("got '%v', expected '%v' [idx:%d]", &gotSources, sources, idx)
			}
		})
	}
}

func TestGenSourcesTrivial(t *testing.T) {
	sources, err := GenSources(nil, nil, nil, nil)
	assert.NoError(t, err)

	jsonOutput, err := json.MarshalIndent(sources, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, string(jsonOutput), `{}`)
}

func TestGenSourcesContainerStorage(t *testing.T) {
	imageID := "sha256:c2ecf25cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f"
	containers := []container.Spec{
		{
			ImageID:      imageID,
			LocalStorage: true,
		},
	}
	sources, err := GenSources(nil, nil, nil, containers)
	assert.NoError(t, err)

	jsonOutput, err := json.MarshalIndent(sources, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, string(jsonOutput), `{
  "org.osbuild.containers-storage": {
    "items": {
      "sha256:c2ecf25cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f": {}
    }
  }
}`)
}

func TestGenSourcesSkopeo(t *testing.T) {
	imageID := "sha256:c2ecf25cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f"
	digest := "sha256:aabbcc5cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f"
	containers := []container.Spec{
		{
			Source:  "some-source",
			Digest:  digest,
			ImageID: imageID,
		},
	}
	sources, err := GenSources(nil, nil, nil, containers)
	assert.NoError(t, err)

	jsonOutput, err := json.MarshalIndent(sources, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, string(jsonOutput), `{
  "org.osbuild.skopeo": {
    "items": {
      "sha256:c2ecf25cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f": {
        "image": {
          "name": "some-source",
          "digest": "sha256:aabbcc5cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f"
        }
      }
    }
  }
}`)
}

func TestGenSourcesWithSkopeoIndex(t *testing.T) {
	imageID := "sha256:c2ecf25cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f"
	digest := "sha256:aabbcc5cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f"
	listDigest := "sha256:ffeeaabbcc90e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f"
	containers := []container.Spec{
		{
			Source:     "some-source",
			Digest:     digest,
			ListDigest: listDigest,
			ImageID:    imageID,
		},
	}
	sources, err := GenSources(nil, nil, nil, containers)
	assert.NoError(t, err)

	jsonOutput, err := json.MarshalIndent(sources, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, string(jsonOutput), `{
  "org.osbuild.skopeo": {
    "items": {
      "sha256:c2ecf25cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f": {
        "image": {
          "name": "some-source",
          "digest": "sha256:aabbcc5cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f"
        }
      }
    }
  },
  "org.osbuild.skopeo-index": {
    "items": {
      "sha256:ffeeaabbcc90e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f": {
        "image": {
          "name": "some-source"
        }
      }
    }
  }
}`)
}
