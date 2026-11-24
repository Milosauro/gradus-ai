package schema

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// ReservedTopLevelKeys lists gradus/v1 keys reserved for Phase 3+ (§9.11).
var ReservedTopLevelKeys = map[string]bool{
	"models":   true,
	"adapters": true,
}

// FieldSpec describes allowed keys and optional nested mapping specs for a YAML object.
type FieldSpec struct {
	Allowed map[string]FieldSpec
	IsSeq   bool
}

// CollectUnknownKeys walks a YAML mapping or sequence node and emits diagnostic notes
// for unknown keys and Phase 0 reserved keys without failing validation (§9.2, §9.11).
func CollectUnknownKeys(node *yaml.Node, spec FieldSpec, prefix string, checkReserved bool) []string {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	var notes []string

	switch node.Kind {
	case yaml.MappingNode:
		if spec.Allowed == nil {
			// Free-form mapping (e.g., applies_when, profile extensions, overridable object).
			return nil
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]
			key := keyNode.Value
			fullPath := key
			if prefix != "" {
				fullPath = prefix + "." + key
			}

			if checkReserved && prefix == "" && ReservedTopLevelKeys[key] {
				notes = append(notes, fmt.Sprintf("reserved key %q is not active in Phase 0", fullPath))
				continue
			}

			childSpec, ok := spec.Allowed[key]
			if !ok {
				notes = append(notes, fmt.Sprintf("unknown key %q", fullPath))
				continue
			}
			notes = append(notes, CollectUnknownKeys(valNode, childSpec, fullPath, false)...)
		}
	case yaml.SequenceNode:
		if spec.IsSeq && spec.Allowed != nil {
			itemSpec := FieldSpec{Allowed: spec.Allowed}
			for idx, item := range node.Content {
				itemPrefix := fmt.Sprintf("%s[%d]", prefix, idx)
				notes = append(notes, CollectUnknownKeys(item, itemSpec, itemPrefix, false)...)
			}
		}
	}

	return notes
}
