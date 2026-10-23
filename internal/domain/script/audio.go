package script

import (
	"fmt"
	"strings"

	"github.com/Galdoba/lazyam3/internal/domain/callname"
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
)

// audioTrack represents an audio stream selected for output.
type audioTrack struct {
	StreamIndex int
	Language    string
	Layout      string
	Suffix      string // e.g. "AUDIO_00_RUS20"
}

// layoutSuffix maps ingest.Layout to the ffmpeg-style layout suffix used in output naming.
func layoutSuffix(layout ingest.Layout) string {
	switch layout {
	case ingest.LayoutStereo:
		return "20"
	case ingest.LayoutSurround51:
		return "51"
	case ingest.LayoutSurround71:
		return "70"
	case ingest.LayoutMono:
		return "10"
	default:
		return "20"
	}
}

// selectAudioTracks returns audio tracks from the config's streams and selections.
// If selections are provided, only selected audio streams are included.
// If no selections are provided, all audio streams are included.
// Deprecated: replaced by callname package.
// func selectAudioTracks(cfg ScriptConfig) []audioTrack {
// 	var tracks []audioTrack
//
// 	if len(cfg.StreamSelections) > 0 {
// 		for _, sel := range cfg.StreamSelections {
// 			if sel.Kind != "a" {
// 				continue
// 			}
// 			for _, stream := range cfg.Streams {
// 				if stream.Kind != ingest.StreamKindAudio {
// 					continue
// 				}
// 				if stream.FileIndex != sel.InputIndex || stream.Index != sel.StreamIndex {
// 					continue
// 				}
// 				tracks = append(tracks, audioTrack{
// 					StreamIndex: stream.Index,
// 					Language:    string(stream.Language),
// 					Layout:      string(stream.ChannelLayout),
// 				})
// 			}
// 		}
// 	} else {
// 		for _, stream := range cfg.Streams {
// 			if stream.Kind != ingest.StreamKindAudio {
// 				continue
// 			}
// 			tracks = append(tracks, audioTrack{
// 				StreamIndex: stream.Index,
// 				Language:    string(stream.Language),
// 				Layout:      string(stream.ChannelLayout),
// 			})
// 		}
// 	}
//
// 	// Assign suffixes and normalize
// 	for i, t := range tracks {
// 		if t.Language == "" {
// 			t.Language = "und"
// 		}
// 		if t.Layout == "" {
// 			t.Layout = string(ingest.LayoutStereo)
// 		}
// 		t.Layout = layoutSuffix(ingest.Layout(t.Layout))
// 		t.Suffix = fmt.Sprintf("AUDIO_%02d_%s%s", i, strings.ToUpper(t.Language), t.Layout)
// 		tracks[i] = t
// 	}
//
// 	return tracks
// }

// buildAudioFilter creates the filter_complex segment for audio tracks.
// Each track gets: [0:N]aresample=48000,atempo=25/(25/1)[audK], where N is the
// absolute stream index within the source file (same numbering as ffprobe
// streams[].index / ffmpeg -map 0:N). Type-local a:N is NOT used: it would
// miscount whenever non-audio streams precede the audio ones.
// Segments are joined with ";" and the whole block is prefixed with ";" to
// separate it from the video chain — no trailing separator: ffmpeg rejects
// graphs with empty chains ("No such filter: ”").
func buildAudioFilter(tracks []audioTrack) string {
	if len(tracks) == 0 {
		return ""
	}

	parts := make([]string, 0, len(tracks))
	for i, t := range tracks {
		chunk := fmt.Sprintf("[0:%d]aresample=48000,atempo=25/(25/1)[aud%d]", t.StreamIndex, i)
		parts = append(parts, chunk)
	}
	return ";" + strings.Join(parts, ";")
}

// Deprecated: replaced by callname package.
// func buildOutputFiles(tracks []audioTrack) string {
// 	lines := []string{
// 		`  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD.mp4"`,
// 	}
// 	for i, t := range tracks {
// 		lines = append(lines, fmt.Sprintf(`  -map "[aud%d]" -c:a alac -compression_level 0 "${TARGET_DIR}/${OUTBASE}_HD_%s.m4a"`, i, t.Suffix))
// 	}
// 	for i := 0; i < len(lines)-1; i++ {
// 		lines[i] += ` \`
// 	}
// 	return strings.Join(lines, "\n")
// }

// buildOutputFilesPredictions renders the -map output block using predicted
// paths. The first prediction is always video; subsequent audio predictions
// get their full paths directly. Uses AudioIndex for the [audN] map label.
func buildOutputFilesPredictions(predictions []callname.PredictedOutput) string {
	lines := make([]string, 0, len(predictions))
	for _, p := range predictions {
		switch p.MediaType {
		case "video":
			line := fmt.Sprintf(`  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0 -map_metadata -1 -map_chapters -1 "%s"`, p.Path)
			lines = append(lines, line)
		case "audio":
			line := fmt.Sprintf(`  -map "[aud%d]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "%s"`, p.AudioIndex, p.Path)
			lines = append(lines, line)
		}
	}
	for i := 0; i < len(lines); i++ {
		lines[i] += ` \`
	}
	return strings.Join(lines, "\n")
}

// buildVideoFilter returns the yadif filter string if the video is interlaced, empty otherwise.
func buildVideoFilter(interlaced bool) string {
	if interlaced {
		return "yadif,"
	}
	return ""
}

// getInterlaced returns true if any video stream is interlaced.
func getInterlaced(streams []ingest.Stream) bool {
	for _, s := range streams {
		if s.Kind == ingest.StreamKindVideo && s.Interlaced {
			return true
		}
	}
	return false
}
