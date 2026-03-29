package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStorageSchemas(t *testing.T) {
	content := `[default]
pattern = .*
retentions = 10s:7d,1m:30d,1h:1y

[carbon]
pattern = ^carbon\.
retentions = 1m:90d
`

	tmpfile, err := os.CreateTemp("", "storage-schemas.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	schemas, err := ParseStorageSchemas(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseStorageSchemas() error = %v", err)
	}

	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}

	if schemas[0].Name != "default" {
		t.Errorf("expected name 'default', got %q", schemas[0].Name)
	}
	if schemas[0].Pattern == nil {
		t.Error("expected pattern to be compiled")
	}
	if !schemas[0].Pattern.MatchString("any.metric.name") {
		t.Error("expected pattern to match 'any.metric.name'")
	}

	if schemas[1].Name != "carbon" {
		t.Errorf("expected name 'carbon', got %q", schemas[1].Name)
	}
	if len(schemas[1].Retentions) != 1 {
		t.Errorf("expected 1 retention, got %d", len(schemas[1].Retentions))
	}
}

func TestParseStorageSchemasWithComments(t *testing.T) {
	content := `# This is a comment
[default]
# Another comment
pattern = .*
retentions = 10s:7d
`

	tmpfile, err := os.CreateTemp("", "storage-schemas.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	schemas, err := ParseStorageSchemas(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseStorageSchemas() error = %v", err)
	}

	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
}

func TestCompareSpecsEqual(t *testing.T) {
	tests := []struct {
		a      []ArchiveSpec
		b      []ArchiveSpec
		equals bool
	}{
		{
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 604800}},
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 604800}},
			true,
		},
		{
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 604800}},
			[]ArchiveSpec{{SecondsPerPoint: 10, RetentionSecs: 604800}},
			false,
		},
		{
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 604800}},
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 3600}},
			false,
		},
		{
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 604800}, {SecondsPerPoint: 3600, RetentionSecs: 31536000}},
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 604800}, {SecondsPerPoint: 3600, RetentionSecs: 31536000}},
			true,
		},
		{
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 604800}},
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 604800}, {SecondsPerPoint: 3600, RetentionSecs: 31536000}},
			false,
		},
		{
			[]ArchiveSpec{},
			[]ArchiveSpec{},
			true,
		},
	}

	for i, tt := range tests {
		t.Run(string(rune('0'+i)), func(t *testing.T) {
			got := CompareSpecsEqual(tt.a, tt.b)
			if got != tt.equals {
				t.Errorf("CompareSpecsEqual() = %v, want %v", got, tt.equals)
			}
		})
	}
}

func TestFindWhisperFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "whisper-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "test1.wsp"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test2.wsp"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test3.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "subdir", "nested.wsp"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := FindWhisperFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindWhisperFiles() error = %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 whisper files, got %d", len(files))
	}
}

func TestMetricFromPath(t *testing.T) {
	tests := []struct {
		root string
		full string
		want string
	}{
		{"/var/lib/whisper", "/var/lib/whisper/servers/web01/cpu.wsp", "servers.web01.cpu"},
		{"/var/lib/whisper", "/var/lib/whisper/metric.wsp", "metric"},
		{"/whisper", "/whisper/a.b.c.wsp", "a.b.c"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := MetricFromPath(tt.root, tt.full)
			if got != tt.want {
				t.Errorf("MetricFromPath(%q, %q) = %v, want %v", tt.root, tt.full, got, tt.want)
			}
		})
	}
}

func TestCountDefinitions(t *testing.T) {
	schemas := []Schema{
		{
			Name:       "default",
			PatternRaw: ".*",
			Pattern:    nil,
		},
	}

	files := []string{
		"/whisper/metric1.wsp",
		"/whisper/metric2.wsp",
		"/whisper/metric3.wsp",
	}

	schemas[0].Pattern = nil

	counts, err := CountDefinitions(schemas, "/whisper", files)
	if err != nil {
		t.Fatalf("CountDefinitions() error = %v", err)
	}

	if len(counts) != 1 {
		t.Fatalf("expected 1 count, got %d", len(counts))
	}

	if counts[0].Count != 0 {
		t.Errorf("expected 0 count with nil pattern, got %d", counts[0].Count)
	}
}

func TestParseStorageSchemasWithNilPattern(t *testing.T) {
	content := `[default]
retentions = 10s:7d
`

	tmpfile, err := os.CreateTemp("", "storage-schemas.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	schemas, err := ParseStorageSchemas(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseStorageSchemas() error = %v", err)
	}

	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}

	if schemas[0].Pattern != nil {
		t.Error("expected nil pattern for no pattern in config")
	}
}

func TestParseStorageSchemasInvalidPattern(t *testing.T) {
	content := `[default]
pattern = [invalid
retentions = 10s:7d
`

	tmpfile, err := os.CreateTemp("", "storage-schemas.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = ParseStorageSchemas(tmpfile.Name())
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestParseStorageSchemasEmptySection(t *testing.T) {
	content := `[default]

[carbon]
pattern = ^carbon\.
retentions = 1m:90d
`

	tmpfile, err := os.CreateTemp("", "storage-schemas.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	schemas, err := ParseStorageSchemas(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseStorageSchemas() error = %v", err)
	}

	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema (empty one should be skipped), got %d", len(schemas))
	}

	if schemas[0].Name != "carbon" {
		t.Errorf("expected 'carbon', got %q", schemas[0].Name)
	}
}
