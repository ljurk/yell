package lib

import (
	"testing"
)

func TestFromHuman(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"10s", 10, false},
		{"1m", 60, false},
		{"1h", 3600, false},
		{"1d", 86400, false},
		{"1y", 31536000, false},
		{"5m", 300, false},
		{"2h", 7200, false},
		{"7d", 604800, false},
		{"", -1, true},
		{"invalid", -1, true},
		{"10x", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := fromHuman(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("fromHuman(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("fromHuman(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestToHuman(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{10, "10s"},
		{60, "1m"},
		{300, "5m"},
		{3600, "1h"},
		{7200, "2h"},
		{86400, "1d"},
		{604800, "7d"},
		{31536000, "1y"},
		{0, "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := ToHuman(tt.input)
			if got != tt.expected {
				t.Errorf("ToHuman(%d) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatRetentionList(t *testing.T) {
	tests := []struct {
		specs    []ArchiveSpec
		expected string
	}{
		{
			[]ArchiveSpec{{SecondsPerPoint: 60, RetentionSecs: 604800}},
			"1m:7d",
		},
		{
			[]ArchiveSpec{{SecondsPerPoint: 10, RetentionSecs: 604800}, {SecondsPerPoint: 60, RetentionSecs: 2592000}},
			"10s:7d,1m:30d",
		},
		{
			[]ArchiveSpec{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := FormatRetentionList(tt.specs)
			if got != tt.expected {
				t.Errorf("FormatRetentionList() = %v, want %v", got, tt.expected)
			}
		})
	}
}
