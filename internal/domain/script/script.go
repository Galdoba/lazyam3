package script

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
)

// StreamSelection represents a stream selection rule for the rendering script.
type StreamSelection struct {
	InputIndex  int
	StreamIndex int
	Kind        string // "v", "a", "s"
}

// Script is a rendered bash script ready to be written to disk.
type Script struct {
	Destination string
	Content     string
}

// BuildScript creates a rendering script from the given config.
// It is a pure function: no IO, no side effects, fully deterministic.
func BuildScript(cfg ScriptConfig) (Script, error) {
	// Validate inputs
	if err := validateConfig(cfg); err != nil {
		return Script{}, err
	}

	// Derive naming from product
	cfg.BaseName = deriveBaseName(cfg.Product)
	cfg.SeasonTag = deriveSeasonTag(cfg.Product)
	cfg.EpisodeTag = deriveEpisodeTag(cfg.Product)
	cfg.NameTag = deriveNameTag(cfg.SeasonTag, cfg.EpisodeTag)
	cfg.SeasonDir = deriveSeasonDir(cfg.SeasonTag)

	// Build video section
	interlaced := getInterlaced(cfg.Streams)
	yadif := buildVideoFilter(interlaced)
	hsub := buildHSUB(cfg.HSUBFile, cfg.Env.InProgress)

	// Build audio section
	audioTracks := selectAudioTracks(cfg)
	audioFilter := buildAudioFilter(audioTracks)

	// Build SRT sections
	srtMove := buildSRTMove(cfg.SRT)
	srtSection := buildSRTSection(cfg.SRT)
	srtCleanup := buildSRTCleanup(cfg.SRT)

	// Build HSUB sections
	hsubMove := buildHSUBMove(cfg.HSUBFile)
	hsubCleanup := buildHSUBCleanup(cfg.HSUBFile)

	// Fill template placeholders
	content := AmediaGeneric
	content = replaceAll(content, "source", cfg.Filenames[0])
	content = replaceAll(content, "srt", cfg.SRT)
	content = replaceAll(content, "hsubfile", cfg.HSUBFile)
	content = replaceAll(content, "basename", cfg.BaseName)
	content = replaceAll(content, "name_tag", cfg.NameTag)
	content = replaceAll(content, "season_dir", cfg.SeasonDir)
	content = replaceAll(content, "yadif", yadif)
	content = replaceAll(content, "hsub", hsub)
	content = replaceAll(content, "audio_filter", audioFilter)
	content = replaceAll(content, "output_files", buildOutputFiles(audioTracks))
	content = replaceAll(content, "srt_move", srtMove)
	content = replaceAll(content, "srt_section", srtSection)
	content = replaceAll(content, "srt_cleanup", srtCleanup)
	content = replaceAll(content, "hsub_move", hsubMove)
	content = replaceAll(content, "hsub_cleanup", hsubCleanup)
	content = replaceAll(content, "priority", fmt.Sprintf("%d", cfg.Env.Priority))
	content = replaceAll(content, "buffer", cfg.Env.Buffer)
	content = replaceAll(content, "in_progress", cfg.Env.InProgress)
	content = replaceAll(content, "done", cfg.Env.Done)
	content = replaceAll(content, "notifications", cfg.Env.Notifications)
	content = replaceAll(content, "target_root", cfg.Env.TargetRoot)

	// Validate no unresolved placeholders remain
	if unresolved := findUnresolved(content); len(unresolved) > 0 {
		return Script{}, fmt.Errorf("unresolved placeholders: %s", strings.Join(unresolved, ", "))
	}

	return Script{
		Destination: cfg.Destination,
		Content:     content,
	}, nil
}

// validateConfig checks that the config has all required data.
func validateConfig(cfg ScriptConfig) error {
	if cfg.Destination == "" {
		return fmt.Errorf("script: destination must not be empty")
	}

	if len(cfg.Streams) == 0 {
		return fmt.Errorf("script: at least one stream is required")
	}

	// Check for at least one video stream
	hasVideo := false
	for _, s := range cfg.Streams {
		if s.Kind == ingest.StreamKindVideo {
			hasVideo = true
			break
		}
	}
	if !hasVideo {
		return fmt.Errorf("script: at least one video stream is required")
	}

	// Validate selections reference existing streams
	for _, sel := range cfg.StreamSelections {
		found := false
		for _, s := range cfg.Streams {
			if s.FileIndex == sel.InputIndex && s.Index == sel.StreamIndex {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("script: selection [%d:%s:%d] references non-existent stream", sel.InputIndex, sel.Kind, sel.StreamIndex)
		}
	}

	return nil
}

// findUnresolved returns a list of unresolved placeholders in the content.
func findUnresolved(content string) []string {
	re := regexp.MustCompile(`\|=(\w+)=\|`)
	matches := re.FindAllStringSubmatch(content, -1)
	var unresolved []string
	for _, m := range matches {
		unresolved = append(unresolved, m[1])
	}
	return unresolved
}

// replaceAll replaces a placeholder |=key=| with value in the template.
func replaceAll(template, key, value string) string {
	placeholder := fmt.Sprintf("|=%s=|", key)
	return strings.ReplaceAll(template, placeholder, value)
}

// buildSRTMove generates the shell line to move the SRT file from buffer to in_progress.
func buildSRTMove(srt string) string {
	if srt == "" {
		return ""
	}
	return `mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"`
}

// buildSRTSection generates the shell line to copy the SRT file to the target directory.
func buildSRTSection(srt string) string {
	if srt == "" {
		return ""
	}
	return `cp "${IN_PROGRESS}/${SRT}" "${TARGET_DIR}/${OUTBASE}.srt"`
}

// buildSRTCleanup generates the shell line to move the SRT file to done.
func buildSRTCleanup(srt string) string {
	if srt == "" {
		return ""
	}
	return `mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"`
}

// buildHSUB generates the hardsub filter string if HSUBFile is set.
// The path is resolved against inProgress so the filter references the
// absolute location inside IN_PROGRESS where the script moves the file.
func buildHSUB(hsubFile, inProgress string) string {
	if hsubFile == "" {
		return ""
	}
	path := hsubFile
	if inProgress != "" {
		path = filepath.Join(inProgress, hsubFile)
	}
	return fmt.Sprintf(`subtitles=filename=%s:original_size=1920x1080:force_style='FontName=Segoe UI,Fontsize=19.4,PrimaryColour='&H00EBEBEB',SecondaryColour='&H000000FF',OutlineColour='&H00000000',BackColour='&H80000000',Underline=0,StrikeOut=0,ScaleX=1,ScaleY=1,Spacing=0,Angle=0,BorderStyle=1,Outline=0.5,Shadow=0,Alignment=2,MarginL=5,MarginR=5,MarginV=20,Encoding=1',`, path)
}

// buildHSUBMove generates the shell line to move the hardsub file from
// buffer to in_progress alongside the media.
func buildHSUBMove(hsubFile string) string {
	if hsubFile == "" {
		return ""
	}
	return `mv "${BUFFER}/${HSUBFILE}" "${IN_PROGRESS}/${HSUBFILE}"`
}

// buildHSUBCleanup generates the shell line to move the hardsub file to
// done after rendering is complete.
func buildHSUBCleanup(hsubFile string) string {
	if hsubFile == "" {
		return ""
	}
	return `mv "${IN_PROGRESS}/${HSUBFILE}" "${DONE}/${HSUBFILE}"`
}
