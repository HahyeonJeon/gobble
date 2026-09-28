package appservice

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Service) stageLaunch(ctx context.Context, rec *launchRecord, p projectRecord) error {
	probe, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := s.runtime.verify(probe, rec.Runtime); err != nil {
		return err
	}
	label, err := s.runtime.commandAt(probe, rec.Runtime.Endpoint, "image", "inspect", "--format", `{{index .Config.Labels "io.gobble.launch.scope"}}`, rec.Runtime.ImageID)
	if err != nil || strings.TrimSpace(string(label)) != "single-end-trim-fastqc-v1" {
		return problem("unsupported", "This preparation engine cannot start reviewed Runs. Prepare again with an execution-capable engine.")
	}
	support, err := s.engineContinuationSupport(probe, rec.Runtime)
	if err != nil {
		return err
	}
	if support {
		rec.Intent.SchemaVersion = 2
	}
	for _, step := range rec.Value.Preparation.Prepared.Steps {
		raw, err := s.runtime.commandAt(probe, rec.Runtime.Endpoint, "image", "inspect", "--format", "{{.Id}} {{.Os}}/{{.Architecture}}", step.Image)
		if err != nil {
			return problem("runtime_unavailable", "A required analysis tool is not installed. Install the matching tools, then check again.")
		}
		fields := strings.Fields(string(raw))
		if len(fields) != 2 || !digestPattern.MatchString(fields[0]) || fields[1] != "linux/amd64" {
			return problem("incompatible_runtime", "An installed analysis tool has a different platform.")
		}
		rec.Intent.Tools[step.Image] = fields[0]
	}
	root, err := openProject(p)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := launchDiskSpace(p.Root, rec.Value.TotalBytes); err != nil {
		return err
	}
	if err := root.MkdirAll("runs", 0700); err != nil {
		return err
	}
	if err := root.Mkdir(rec.Value.OutputPath, 0700); err != nil {
		return problem("request_conflict", "The reserved result location already exists. No files were overwritten.")
	}
	info, err := root.Stat(rec.Value.OutputPath)
	if err != nil {
		return err
	}
	rec.TargetIdentity = rootIdentity(info)
	if err := root.Mkdir(rec.Value.OutputPath+"/inputs", 0700); err != nil {
		return err
	}
	marker, err := root.OpenFile(rec.Value.OutputPath+"/.gobble-launch-target", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = marker.WriteString(rec.Value.RequestID)
	if err == nil {
		err = marker.Sync()
	}
	marker.Close()
	if err != nil {
		return err
	}
	source := rec.Value.Preparation.Prepared.Input
	if observed, err := observeCreationInput(p, source.ResourceID); err != nil || observed != source {
		return problem("stale_revision", "The selected source changed before copying.")
	}
	src, err := openResource(root, source.RelativePath)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err = src.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != source.Size || digest([]byte(rootIdentity(info))) != source.Identity {
		return problem("stale_revision", "The opened source differs from the selected data.")
	}
	path := rec.Value.OutputPath + "/" + rec.Intent.Binding.InputPath
	dst, err := root.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			dst.Close()
		}
	}()
	last := time.Now()
	sum, size, reads, err := copyFASTQ(ctx, src, dst, strings.HasSuffix(strings.ToLower(source.RelativePath), ".gz"), func(n int64) {
		if time.Since(last) < 500*time.Millisecond {
			return
		}
		last = time.Now()
		s.mu.Lock()
		rec.Value.CopiedBytes = n
		_ = s.saveLaunch(*rec)
		s.mu.Unlock()
	})
	if err != nil {
		dst.Close()
		closed = true
		_ = root.Remove(path)
		_ = root.Remove(rec.Value.OutputPath + "/inputs")
		_ = root.Remove(rec.Value.OutputPath + "/.gobble-launch-target")
		_ = root.Remove(rec.Value.OutputPath)
		return err
	}
	if err = dst.Chmod(0444); err != nil {
		return err
	}
	if err = dst.Close(); err != nil {
		return err
	}
	closed = true
	if observed, err := observeCreationInput(p, source.ResourceID); err != nil || observed != source {
		return problem("stale_revision", "The selected source changed during the data copy.")
	}
	rec.Value.CopiedBytes = size
	rec.Value.ReadCount = reads
	rec.Value.InputSHA256 = sum
	rec.Intent.InputSHA256 = sum
	rec.Intent.InputSize = size
	raw, err := readBoundedFile(filepath.Join(s.preparationDirectory(rec.Value.PipelineID, rec.Value.PreparationID), "payload.json"), 3<<20)
	if err != nil || digest(raw) != rec.Intent.PreparedDigest {
		return problem("internal", "The retained prepared plan failed its integrity check.")
	}
	bundle := filepath.Join(s.launchDirectory(p.ProjectID, rec.Value.RequestID), "bundle")
	if err = os.Mkdir(bundle, 0755); err != nil {
		return err
	}
	if err = atomicWrite(bundle, "payload.json", raw); err != nil {
		return err
	}
	return os.Chmod(filepath.Join(bundle, "payload.json"), 0444)
}
func verifyLaunchTarget(ctx context.Context, p projectRecord, rec launchRecord) error {
	if err := launchTargetIdentity(p, rec); err != nil {
		return err
	}
	root, err := openProject(p)
	if err != nil {
		return err
	}
	defer root.Close()
	input := rec.Value.OutputPath + "/" + rec.Intent.Binding.InputPath
	allowed := map[string]bool{rec.Value.OutputPath: true, rec.Value.OutputPath + "/inputs": true, rec.Value.OutputPath + "/.gobble-launch-target": true, input: true}
	if err := fs.WalkDir(root.FS(), rec.Value.OutputPath, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !allowed[path] || d.Type()&os.ModeSymlink != 0 {
			return problem("stale_revision", "The result location contains unexpected files. No files were overwritten.")
		}
		return nil
	}); err != nil {
		return err
	}
	f, err := openResource(root, input)
	if err != nil {
		return err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || !stat.Mode().IsRegular() || stat.Size() != rec.Intent.InputSize {
		return problem("stale_revision", "The checked data copy changed.")
	}
	sum, err := hashLaunchInput(ctx, f)
	if err != nil || sum != rec.Intent.InputSHA256 {
		return problem("stale_revision", "The checked data copy changed. Check again.")
	}
	return nil
}
func writeLaunchIntent(dir string, rec launchRecord) error {
	data, err := json.Marshal(rec.Intent)
	if err != nil {
		return err
	}
	bundle := filepath.Join(dir, "bundle")
	if err := atomicWrite(bundle, "intent.json", data); err != nil {
		return err
	}
	return os.Chmod(filepath.Join(bundle, "intent.json"), 0444)
}
func launchError(err error) string {
	if err == nil {
		return ""
	}
	return publicPreparationIssue(err)
}
