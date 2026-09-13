// Package packaging implements the selective-export orchestration pipeline used
// by "iics package create": resolving a selection manifest against a source
// export's exportMetadata.v2.json, expanding container/closure/CDI dependency
// inclusion, pruning dangling references, and rebuilding the filtered package
// file set (exportMetadata.v2.json + ContentsofExportPackage_<name>.csv).
//
// This package has no Cobra or internal/output dependency so its core logic is
// unit-testable without a CLI harness or disk I/O.
package packaging

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/jbrazda/iics-cli/internal/release"
)

// ExportMetadata represents the exportMetadata.v2.json file in an IICS export package.
type ExportMetadata struct {
	Name            string           `json:"name"`
	SourceOrgID     string           `json:"sourceOrgId"`
	SourceOrgName   string           `json:"sourceOrgName"`
	Tags            []ExportTag      `json:"tags,omitempty"`
	ExportedObjects []ExportedObject `json:"exportedObjects"`
}

// ExportTag is a root-level tag entry in exportMetadata.v2.json.
type ExportTag struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// ExportedObject is one asset record in exportMetadata.v2.json.
type ExportedObject struct {
	ObjectGUID   string          `json:"objectGuid"`
	ObjectName   string          `json:"objectName"`
	ObjectType   string          `json:"objectType"`
	Path         string          `json:"path"`
	ProviderName json.RawMessage `json:"providerName,omitempty"`
	Metadata     json.RawMessage `json:"metadata"`
}

// ExportedObjectsToManifestLog converts exported objects into manifest log asset
// rows for release.RenderPackageCreateLog.
func ExportedObjectsToManifestLog(objects []ExportedObject) []release.ManifestLogAsset {
	rows := make([]release.ManifestLogAsset, 0, len(objects))
	for _, object := range objects {
		rows = append(rows, release.ManifestLogAsset{
			ID:   object.ObjectGUID,
			Type: object.ObjectType,
			Path: object.Path,
		})
	}
	return rows
}

// ObjectRefs extracts metadata.objectRefs while preserving all other metadata fields.
func (o ExportedObject) ObjectRefs() []string {
	metadata := bytes.TrimSpace(o.Metadata)
	if len(metadata) == 0 || bytes.Equal(metadata, []byte("null")) {
		return nil
	}
	var m struct {
		ObjectRefs []string `json:"objectRefs"`
	}
	if err := json.Unmarshal(metadata, &m); err != nil {
		return nil
	}
	return append([]string(nil), m.ObjectRefs...)
}

// SetObjectRefs updates metadata.objectRefs without dropping other metadata fields.
func (o *ExportedObject) SetObjectRefs(refs []string) error {
	metadata := bytes.TrimSpace(o.Metadata)
	m := make(map[string]json.RawMessage)
	if len(metadata) > 0 && !bytes.Equal(metadata, []byte("null")) {
		if err := json.Unmarshal(metadata, &m); err != nil {
			return fmt.Errorf("parsing metadata for %s: %w", o.ObjectGUID, err)
		}
	}
	rawRefs, err := json.Marshal(refs)
	if err != nil {
		return fmt.Errorf("serializing metadata refs for %s: %w", o.ObjectGUID, err)
	}
	m["objectRefs"] = rawRefs
	updated, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("serializing metadata for %s: %w", o.ObjectGUID, err)
	}
	o.Metadata = updated
	return nil
}
