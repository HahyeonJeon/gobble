package appservice

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type fileEntry struct {
	ResourceID string `json:"resourceId"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Size       int64  `json:"size"`
}

type directoryResult struct {
	ProjectID   string      `json:"projectId"`
	DirectoryID string      `json:"directoryId"`
	Entries     []fileEntry `json:"entries"`
	Truncated   bool        `json:"truncated"`
}

type fileResult struct {
	ProjectID  string `json:"projectId"`
	ResourceID string `json:"resourceId"`
	Name       string `json:"name"`
	Revision   string `json:"revision"`
	Size       int    `json:"size"`
	Content    any    `json:"content"`
}

type textContent struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}
type tableColumn struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type tableRow struct {
	Key   string   `json:"key"`
	Cells []string `json:"cells"`
}
type tableContent struct {
	Kind      string        `json:"kind"`
	Columns   []tableColumn `json:"columns"`
	Rows      []tableRow    `json:"rows"`
	Truncated bool          `json:"truncated"`
}
type notebookContent struct {
	Kind   string `json:"kind"`
	Base64 string `json:"base64"`
}

type pdfContent struct {
	Kind   string `json:"kind"`
	Base64 string `json:"base64"`
}

type imageContent struct {
	Kind      string `json:"kind"`
	MediaType string `json:"mediaType"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Base64    string `json:"base64"`
}

func (s *store) listFiles(projectID, directoryID string) (directoryResult, error) {
	p, err := s.project(projectID)
	if err != nil {
		return directoryResult{}, err
	}
	if directoryID == "" {
		directoryID = p.RootResourceID
	}
	path, ok := p.Resources[directoryID]
	if !ok {
		return directoryResult{}, problem("not_found", "Directory not found in this Project.")
	}
	root, err := openProject(p)
	if err != nil {
		return directoryResult{}, err
	}
	defer root.Close()
	file, err := openResource(root, path)
	if err != nil {
		return directoryResult{}, fileAccessError(err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.IsDir() {
		return directoryResult{}, problem("invalid_request", "The selected resource is not a directory.")
	}
	entries, err := file.ReadDir(maxDirectoryEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return directoryResult{}, fileAccessError(err)
	}
	truncated := len(entries) > maxDirectoryEntries
	if truncated {
		entries = entries[:maxDirectoryEntries]
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	out := directoryResult{projectID, directoryID, []fileEntry{}, truncated}
	resources := map[string]string{}
	for _, entry := range entries {
		rel := filepath.ToSlash(filepath.Join(path, entry.Name()))
		if !validRelative(rel) {
			continue
		}
		id := resourceID(projectID, rel)
		kind, size := "unsupported", int64(0)
		info, err := entry.Info()
		if err == nil {
			size = info.Size()
			if size < 0 {
				size = 0
			}
			if info.Mode().IsRegular() {
				kind = "file"
			}
			if info.IsDir() {
				kind = "directory"
			}
		}
		out.Entries = append(out.Entries, fileEntry{id, entry.Name(), kind, size})
		if kind != "unsupported" {
			resources[id] = rel
		}
	}
	if err := s.rememberResources(projectID, resources); err != nil {
		return directoryResult{}, err
	}
	return out, nil
}

func (s *store) readFile(projectID, id, expectedRevision string) (fileResult, error) {
	p, err := s.project(projectID)
	if err != nil {
		return fileResult{}, err
	}
	path, ok := p.Resources[id]
	if !ok {
		return fileResult{}, problem("not_found", "File not found in this Project.")
	}
	root, err := openProject(p)
	if err != nil {
		return fileResult{}, err
	}
	defer root.Close()
	file, err := openResource(root, path)
	if err != nil {
		return fileResult{}, fileAccessError(err)
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return fileResult{}, fileAccessError(err)
	}
	if !before.Mode().IsRegular() {
		return fileResult{}, problem("unsupported", "Only regular files can be viewed.")
	}
	if before.Size() > maxFileBytes {
		return fileResult{}, problem("unsupported", "This file exceeds the 8 MiB preview limit.")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil {
		return fileResult{}, fileAccessError(err)
	}
	if len(data) > maxFileBytes {
		return fileResult{}, problem("unsupported", "This file exceeds the 8 MiB preview limit.")
	}
	after, err := file.Stat()
	if err != nil || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return fileResult{}, problem("stale_revision", "The file changed while it was being read. Refresh the view.")
	}
	revision := digest(data)
	if expectedRevision != "" && expectedRevision != revision {
		return fileResult{}, problem("stale_revision", "The selected file version has changed.")
	}
	content, err := decodeContent(path, data)
	if err != nil {
		return fileResult{}, err
	}
	return fileResult{projectID, id, filepath.Base(path), revision, len(data), content}, nil
}

func decodeContent(path string, data []byte) (any, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".ipynb" {
		return notebookContent{"notebook", base64.StdEncoding.EncodeToString(data)}, nil
	}
	if ext == ".pdf" {
		if !bytes.HasPrefix(data, []byte("%PDF-")) {
			return nil, problem("unsupported", "The PDF header is invalid.")
		}
		return pdfContent{"pdf", base64.StdEncoding.EncodeToString(data)}, nil
	}
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
		config, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || (format != "png" && format != "jpeg") || config.Width <= 0 || config.Height <= 0 || config.Width > 100_000 || config.Height > 100_000 || int64(config.Width)*int64(config.Height) > 20_000_000 {
			return nil, problem("unsupported", "The image is invalid or exceeds the 20 megapixel preview limit.")
		}
		return imageContent{"image", "image/" + format, config.Width, config.Height, base64.StdEncoding.EncodeToString(data)}, nil
	}
	if !utf8.Valid(data) || bytes.ContainsRune(data, 0) {
		return nil, problem("unsupported", "This file is not a supported UTF-8 text, CSV, PNG or JPEG file.")
	}
	if ext == ".csv" {
		return readCSV(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}))
	}
	if len(data) > 1<<20 {
		return nil, problem("unsupported", "Text previews are limited to 1 MiB.")
	}
	return textContent{"text", string(data)}, nil
}

func readCSV(data []byte) (tableContent, error) {
	r := csv.NewReader(bytes.NewReader(data))
	header, err := r.Read()
	if err != nil || len(header) > 100 {
		return tableContent{}, problem("unsupported", "The CSV header is invalid or has more than 100 columns.")
	}
	out := tableContent{Kind: "table", Columns: []tableColumn{}, Rows: []tableRow{}}
	for i, name := range header {
		out.Columns = append(out.Columns, tableColumn{"col_" + strconv.Itoa(i), name})
	}
	for i := 0; i <= 500; i++ {
		cells, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return tableContent{}, problem("invalid_request", "The CSV contains malformed rows.")
		}
		if i == 500 {
			out.Truncated = true
			break
		}
		out.Rows = append(out.Rows, tableRow{"row_" + strconv.Itoa(i+1), cells})
	}
	return out, nil
}

func fileAccessError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return problem("not_found", "The resource is no longer available.")
	}
	return problem("outside_project", "The resource cannot be accessed inside the registered Project.")
}
