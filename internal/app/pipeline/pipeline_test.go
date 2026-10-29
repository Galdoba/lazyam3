package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/domain/script"
	"github.com/Galdoba/lazyam3/internal/platform"
)

type fakeOrganizer struct {
	units map[string]ingest.MediaUnit
	errs  map[string]error
}

func (f *fakeOrganizer) Ingest(dir string) (ingest.MediaUnit, error) {
	if err, ok := f.errs[dir]; ok {
		return ingest.MediaUnit{}, err
	}
	if u, ok := f.units[dir]; ok {
		return u, nil
	}
	return ingest.MediaUnit{}, fmt.Errorf("unexpected dir %q", dir)
}

type fakeRepo struct {
	products map[int64]*metadata.Product
	errs     map[int64]error
}

func (f *fakeRepo) FindByID(id int64) (*metadata.Product, error) {
	if err, ok := f.errs[id]; ok {
		return nil, err
	}
	if p, ok := f.products[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("product %d not found", id)
}

type fakeRenderer struct {
	scripts []script.Script
	err     error
}

func (f *fakeRenderer) Render(sc script.Script) error {
	f.scripts = append(f.scripts, sc)
	return f.err
}

func testUnit() ingest.MediaUnit {
	return ingest.MediaUnit{
		ID:        12094,
		Filenames: []string{"12094.mp4", "12094-rus.srt"},
		Streams: map[string]ingest.Stream{
			"[1:s:0]": {FileIndex: 1, Index: 0, Kind: ingest.StreamKindSubtitle},
			"[0:a:1]": {FileIndex: 0, Index: 1, Kind: ingest.StreamKindAudio},
			"[0:v:0]": {FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo},
		},
	}
}

func testProduct() *metadata.Product {
	season, episode := 1, 6
	return &metadata.Product{
		ID:            12094,
		Kind:          metadata.KindSeriesEpisode,
		SeriesTitle:   "На грани",
		SeasonNumber:  &season,
		EpisodeNumber: &episode,
	}
}

// sourceRoot creates a scan fixture: <root>/ep1/12094.mp4 (writable)
// and returns (root, unitDir).
func sourceRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "ep1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "12094.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, dir
}

func newTestPipeline(t *testing.T, root string, in *fakeOrganizer, repo *fakeRepo, rd *fakeRenderer, events chan PipelineEvent, dryRun bool) *Pipeline {
	t.Helper()
	dest := t.TempDir()
	return NewPipeline(Options{
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Ingest:        in,
		Metadata:      repo,
		Renderer:      rd,
		Events:        events,
		SourceDir:     root,
		Destination:   dest,
		DryRun:        dryRun,
		HardsubTag:    "",
		ExecutorPaths: nil,
		PathBuilder:   platform.NewPathBuilder(dest),
	})
}

func drainEvents(ch chan PipelineEvent) []PipelineEvent {
	var out []PipelineEvent
	for {
		select {
		case e := <-ch:
			out = append(out, e)
		default:
			return out
		}
	}
}

func TestRunOnce_ProcessesUnit(t *testing.T) {
	root, unitDir := sourceRoot(t)
	// Create all files that the fake unit references.
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Processed != 1 || out.Failed != 0 {
		t.Errorf("RunOnce() = %+v, want Processed=1 Failed=0", out)
	}

	if len(rd.scripts) != 1 {
		t.Fatalf("renderer got %d scripts, want 1", len(rd.scripts))
	}
	sc := rd.scripts[0]
	// Destination should use OUTBASE (derived from predictions).
	if !strings.HasSuffix(sc.Destination, ".sh") {
		t.Errorf("Destination = %q, want .../.sh", sc.Destination)
	}
	if !strings.HasPrefix(sc.Content, "#!/bin/bash") {
		t.Errorf("Content does not start with shebang")
	}
	// SOURCE must be a call name (not the original filename).
	if strings.Contains(sc.Content, `SOURCE="12094.mp4"`) {
		t.Error("SOURCE should be a call name, not the original filename")
	}
	// SRT must be a call name (not the original filename).
	if strings.Contains(sc.Content, `SRT="12094-rus.srt"`) {
		t.Error("SRT should be a call name, not the original filename")
	}

	evs := drainEvents(events)
	if len(evs) != 2 || evs[0].Type != EventUnitDone || evs[0].ID != 12094 || evs[1].Type != EventScanDone {
		t.Errorf("events = %+v, want [UnitDone(12094) ScanDone]", evs)
	}
}

func TestRunOnce_MetadataError(t *testing.T) {
	root, unitDir := sourceRoot(t)
	// Create all files that the fake unit references (needed for fallback processing).
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{errs: map[int64]error{12094: errors.New("not found")}},
		rd, events, false)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Processed != 1 || out.Failed != 0 {
		t.Errorf("RunOnce() = %+v, want Processed=1 Failed=0", out)
	}
	if len(rd.scripts) != 1 {
		t.Errorf("expected 1 script from fallback, got %d", len(rd.scripts))
	}

	evs := drainEvents(events)
	if len(evs) != 2 || evs[0].Type != EventUnitDone || evs[1].Type != EventScanDone {
		t.Errorf("events = %+v, want [UnitDone ScanDone]", evs)
	}
}

func TestRunOnce_FallbackProduct_Naming(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{errs: map[int64]error{12094: errors.New("not found")}},
		rd, events, false)

	_, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if len(rd.scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(rd.scripts))
	}
	// Fallback Product uses dir basename "ep1" as SeriesTitle.
	// verify OUTBASE in script contains "Ep1" (transliterated + titlecased).
	script := rd.scripts[0]
	if !strings.Contains(script.Content, "Ep1") {
		t.Errorf("script should contain fallback base 'Ep1', got:\n%s", script.Content)
	}
}

func TestRunOnce_IngestError(t *testing.T) {
	root, unitDir := sourceRoot(t)
	events := make(chan PipelineEvent, 100)
	p := newTestPipeline(t, root,
		&fakeOrganizer{errs: map[string]error{unitDir: errors.New("probe failed")}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		&fakeRenderer{}, events, false)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Failed != 1 {
		t.Errorf("Failed = %d, want 1", out.Failed)
	}
	evs := drainEvents(events)
	if len(evs) != 2 || evs[0].Type != EventUnitFailed || evs[0].ID != 0 {
		t.Errorf("events = %+v, want [UnitFailed(0) ScanDone]", evs)
	}
}

func TestRunOnce_DryRunSkipsBuild(t *testing.T) {
	root, unitDir := sourceRoot(t)
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, true)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Processed != 1 || out.Failed != 0 {
		t.Errorf("RunOnce() = %+v, want Processed=1 Failed=0", out)
	}
	if len(rd.scripts) != 0 {
		t.Errorf("dry run must not render scripts, got %d", len(rd.scripts))
	}
}

func TestRunOnce_ScanError(t *testing.T) {
	p := NewPipeline(Options{
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Ingest:        &fakeOrganizer{},
		Metadata:      &fakeRepo{},
		Renderer:      &fakeRenderer{},
		SourceDir:     filepath.Join(t.TempDir(), "does-not-exist"),
		ExecutorPaths: nil,
	})
	if _, err := p.RunOnce(context.Background()); err == nil {
		t.Fatal("RunOnce() on missing source dir: expected error, got nil")
	}
}

func TestRun_CtxCancelled(t *testing.T) {
	p := newTestPipeline(t, t.TempDir(), &fakeOrganizer{}, &fakeRepo{}, &fakeRenderer{}, nil, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}

func TestNewPipeline_PollIntervalDefault(t *testing.T) {
	p := NewPipeline(Options{})
	if p.pollInterval != defaultPollInterval {
		t.Errorf("NewPipeline() pollInterval = %v, want %v", p.pollInterval, defaultPollInterval)
	}
	p2 := NewPipeline(Options{PollInterval: 5 * time.Second})
	if p2.pollInterval != 5*time.Second {
		t.Errorf("NewPipeline() pollInterval = %v, want %v", p2.pollInterval, 5*time.Second)
	}
}

func TestRun_LoopsUntilContextCancelled(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	p := NewPipeline(Options{
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Ingest:       &fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		Metadata:     &fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		Renderer:     &fakeRenderer{},
		Events:       events,
		SourceDir:    root,
		Destination:  t.TempDir(),
		DryRun:       false,
		PollInterval: 10 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var out RunOutput
	var runErr error
	go func() {
		defer close(done)
		out, runErr = p.Run(ctx)
	}()

	// First pass: unit processed, source files moved to the buffer directory.
	firstScanDone := make(chan struct{})
	go func() {
		for e := range events {
			if e.Type == EventScanDone {
				close(firstScanDone)
				return
			}
		}
	}()
	select {
	case <-firstScanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first scan done")
	}

	// Second pass: source is empty now (files were moved), scan done again.
	secondScanDone := make(chan struct{})
	go func() {
		for e := range events {
			if e.Type == EventScanDone {
				close(secondScanDone)
				return
			}
		}
	}()
	select {
	case <-secondScanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("expected second scan pass after poll interval")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", runErr)
	}
	if out.Processed != 1 {
		t.Errorf("Run() totals after two passes = %+v, want Processed=1", out)
	}
}

func TestEmptyDir_MarkerCreatedOnSuccess(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	_, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if len(rd.scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(rd.scripts))
	}
	scriptName := filepath.Base(rd.scripts[0].Destination)
	scriptName = strings.TrimSuffix(scriptName, filepath.Ext(rd.scripts[0].Destination))
	metaDir := filepath.Join(p.destination, "metadata")
	marker := filepath.Join(metaDir, scriptName+".processing")
	if !fileExists(marker) {
		t.Errorf("marker %q should exist after successful build", marker)
	}
}

func TestEmptyDir_MarkerNotCreatedOnFailure(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{err: errors.New("render failed")}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Failed != 1 {
		t.Errorf("RunOnce() Failed = %d, want 1", out.Failed)
	}
	metaDir := filepath.Join(p.destination, "metadata")
	if fileExists(metaDir) {
		t.Errorf("metadata dir %q should not exist after failure", metaDir)
	}
}

func TestEmptyDir_SecondScan_SkipsCompletedUnit(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	out1, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("first RunOnce() error: %v", err)
	}
	if out1.Processed != 1 || out1.Failed != 0 {
		t.Errorf("first RunOnce() = %+v, want Processed=1 Failed=0", out1)
	}

	// Second scan: files were moved to buffer, dir is now empty.
	out2, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("second RunOnce() error: %v", err)
	}
	if out2.Processed != 0 {
		t.Errorf("second RunOnce() Processed = %d, want 0 (empty dir skipped)", out2.Processed)
	}
	if len(rd.scripts) != 1 {
		t.Errorf("expected 1 script total, got %d", len(rd.scripts))
	}
}

func TestEmptyDir_DryRun_NoMarkerCreated(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, true)

	_, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	metaDir := filepath.Join(p.destination, "metadata")
	if fileExists(metaDir) {
		t.Errorf("metadata dir %q should not exist on dry run", metaDir)
	}
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func TestBuildScriptConfig(t *testing.T) {
	unit := testUnit()
	root := t.TempDir()
	cfg, err := buildScriptConfig(unit, *testProduct(), root, nil, "", "", script.HarsubConfig{OriginalSize: "1920x1080", FontName: "Segoe UI", FontSize: 19.4})
	if err != nil {
		t.Fatalf("buildScriptConfig() error: %v", err)
	}

	// SRT/HSUBFile must be call names, not original filenames.
	if cfg.SRT == "12094-rus.srt" {
		t.Error("SRT should be a call name, not the original filename")
	}
	if cfg.SRT == "" {
		t.Error("SRT must not be empty")
	}
	if len(cfg.CallNames) == 0 {
		t.Fatal("CallNames must be populated")
	}
	// CallNames must map original filenames to call names.
	for _, orig := range unit.Filenames {
		if _, ok := cfg.CallNames[orig]; !ok {
			t.Errorf("CallNames missing entry for %q", orig)
		}
	}
	// Destination should use OUTBASE, not unit ID.
	if cfg.Destination == "" {
		t.Error("Destination should not be empty")
	}
	if strings.Contains(cfg.Destination, "12094.sh") {
		t.Error("Destination should use OUTBASE, not unit ID")
	}
	// Without executor paths, env fields should default to root.
	if cfg.Env.Buffer != root || cfg.Env.InProgress != root {
		t.Errorf("Env paths should default to root without executor paths: %+v", cfg.Env)
	}

	// Streams sorted by (FileIndex, Index) regardless of map order.
	want := []ingest.Stream{
		{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo},
		{FileIndex: 0, Index: 1, Kind: ingest.StreamKindAudio},
		{FileIndex: 1, Index: 0, Kind: ingest.StreamKindSubtitle},
	}
	if len(cfg.Streams) != len(want) {
		t.Fatalf("Streams len = %d, want %d", len(cfg.Streams), len(want))
	}
	for i, w := range want {
		if cfg.Streams[i].FileIndex != w.FileIndex || cfg.Streams[i].Index != w.Index || cfg.Streams[i].Kind != w.Kind {
			t.Errorf("Streams[%d] = %+v, want %+v", i, cfg.Streams[i], w)
		}
	}
}

func TestBuildScriptConfig_RequiresRoot(t *testing.T) {
	if _, err := buildScriptConfig(testUnit(), *testProduct(), "", nil, "", "", script.HarsubConfig{OriginalSize: "1920x1080", FontName: "Segoe UI", FontSize: 19.4}); err == nil {
		t.Fatal("expected error for empty destination root, got nil")
	}
}

func TestBuildScriptConfig_ExecutorPaths(t *testing.T) {
	unit := testUnit()
	root := t.TempDir()
	execPaths := map[string]string{
		"buffer":        "/exec/buffer",
		"in_progress":   "/exec/in_progress",
		"done":          "/exec/done",
		"notifications": "/exec/notifications",
		"target":        "/exec/target",
	}
	cfg, err := buildScriptConfig(unit, *testProduct(), root, execPaths, "", "", script.HarsubConfig{})
	if err != nil {
		t.Fatalf("buildScriptConfig() error: %v", err)
	}

	if cfg.Env.Buffer != "/exec/buffer" {
		t.Errorf("Env.Buffer = %q, want /exec/buffer", cfg.Env.Buffer)
	}
	if cfg.Env.InProgress != "/exec/in_progress" {
		t.Errorf("Env.InProgress = %q, want /exec/in_progress", cfg.Env.InProgress)
	}
	if cfg.Env.Done != "/exec/done" {
		t.Errorf("Env.Done = %q, want /exec/done", cfg.Env.Done)
	}
	if cfg.Env.Notifications != "/exec/notifications" {
		t.Errorf("Env.Notifications = %q, want /exec/notifications", cfg.Env.Notifications)
	}
	if cfg.Env.TargetRoot != "/exec/target" {
		t.Errorf("Env.TargetRoot = %q, want /exec/target", cfg.Env.TargetRoot)
	}
}

func TestEnsureEnvDirs(t *testing.T) {
	root := t.TempDir()
	if err := EnsureEnvDirs(root); err != nil {
		t.Fatalf("EnsureEnvDirs() error: %v", err)
	}
	for _, dir := range []string{"buffer", "in_progress", "done/bash", "notifications", "target", "scripts"} {
		if fi, err := os.Stat(filepath.Join(root, dir)); err != nil || !fi.IsDir() {
			t.Errorf("env dir %q missing or not a directory (err=%v)", dir, err)
		}
	}
	if err := EnsureEnvDirs(""); err == nil {
		t.Error("EnsureEnvDirs(\"\"): expected error, got nil")
	}
}

func TestFindSRT(t *testing.T) {
	tests := []struct {
		names      []string
		hardsubTag string
		wantSrt    string
		wantHsub   string
	}{
		{[]string{"a.mp4", "a.srt"}, "", "a.srt", ""},
		{[]string{"a.mp4", "a.srt"}, "hardsub", "a.srt", ""},
		{[]string{"a.mp4", "a_hardsub.srt"}, "", "a_hardsub.srt", ""},
		{[]string{"a.mp4", "a_hardsub.srt"}, "hardsub", "", "a_hardsub.srt"},
		{[]string{"a.mp4"}, "", "", ""},
		{nil, "", "", ""},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v-%s", tt.names, tt.hardsubTag), func(t *testing.T) {
			gotSrt, gotHsub := findSubtitle(tt.names, tt.hardsubTag)
			if gotSrt != tt.wantSrt {
				t.Errorf("findSubtitle(%v, %q) srt = %q, want %q", tt.names, tt.hardsubTag, gotSrt, tt.wantSrt)
			}
			if gotHsub != tt.wantHsub {
				t.Errorf("findSubtitle(%v, %q) hsub = %q, want %q", tt.names, tt.hardsubTag, gotHsub, tt.wantHsub)
			}
		})
	}
}

func TestBuildScriptConfig_FfmpegBinary(t *testing.T) {
	unit := testUnit()
	root := t.TempDir()
	cfg, err := buildScriptConfig(unit, *testProduct(), root, nil, "", "/opt/bin/ffmpeg", script.HarsubConfig{})
	if err != nil {
		t.Fatalf("buildScriptConfig() error: %v", err)
	}
	if cfg.FfmpegBinary != "/opt/bin/ffmpeg" {
		t.Errorf("FfmpegBinary = %q, want /opt/bin/ffmpeg", cfg.FfmpegBinary)
	}
}

func TestBuildScriptConfig_FfmpegBinaryFallback(t *testing.T) {
	unit := testUnit()
	root := t.TempDir()
	cfg, err := buildScriptConfig(unit, *testProduct(), root, nil, "", "", script.HarsubConfig{OriginalSize: "1920x1080", FontName: "Segoe UI", FontSize: 19.4})
	if err != nil {
		t.Fatalf("buildScriptConfig() error: %v", err)
	}
	if cfg.FfmpegBinary != "" {
		t.Errorf("FfmpegBinary = %q, want empty (fallback handled in BuildScript)", cfg.FfmpegBinary)
	}
}

func TestBuildScriptConfig_FfmpegBinaryEmpty(t *testing.T) {
	// Empty ffmpeg binary should still produce a valid config.
	// The fallback to "ffmpeg" happens in BuildScript, not here.
	unit := testUnit()
	root := t.TempDir()
	cfg, err := buildScriptConfig(unit, *testProduct(), root, nil, "", "", script.HarsubConfig{OriginalSize: "1920x1080", FontName: "Segoe UI", FontSize: 19.4})
	if err != nil {
		t.Fatalf("buildScriptConfig() error: %v", err)
	}
	if cfg.FfmpegBinary != "" {
		t.Errorf("FfmpegBinary = %q, want empty", cfg.FfmpegBinary)
	}
	// But BuildScript should still produce valid output with fallback.
	script, err := script.BuildScript(cfg)
	if err != nil {
		t.Fatalf("BuildScript() error: %v", err)
	}
	if !strings.Contains(script.Content, "ffmpeg") {
		t.Error("BuildScript() with empty FfmpegBinary should fallback to 'ffmpeg'")
	}
}

func TestNewPipeline_DormancyPeriodDefault(t *testing.T) {
	p := NewPipeline(Options{})
	if p.dormancyPeriod != 0 {
		t.Errorf("NewPipeline() dormancyPeriod = %v, want 0 (disabled)", p.dormancyPeriod)
	}
	p2 := NewPipeline(Options{DormancyPeriod: 5 * time.Second})
	if p2.dormancyPeriod != 5*time.Second {
		t.Errorf("NewPipeline() dormancyPeriod = %v, want 5s", p2.dormancyPeriod)
	}
	// Zero value means disabled — pipeline should use polling, not dormancy.
	if p2.dormancyPeriod <= 0 {
		t.Error("non-zero DormancyPeriod should be > 0")
	}
}

func TestRun_DormancyMode_SleepsBetweenPasses(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	dormancy := 100 * time.Millisecond
	p := NewPipeline(Options{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Ingest:         &fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		Metadata:       &fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		Renderer:       &fakeRenderer{},
		Events:         events,
		SourceDir:      root,
		Destination:    t.TempDir(),
		DryRun:         false,
		DormancyPeriod: dormancy,
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var runErr error
	go func() {
		defer close(done)
		_, runErr = p.Run(ctx)
	}()

	// Wait for first scan done.
	firstScanDone := make(chan struct{})
	go func() {
		for e := range events {
			if e.Type == EventScanDone {
				close(firstScanDone)
				return
			}
		}
	}()
	select {
	case <-firstScanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first scan done")
	}

	// After dormancy, wait for second scan done.
	secondScanDone := make(chan struct{})
	go func() {
		for e := range events {
			if e.Type == EventScanDone {
				close(secondScanDone)
				return
			}
		}
	}()
	select {
	case <-secondScanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("expected second scan after dormancy period")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", runErr)
	}
}

func TestRun_DormancyMode_ContextCancelledDuringSleep(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	dormancy := 10 * time.Second
	p := NewPipeline(Options{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Ingest:         &fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		Metadata:       &fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		Renderer:       &fakeRenderer{},
		Events:         events,
		SourceDir:      root,
		Destination:    t.TempDir(),
		DryRun:         false,
		DormancyPeriod: dormancy,
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var runErr error
	go func() {
		defer close(done)
		_, runErr = p.Run(ctx)
	}()

	// Wait for first scan done.
	firstScanDone := make(chan struct{})
	go func() {
		for e := range events {
			if e.Type == EventScanDone {
				close(firstScanDone)
				return
			}
		}
	}()
	select {
	case <-firstScanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first scan done")
	}

	// Cancel during dormancy sleep.
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation during dormancy")
	}
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", runErr)
	}
}

func TestRun_DormancyDisabled_Polling(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	p := NewPipeline(Options{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Ingest:         &fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		Metadata:       &fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		Renderer:       &fakeRenderer{},
		Events:         events,
		SourceDir:      root,
		Destination:    t.TempDir(),
		DryRun:         false,
		DormancyPeriod: 0,
		PollInterval:   10 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Run(ctx)
	}()

	// Wait for two scan passes (polling, not dormancy).
	firstScanDone := make(chan struct{})
	go func() {
		for e := range events {
			if e.Type == EventScanDone {
				close(firstScanDone)
				return
			}
		}
	}()
	select {
	case <-firstScanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first scan done")
	}

	secondScanDone := make(chan struct{})
	go func() {
		for e := range events {
			if e.Type == EventScanDone {
				close(secondScanDone)
				return
			}
		}
	}()
	select {
	case <-secondScanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("expected second scan pass after poll interval")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestRun_DormancyMode_ContinuesAfterScanFailure(t *testing.T) {
	root := t.TempDir()
	events := make(chan PipelineEvent, 100)
	dormancy := 100 * time.Millisecond
	p := NewPipeline(Options{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Ingest:         &fakeOrganizer{},
		Metadata:       &fakeRepo{},
		Renderer:       &fakeRenderer{},
		Events:         events,
		SourceDir:      root,
		Destination:    t.TempDir(),
		DryRun:         false,
		DormancyPeriod: dormancy,
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var runErr error
	go func() {
		defer close(done)
		_, runErr = p.Run(ctx)
	}()

	// Wait for first scan done (scan will have found 0 units but still emit ScanDone).
	firstScanDone := make(chan struct{})
	go func() {
		for e := range events {
			if e.Type == EventScanDone {
				close(firstScanDone)
				return
			}
		}
	}()
	select {
	case <-firstScanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first scan done")
	}

	// After dormancy, second scan should happen even after first had no units.
	secondScanDone := make(chan struct{})
	go func() {
		for e := range events {
			if e.Type == EventScanDone {
				close(secondScanDone)
				return
			}
		}
	}()
	select {
	case <-secondScanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("expected second scan after dormancy despite first pass having no units")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", runErr)
	}
}

// pathExists returns true if path exists (file or dir).
func pathExists(path string) bool {
	e, err := os.Lstat(path)
	return err == nil && e != nil
}

func emptyDirFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	emptyDir := filepath.Join(root, "empty-unit")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, emptyDir
}

func writeDoneFile(t *testing.T, metaDir, scriptName, sourceDir string) string {
	t.Helper()
	donePath := filepath.Join(metaDir, scriptName+".done")
	payload := fmt.Sprintf(`{"source_dir": "%s", "unit_id": 1, "script_name": "test"}`, sourceDir)
	if err := os.WriteFile(donePath, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	return donePath
}

func TestSanitize_EmptyDirWithDoneFile_Deleted(t *testing.T) {
	root, emptyDir := emptyDirFixture(t)
	absEmptyDir, err := filepath.Abs(emptyDir)
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	metaDir := filepath.Join(dest, "metadata")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	donePath := writeDoneFile(t, metaDir, "test.done", absEmptyDir)
	events := make(chan PipelineEvent, 100)
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{}},
		&fakeRepo{products: map[int64]*metadata.Product{}},
		&fakeRenderer{}, events, false)
	p.destination = dest
	p.pathBuilder = platform.NewPathBuilder(dest)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Processed != 0 || out.Failed != 0 {
		t.Errorf("RunOnce() = %+v, want Processed=0 Failed=0", out)
	}
	if pathExists(emptyDir) {
		t.Errorf("empty dir %q should be deleted", emptyDir)
	}
	if pathExists(donePath) {
		t.Errorf("done file %q should be consumed", donePath)
	}
}

func TestSanitize_EmptyDirWithoutDoneFile_Kept(t *testing.T) {
	root, emptyDir := emptyDirFixture(t)
	dest := t.TempDir()
	metaDir := filepath.Join(dest, "metadata")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	events := make(chan PipelineEvent, 100)
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{}},
		&fakeRepo{products: map[int64]*metadata.Product{}},
		&fakeRenderer{}, events, false)
	p.destination = dest
	p.pathBuilder = platform.NewPathBuilder(dest)

	_, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if !pathExists(emptyDir) {
		t.Errorf("empty dir %q should be kept (no .done file)", emptyDir)
	}
}

func TestSanitize_InvalidJSONDoneFile_Skipped(t *testing.T) {
	root, emptyDir := emptyDirFixture(t)
	dest := t.TempDir()
	metaDir := filepath.Join(dest, "metadata")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	donePath := filepath.Join(metaDir, "bad.done")
	if err := os.WriteFile(donePath, []byte("not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	events := make(chan PipelineEvent, 100)
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{}},
		&fakeRepo{products: map[int64]*metadata.Product{}},
		&fakeRenderer{}, events, false)
	p.destination = dest
	p.pathBuilder = platform.NewPathBuilder(dest)

	if _, err := p.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if !pathExists(emptyDir) {
		t.Errorf("empty dir %q should be kept (invalid .done JSON)", emptyDir)
	}
}

func TestSanitize_DoneFileConsumedAfterDeletion(t *testing.T) {
	root, emptyDir := emptyDirFixture(t)
	absEmptyDir, err := filepath.Abs(emptyDir)
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	metaDir := filepath.Join(dest, "metadata")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeDoneFile(t, metaDir, "test2.done", absEmptyDir)
	events := make(chan PipelineEvent, 100)
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{}},
		&fakeRepo{products: map[int64]*metadata.Product{}},
		&fakeRenderer{}, events, false)
	p.destination = dest
	p.pathBuilder = platform.NewPathBuilder(dest)

	if _, err := p.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if pathExists(filepath.Join(metaDir, "test2.done")) {
		t.Errorf("consumed .done file should be deleted")
	}
}

func readProcessingArtifact(t *testing.T, metaDir, scriptName string) map[string]any {
	marker := filepath.Join(metaDir, scriptName+".processing")
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read processing artifact %q: %v", marker, err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("parse processing artifact %q: %v", marker, err)
	}
	return payload
}

func TestProcessUnitArtifactProduct(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	if _, err := p.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	scriptName := filepath.Base(rd.scripts[0].Destination)
	scriptName = strings.TrimSuffix(scriptName, filepath.Ext(rd.scripts[0].Destination))
	metaDir := filepath.Join(p.destination, "metadata")
	payload := readProcessingArtifact(t, metaDir, scriptName)
	productName, ok := payload["product_name"].(string)
	if !ok || productName == "" {
		t.Errorf("product_name missing or empty: %v", payload["product_name"])
	}
}

func TestProcessUnitArtifactCreatedAt(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	if _, err := p.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	scriptName := filepath.Base(rd.scripts[0].Destination)
	scriptName = strings.TrimSuffix(scriptName, filepath.Ext(rd.scripts[0].Destination))
	metaDir := filepath.Join(p.destination, "metadata")
	payload := readProcessingArtifact(t, metaDir, scriptName)
	createdAt, ok := payload["created_at"].(string)
	if !ok {
		t.Fatalf("created_at missing or not a string: %v", payload["created_at"])
	}
	// Verify RFC3339 format with milliseconds and UTC timezone.
	if !strings.Contains(createdAt, "T") || !strings.HasSuffix(createdAt, "Z") {
		t.Errorf("created_at %q not in RFC3339 UTC format", createdAt)
	}
	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Errorf("created_at %q failed to parse as RFC3339: %v", createdAt, err)
	}
}

func TestProcessUnitArtifactInitialFiles(t *testing.T) {
	root, unitDir := sourceRoot(t)
	files := []string{"12094.mp4", "12094-rus.srt"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	if _, err := p.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	scriptName := filepath.Base(rd.scripts[0].Destination)
	scriptName = strings.TrimSuffix(scriptName, filepath.Ext(rd.scripts[0].Destination))
	metaDir := filepath.Join(p.destination, "metadata")
	payload := readProcessingArtifact(t, metaDir, scriptName)
	initialFiles, ok := payload["initial_files"].([]any)
	if !ok {
		t.Fatalf("initial_files missing or not a list: %v", payload["initial_files"])
	}
	if len(initialFiles) != 2 {
		t.Errorf("initial_files has %d entries, want 2", len(initialFiles))
	}
	for _, f := range initialFiles {
		pathStr, ok := f.(string)
		if !ok || !filepath.IsAbs(pathStr) {
			t.Errorf("initial_files entry %v not an absolute path string", f)
		}
	}
}

func TestProcessUnitArtifactCreatedFiles(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	if _, err := p.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	scriptName := filepath.Base(rd.scripts[0].Destination)
	scriptName = strings.TrimSuffix(scriptName, filepath.Ext(rd.scripts[0].Destination))
	metaDir := filepath.Join(p.destination, "metadata")
	payload := readProcessingArtifact(t, metaDir, scriptName)
	createdFiles, ok := payload["created_files"].([]any)
	if !ok {
		t.Fatalf("created_files missing or not a list: %v", payload["created_files"])
	}
	if len(createdFiles) == 0 {
		t.Errorf("created_files is empty, want predictions")
	}
	for _, f := range createdFiles {
		pathStr, ok := f.(string)
		if !ok || !filepath.IsAbs(pathStr) {
			t.Errorf("created_files entry %v not an absolute path string", f)
		}
	}
}

func TestProcessUnitArtifactBackwardCompat(t *testing.T) {
	root, unitDir := sourceRoot(t)
	for _, name := range []string{"12094.mp4", "12094-rus.srt"} {
		if err := os.WriteFile(filepath.Join(unitDir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	if _, err := p.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	scriptName := filepath.Base(rd.scripts[0].Destination)
	scriptName = strings.TrimSuffix(scriptName, filepath.Ext(rd.scripts[0].Destination))
	metaDir := filepath.Join(p.destination, "metadata")
	payload := readProcessingArtifact(t, metaDir, scriptName)
	if payload["source_dir"] != unitDir {
		t.Errorf("source_dir = %v, want %q", payload["source_dir"], unitDir)
	}
	if payload["unit_id"] != float64(12094) {
		t.Errorf("unit_id = %v, want 12094", payload["unit_id"])
	}
}
