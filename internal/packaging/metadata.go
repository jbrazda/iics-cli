package packaging

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ReadExportMetadata reads exportMetadata.v2.json from a ZIP file (filePath) or an
// expanded workspace directory (workspace). Exactly one of filePath/workspace should
// be non-empty; workspace takes precedence when both are set.
func ReadExportMetadata(filePath, workspace string) (*ExportMetadata, error) {
	if workspace != "" {
		data, err := os.ReadFile(filepath.Join(workspace, "exportMetadata.v2.json"))
		if err != nil {
			return nil, fmt.Errorf("reading exportMetadata.v2.json: %w", err)
		}
		var meta ExportMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, fmt.Errorf("parsing exportMetadata.v2.json: %w", err)
		}
		return &meta, nil
	}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening package file: %w", err)
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if f.Name == "exportMetadata.v2.json" {
			rc, oErr := f.Open()
			if oErr != nil {
				return nil, fmt.Errorf("opening exportMetadata.v2.json in ZIP: %w", oErr)
			}
			data, rErr := io.ReadAll(rc)
			_ = rc.Close()
			if rErr != nil {
				return nil, fmt.Errorf("reading exportMetadata.v2.json from ZIP: %w", rErr)
			}
			var meta ExportMetadata
			if err := json.Unmarshal(data, &meta); err != nil {
				return nil, fmt.Errorf("parsing exportMetadata.v2.json: %w", err)
			}
			return &meta, nil
		}
	}
	return nil, fmt.Errorf("exportMetadata.v2.json not found in package")
}
