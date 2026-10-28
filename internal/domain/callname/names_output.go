package callname

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// PredictedOutput describes a single file that will be produced by the
// rendering pipeline.
type PredictedOutput struct {
	// Path is the full absolute path (e.g. "/mnt/ROOT/EDIT/Show_s01e06/Show_s01e06_HD.mp4").
	Path string
	// MediaType is "video", "audio", or "subtitle".
	MediaType string
	// Language is the IANA language tag (e.g. "rus", "jpn", "und").
	Language string
	// Layout is the channel-layout suffix ("20", "51", "70", "10", "XX").
	Layout string
	// AudioIndex is the zero-based index among audio outputs.
	AudioIndex int
	// HsubFile is the hardsub SRT filename to burn into the video, or "" if none.
	// Only meaningful on video predictions.
	HsubFile string
}

// PredictOutputs returns the set of files the rendering pipeline will
// produce for the given media unit.  It derives the same naming logic
// as the script domain (video, audio, SRT) without depending on it.
// hardsubTag is a filename substring that marks sidecar subtitles as hardsub.
// prodID is the product ID, used in output filenames as _idN.
func PredictOutputs(product metadata.Product, unit ingest.MediaUnit, targetRoot, hardsubTag string, prodID int64) []PredictedOutput {
	streams := make([]ingest.Stream, 0, len(unit.Streams))
	for _, s := range unit.Streams {
		streams = append(streams, s)
	}
	// Sort streams by file index then by stream index for deterministic output order.
	slices.SortFunc(streams, func(a, b ingest.Stream) int {
		if a.FileIndex != b.FileIndex {
			return a.FileIndex - b.FileIndex
		}
		return a.Index - b.Index
	})

	baseName := deriveOutBase(product)
	seasonTag := deriveSeasonTag(product)
	nameTag := deriveOutputNameTag(seasonTag, deriveEpisodeTag(product))
	seasonDir := ""
	if seasonTag != "" {
		seasonDir = "_s" + seasonTag
	}

	// TARGET_DIR = targetRoot / (baseName + seasonDir)
	// e.g. /mnt/ROOT/EDIT/Belaya_koroleva_s01
	var targetDir string
	if seasonDir != "" {
		targetDir = filepath.Join(targetRoot, baseName+seasonDir)
	} else {
		targetDir = filepath.Join(targetRoot, baseName)
	}
	// Normalize to forward slashes for cross-platform compatibility
	targetDir = filepath.ToSlash(targetDir)

	// OUTBASE = baseName + nameTag + "_PRT" + timestamp + "_idN"
	// (e.g. Belaya_koroleva_s01_01_PRT261001120000_id12345)
	outBase := baseName + nameTag
	if unit.PRTTimestamp != "" {
		outBase = outBase + "_PRT" + unit.PRTTimestamp
	}
	if prodID > 0 {
		outBase += fmt.Sprintf("_id%d", prodID)
	}

	// Detect hardsub: a subtitle file whose name contains the hardsub tag.
	hsubFile := detectHSUB(unit.Filenames, hardsubTag)

	var outputs []PredictedOutput

	// Video: always one output.
	video := PredictedOutput{
		Path:      filepath.ToSlash(filepath.Join(targetDir, outBase+"_HD.mp4")),
		MediaType: "video",
		HsubFile:  hsubFile,
	}
	outputs = append(outputs, video)

	// Audio: one per audio stream.
	audioIdx := 0
	for _, s := range streams {
		if s.Kind != ingest.StreamKindAudio {
			continue
		}
		lang := string(s.Language)
		if lang == "" {
			lang = "und"
		}
		layout := layoutTag(s.ChannelLayout)
		outputs = append(outputs, PredictedOutput{
			Path:       filepath.ToSlash(filepath.Join(targetDir, fmt.Sprintf("%s_HD_AUDIO%02d_%s%s.m4a", outBase, audioIdx, strings.ToUpper(lang), layout))),
			MediaType:  "audio",
			Language:   lang,
			Layout:     layout,
			AudioIndex: audioIdx,
		})
		audioIdx++
	}

	// Subtitle: one SRT if any subtitle stream exists.
	for _, s := range streams {
		if s.Kind == ingest.StreamKindSubtitle {
			outputs = append(outputs, PredictedOutput{
				Path:      filepath.ToSlash(filepath.Join(targetDir, outBase+"_HD.srt")),
				MediaType: "subtitle",
			})
			break
		}
	}

	return outputs
}

// detectHSUB returns the first filename that looks like a hardsub file
// (a subtitle file containing hardsubTag in its name), or "" if none.
func detectHSUB(filenames []string, hardsubTag string) string {
	if hardsubTag == "" {
		// Legacy: fall back to "hardsub" substring for backwards compat.
		for _, f := range filenames {
			if strings.Contains(strings.ToLower(f), "hardsub") {
				return f
			}
		}
		return ""
	}
	lowerTag := strings.ToLower(hardsubTag)
	for _, f := range filenames {
		if strings.Contains(strings.ToLower(f), lowerTag) {
			return f
		}
	}
	return ""
}

// layoutTag maps an ingest.Layout to the suffix used in output filenames.
func layoutTag(layout ingest.Layout) string {
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

// deriveSeasonTag returns the season number as a zero-padded string.
func deriveSeasonTag(p metadata.Product) string {
	if p.SeasonNumber != nil {
		return fmt.Sprintf("%02d", *p.SeasonNumber)
	}
	return ""
}

// deriveEpisodeTag returns the episode number as a zero-padded string.
func deriveEpisodeTag(p metadata.Product) string {
	if p.EpisodeNumber != nil {
		return fmt.Sprintf("%02d", *p.EpisodeNumber)
	}
	return ""
}

// deriveOutputNameTag returns the season/episode suffix for output filenames:
// "_s01_06", "_s01", "_06", or "" (movie). The episode tag has no "e" prefix.
func deriveOutputNameTag(seasonTag, episodeTag string) string {
	switch {
	case seasonTag != "" && episodeTag != "":
		return "_s" + seasonTag + "_" + episodeTag
	case seasonTag != "":
		return "_s" + seasonTag
	case episodeTag != "":
		return "_" + episodeTag
	}
	return ""
}

// deriveNameTagCompact returns the season/episode suffix without separators:
// "s01e06", "s01", "e06", or "" (movie). Used for script filenames.
func deriveNameTagCompact(seasonTag, episodeTag string) string {
	switch {
	case seasonTag != "" && episodeTag != "":
		return "s" + seasonTag + "e" + episodeTag
	case seasonTag != "":
		return "s" + seasonTag
	case episodeTag != "":
		return "e" + episodeTag
	}
	return ""
}

// PredictScriptName generates the script filename for the given product and unit.
// It uses the compact naming format (e.g. "Show_s01e06_PRT261023120000.sh").
func PredictScriptName(product metadata.Product, unit ingest.MediaUnit, targetRoot, hardsubTag, timestamp string) string {
	baseName := deriveOutBase(product)
	seasonTag := deriveSeasonTag(product)
	episodeTag := deriveEpisodeTag(product)
	nameTag := deriveNameTagCompact(seasonTag, episodeTag)

	scriptName := baseName + "_" + nameTag
	if unit.PRTTimestamp != "" {
		scriptName = baseName + "_" + nameTag + "_PRT" + unit.PRTTimestamp
	} else if timestamp != "" {
		scriptName = baseName + "_" + nameTag + "_PRT" + timestamp
	}

	return filepath.ToSlash(filepath.Join(targetRoot, scriptName+".sh"))
}
