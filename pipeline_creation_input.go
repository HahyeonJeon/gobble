package gobble

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

// ReadCreationInput reads only the fixed, service-retained input descriptor.
// It never opens research data. Creation evaluation must mount this file read-only.
func ReadCreationInput() (string, error) {
	file, err := os.Open(pipelinereview.CreationInputFile)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1024 {
		return "", fmt.Errorf("invalid creation input descriptor")
	}
	var input pipelinereview.CreationInput
	decoder := json.NewDecoder(io.LimitReader(file, 1025))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF || input.SchemaVersion != 1 || input.ReadLayout != "single-end" || !pipelinereview.ValidCreationPath(input.Path) {
		return "", fmt.Errorf("invalid creation input descriptor")
	}
	return input.Path, nil
}
