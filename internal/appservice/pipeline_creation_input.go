package appservice

import (
	"path/filepath"
	"strings"
	"time"
)

// creationInput is a metadata observation, not a hash of FASTQ contents or proof
// of scientific validity. Identity is opaque and never exposes a host path.
type creationInput struct {
	ResourceID   string `json:"resourceId"`
	RelativePath string `json:"relativePath"`
	ReadLayout   string `json:"readLayout"`
	Size         int64  `json:"size"`
	ModifiedAt   string `json:"modifiedAt"`
	Identity     string `json:"identity"`
}

func supportedReadFile(path string) bool {
	path = strings.ToLower(path)
	for _, suffix := range []string{".fastq", ".fq", ".fastq.gz", ".fq.gz"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}
func validCreationInput(input creationInput, p projectRecord) bool {
	_, err := time.Parse(time.RFC3339Nano, input.ModifiedAt)
	return input.ResourceID == resourceID(p.ProjectID, input.RelativePath) && p.Resources[input.ResourceID] == input.RelativePath && validRelative(input.RelativePath) && supportedReadFile(input.RelativePath) && input.ReadLayout == "single-end" && input.Size >= 0 && input.Size <= maxDraftGeneration && digestPattern.MatchString(input.Identity) && err == nil
}
func observeCreationInput(p projectRecord, resourceID string) (creationInput, error) {
	path, ok := p.Resources[resourceID]
	if !ok {
		return creationInput{}, problem("not_found", "Selected file not found in this Project.")
	}
	if !supportedReadFile(path) {
		return creationInput{}, problem("unsupported", "Choose one single-end FASTQ or FQ file, optionally gzip-compressed.")
	}
	root, err := openProject(p)
	if err != nil {
		return creationInput{}, err
	}
	defer root.Close()
	file, err := openResource(root, path)
	if err != nil {
		return creationInput{}, fileAccessError(err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return creationInput{}, fileAccessError(err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxDraftGeneration || rootIdentity(info) == "" {
		return creationInput{}, problem("unsupported", "Choose a regular Project read file.")
	}
	// Compare the opened descriptor with the currently named file, detecting replacement.
	named, err := root.Stat(filepath.FromSlash(path))
	if err != nil || rootIdentity(named) != rootIdentity(info) || named.Size() != info.Size() || !named.ModTime().Equal(info.ModTime()) {
		return creationInput{}, problem("stale_revision", "The selected file changed. Review the selected file again.")
	}
	return creationInput{resourceID, path, "single-end", info.Size(), info.ModTime().UTC().Format(time.RFC3339Nano), digest([]byte(rootIdentity(info)))}, nil
}
