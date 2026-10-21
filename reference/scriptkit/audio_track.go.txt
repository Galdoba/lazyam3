package scriptkit

import (
	"fmt"
	"strings"
)

// AudioTrack represents a single audio track for script generation.
type AudioTrack struct {
	// StreamIndex is the 0-based stream index in the file (0:a:0, 0:a:1, …)
	StreamIndex int
	// Language is the audio language tag (e.g. "rus", "eng")
	Language string
	// Layout is the channel layout code (e.g. "20" for stereo, "51" for 5.1)
	Layout string
	// Bitrate is the audio bitrate string from ffprobe (e.g. "317")
	Bitrate string
	// Suffix is the display label used in output filenames, e.g. "AUDIORUS20"
	Suffix string
}

// NewAudioTrack creates an AudioTrack with an auto-generated Suffix.
// The suffix format is "AUDIO_{index}_{LANG}{LAYOUT}" where index is the
// 0-based audio stream position in the file.
func NewAudioTrack(lang, layout string, index int) AudioTrack {
	return AudioTrack{
		StreamIndex: index,
		Language:    lang,
		Layout:      layout,
		Suffix:      fmt.Sprintf("AUDIO_%02d_%s%s", index, strings.ToUpper(lang), layout),
	}
}

// AudioScriptSection builds the audio portion of the fflite command.
// Returns the filter_complex segment (semicolon-separated audio filter chains)
// and the output file mapping lines (one per audio track).
//
// Example for 2 tracks (rus/stereo at stream 0, eng/stereo at stream 1):
//
//	filter: "[0:a:0]aresample=48000,atempo=25/(25/1)[aud1];[0:a:1]aresample=48000,atempo=25/(25/1)[aud2]"
//	outputs: "  -map \"[aud1]\" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 \"${TARGET_DIR}/${OUTBASE}_HD_AUDIORUS20.m4a\" \\\n  -map \"[aud2]\" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 \"${TARGET_DIR}/${OUTBASE}_HD_AUDIOENG20.m4a\""
func AudioScriptSection(tracks []AudioTrack) (filterPart, outputLines string) {
	if len(tracks) == 0 {
		return "", ""
	}

	var filterParts []string
	var outputLinesList []string

	for i, t := range tracks {
		label := fmt.Sprintf("aud%d", i+1)
		filterParts = append(filterParts,
			fmt.Sprintf("[0:a:%d]aresample=48000,atempo=25/(25/1)[%s]", t.StreamIndex, label))
		outputLinesList = append(outputLinesList,
			fmt.Sprintf(`  -map "[%s]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD_%s.m4a"`,
				label, t.Suffix))
	}

	return strings.Join(filterParts, ";"), strings.Join(outputLinesList, " \\\n")
}
