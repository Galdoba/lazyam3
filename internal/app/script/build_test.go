package script

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	appingest "github.com/Galdoba/lazyam3/internal/app/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/callname"
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	dscript "github.com/Galdoba/lazyam3/internal/domain/script"
	mdinfra "github.com/Galdoba/lazyam3/internal/infra/metadata"
	scriptinfra "github.com/Galdoba/lazyam3/internal/infra/script"
)

// realConfig runs the real ingest + metadata chain on the synthetic fixture
// and assembles a ScriptConfig rooted at a fresh temp dir. Skips the test when
// ffprobe, testdata, or metadata are unavailable.
func realConfig(t *testing.T) (dscript.ScriptConfig, string, string) {
	t.Helper()
	const ffprobe = "/usr/bin/ffprobe"
	if _, err := exec.LookPath(ffprobe); err != nil {
		t.Skip("ffprobe not available")
	}
	srcDir := filepath.Join("..", "..", "..", "testdata", "synthetic", "hardsub-with-subtitle")
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		t.Skip("synthetic testdata not available")
	}
	metaPath := filepath.Join("..", "..", "..", "testdata", "metadata.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Skip("metadata testdata not available")
	}

	// Real ingest (ffprobe on the fixture directory).
	unit, err := appingest.New(ffprobe).Ingest(srcDir)
	if err != nil {
		t.Fatalf("Ingest() error: %v", err)
	}
	if unit.ID != 12254 {
		t.Fatalf("unit.ID = %d, want 12254", unit.ID)
	}
	// Ingest orders video-containing files first — the unit guarantees its
	// primary media file at index 0 (the .srt sorts earlier lexicographically).
	if len(unit.Filenames) == 0 {
		t.Fatal("unit.Filenames is empty")
	}
	if unit.Filenames[0] != "12254.mp4" {
		t.Fatalf("unit.Filenames[0] = %q, want 12254.mp4 (unit.Filenames=%v)", unit.Filenames[0], unit.Filenames)
	}

	// Real metadata lookup.
	repo, err := mdinfra.New(metaPath)
	if err != nil {
		t.Fatalf("metadata.New() error: %v", err)
	}
	prod, err := repo.FindByID(unit.ID)
	if err != nil {
		t.Fatalf("FindByID(%d) error: %v", unit.ID, err)
	}
	if prod.SeriesTitle != "Арлисс" {
		t.Errorf("SeriesTitle = %q, want %q", prod.SeriesTitle, "Арлисс")
	}

	streams := make([]ingest.Stream, 0, len(unit.Streams))
	for _, s := range unit.Streams {
		streams = append(streams, s)
	}
	sort.Slice(streams, func(i, j int) bool {
		if streams[i].FileIndex != streams[j].FileIndex {
			return streams[i].FileIndex < streams[j].FileIndex
		}
		return streams[i].Index < streams[j].Index
	})

	envRoot := t.TempDir()

	// Build call names for all unit files.
	timestamp := time.Now().Format("060102150405")
	callNames, err := callname.BuildInputNames(*prod, unit.Filenames, timestamp)
	if err != nil {
		t.Fatalf("BuildInputNames error: %v", err)
	}

	// Predict outputs (drives naming).
	unit.PRTTimestamp = timestamp
	predictions := callname.PredictOutputs(*prod, unit, filepath.Join(envRoot, "target"), "", prod.ID)

	// Resolve SRT/HSUB to call names.
	srtOrig := "12254-rus_Russkii.srt"
	hsubOrig := "12254-rus_Russkii_Forsirovannye.srt"
	srtCall := callNames[srtOrig]
	hsubCall := callNames[hsubOrig]

	cfg := dscript.ScriptConfig{
		Filenames:   unit.Filenames,
		SRT:         srtCall,
		HSUBFile:    hsubCall,
		Streams:     streams,
		Product:     *prod,
		Predictions: predictions,
		CallNames:   callNames,
		Env: dscript.Environment{
			Buffer:        filepath.Join(envRoot, "buffer"),
			InProgress:    filepath.Join(envRoot, "in_progress"),
			Done:          filepath.Join(envRoot, "done"),
			Notifications: filepath.Join(envRoot, "notifications"),
			Archive:       filepath.Join(envRoot, "done", "bash"),
			TargetRoot:    filepath.Join(envRoot, "target"),
			Priority:      6,
		},
		Destination: callname.PredictScriptName(*prod, unit, filepath.Join(envRoot, "target"), "", timestamp),
	}
	return cfg, srcDir, envRoot
}

func TestBuild_RealInputs(t *testing.T) {
	cfg, srcDir, _ := realConfig(t)

	// Build via the real renderer (use case under test).
	if err := Build(context.Background(), cfg, scriptinfra.New()); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Usable-artifact assertions.
	fi, err := os.Stat(cfg.Destination)
	if err != nil {
		t.Fatalf("stat rendered script: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o755 {
		t.Errorf("mode = %04o, want 0755", got)
	}

	content, err := os.ReadFile(cfg.Destination)
	if err != nil {
		t.Fatalf("read rendered script: %v", err)
	}
	body := string(content)

	for _, want := range []string{
		"#!/bin/bash",
		"ffmpeg -n -r 25 -i",
		"-filter_complex",
		"-map \"[aud0]\"",
		"[0:1]aresample=48000",
		"[0:2]aresample=48000",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered script does not contain %q", want)
		}
	}
	// SOURCE and SRT must be call names, not original filenames.
	if strings.Contains(body, `SOURCE="12254.mp4"`) {
		t.Error("SOURCE should be a call name, not the original filename")
	}
	if strings.Contains(body, `SRT="12254-rus_Russkii.srt"`) {
		t.Error("SRT should be a call name, not the original filename")
	}

	if unresolved := regexp.MustCompile(`\|=\w+=\|`).FindAllString(body, -1); len(unresolved) > 0 {
		t.Errorf("unresolved placeholders in rendered script: %v", unresolved)
	}

	// Validity: the artifact must be a syntactically valid bash script.
	if bash, err := exec.LookPath("bash"); err == nil {
		if out, err := exec.Command(bash, "-n", cfg.Destination).CombinedOutput(); err != nil {
			t.Errorf("bash -n failed: %v\n%s", err, out)
		}
	}

	// Semantics: execute the generated filtergraph — bash -n cannot see
	// stream-specifier errors like "matches no streams". The filter line uses
	// shell variables (HSUB/YADIF) that bash expands at runtime; expand them
	// here the same way before handing the graph to ffmpeg.
	// Note: HSUB references a file inside IN_PROGRESS which the script
	// moves at runtime; for the filtergraph test we strip the HSUB filter
	// so ffmpeg only validates the video/audio chain.
	fc := regexp.MustCompile(`-filter_complex "([^"]+)"`).FindStringSubmatch(body)
	if fc == nil {
		t.Fatal("rendered script has no -filter_complex")
	}
	graph := fc[1]
	// Strip HSUB from the graph for the direct-ffmpeg test (the HSUB file
	// lives in IN_PROGRESS which the script populates at runtime, not here).
	graph = regexp.MustCompile(`\$\{HSUB\}`).ReplaceAllString(graph, "")
	for _, name := range []string{"YADIF"} {
		assign := regexp.MustCompile(`(?m)^` + name + `="(.*)"$`).FindStringSubmatch(body)
		if assign == nil {
			t.Fatalf("rendered script has no %s assignment", name)
		}
		graph = strings.ReplaceAll(graph, "${"+name+"}", assign[1])
	}
	if ffmpeg, err := exec.LookPath("ffmpeg"); err == nil {
		out, err := exec.Command(ffmpeg,
			"-v", "error",
			"-i", filepath.Join(srcDir, "12254.mp4"),
			"-filter_complex", graph,
			"-map", "[vidHD]", "-map", "[aud0]", "-map", "[aud1]",
			"-f", "null", "-",
		).CombinedOutput()
		if err != nil {
			t.Errorf("filtergraph execution failed: %v\nfilter: %s\n%s", err, graph, out)
		}
	}
}
