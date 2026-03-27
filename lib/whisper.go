package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	whisper "github.com/go-graphite/go-whisper"
)

func WhisperRetentionsToSpecs(retentions []whisper.Retention) []ArchiveSpec {
	out := make([]ArchiveSpec, 0, len(retentions))
	for _, r := range retentions {
		sp := r.SecondsPerPoint()
		points := r.NumberOfPoints()
		total := sp * points
		out = append(out, ArchiveSpec{
			SecondsPerPoint: sp,
			RetentionSecs:   total,
		})
	}
	return out
}

// findWhisperFiles walks root and returns all files ending with .wsp
func FindWhisperFiles(root string) ([]string, error) {
	out := []string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip unreadable files/directories
			fmt.Fprintf(os.Stderr, "Skipping %s: %v\n", path, err)
			return nil // <- IMPORTANT: continue walking
			// don't stop on single file errors; but return error if stat fails
			// return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".wsp") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

// metricFromPath converts a filesystem path to Graphite metric name relative to root.
// e.g. /var/lib/graphite/whisper/servers/web01/cpu.wsp -> servers.web01.cpu
func MetricFromPath(root, full string) string {
	rel, err := filepath.Rel(root, full)
	if err != nil {
		// fallback to full path turned into dots (not ideal)
		rel = full
	}
	rel = strings.TrimSuffix(rel, ".wsp")
	// on Windows or other OSes, ensure separators are normalized
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	return strings.ReplaceAll(rel, string(filepath.Separator), ".")
}

// ResizeOptions contains options for resizing a whisper file
type ResizeOptions struct {
	XFF         float32
	Compressed  bool
	Aggregation string
}

// ResizeWhisperFile resizes a whisper file to match the given retentions
// Returns old and new retention strings, and an error if something went wrong
func ResizeWhisperFile(path string, retentions []ArchiveSpec, opts ResizeOptions) (oldRetentions, newRetentions string, resized bool, err error) {
	wsp, err := whisper.Open(path)
	if err != nil {
		return "", "", false, fmt.Errorf("failed to open: %w", err)
	}

	oldRetentions = FormatRetentionList(WhisperRetentionsToSpecs(wsp.Retentions()))

	if CompareSpecsEqual(WhisperRetentionsToSpecs(wsp.Retentions()), retentions) {
		_ = wsp.Close()
		return oldRetentions, oldRetentions, false, nil
	}

	newRetentions = FormatRetentionList(retentions)

	whisperRetentions := make(whisper.Retentions, len(retentions))
	for i, r := range retentions {
		points := r.RetentionSecs / r.SecondsPerPoint
		ret := whisper.NewRetention(r.SecondsPerPoint, points)
		whisperRetentions[i] = &ret
	}

	var aggrMethod whisper.AggregationMethod
	switch strings.ToLower(opts.Aggregation) {
	case "sum":
		aggrMethod = whisper.Sum
	case "last":
		aggrMethod = whisper.Last
	case "max":
		aggrMethod = whisper.Max
	case "min":
		aggrMethod = whisper.Min
	case "first":
		aggrMethod = whisper.First
	default:
		aggrMethod = whisper.Average
	}

	xff := opts.XFF
	if xff == 0 {
		xff = 0.5
	}

	options := &whisper.Options{
		Compressed: opts.Compressed,
		FLock:      true,
	}

	err = wsp.UpdateConfig(whisperRetentions, aggrMethod, xff, options)
	if err != nil {
		return oldRetentions, newRetentions, false, fmt.Errorf("failed to resize: %w", err)
	}

	return oldRetentions, newRetentions, true, nil
}
