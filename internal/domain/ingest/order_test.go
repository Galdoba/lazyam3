package ingest

import (
	"reflect"
	"testing"
)

func TestSortUnitFiles(t *testing.T) {
	v := Stream{Kind: StreamKindVideo}
	a := Stream{Kind: StreamKindAudio}
	s := Stream{Kind: StreamKindSubtitle}
	d := Stream{Kind: StreamKindData}

	tests := []struct {
		name  string
		files map[string][]Stream
		want  []string
	}{
		{
			name: "kind groups override lexicographic order",
			files: map[string][]Stream{
				"63603-rus.srt": {s},
				"63603.dat":     {d},
				"63603.mp4":     {v, a},
				"63603_add.aac": {a},
			},
			want: []string{"63603.mp4", "63603_add.aac", "63603-rus.srt", "63603.dat"},
		},
		{
			name: "video+audio file ranks as video",
			files: map[string][]Stream{
				"b.mkv": {v, a},
				"a.srt": {s},
			},
			want: []string{"b.mkv", "a.srt"},
		},
		{
			name: "lexicographic within group",
			files: map[string][]Stream{
				"63603_b.mp4": {v},
				"63603_a.mkv": {v},
			},
			want: []string{"63603_a.mkv", "63603_b.mp4"},
		},
		{
			name: "audio-only unit",
			files: map[string][]Stream{
				"63603.srt": {s},
				"63603.aac": {a},
			},
			want: []string{"63603.aac", "63603.srt"},
		},
		{
			name: "subtitle+audio file ranks as audio",
			files: map[string][]Stream{
				"63603.mp4": {v},
				"63603.mkv": {a, s},
			},
			want: []string{"63603.mp4", "63603.mkv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortUnitFiles(tt.files)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortUnitFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileRank(t *testing.T) {
	tests := []struct {
		name    string
		streams []Stream
		want    int
	}{
		{"video", []Stream{{Kind: StreamKindVideo}}, 0},
		{"video+audio", []Stream{{Kind: StreamKindAudio}, {Kind: StreamKindVideo}}, 0},
		{"audio-only", []Stream{{Kind: StreamKindAudio}}, 1},
		{"audio+subtitle", []Stream{{Kind: StreamKindSubtitle}, {Kind: StreamKindAudio}}, 1},
		{"subtitle-only", []Stream{{Kind: StreamKindSubtitle}}, 2},
		{"data-only", []Stream{{Kind: StreamKindData}}, 3},
		{"empty", nil, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileRank(tt.streams); got != tt.want {
				t.Errorf("fileRank() = %d, want %d", got, tt.want)
			}
		})
	}
}
