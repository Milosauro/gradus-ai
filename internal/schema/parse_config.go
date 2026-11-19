package schema

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

var pyramidFieldSpec = FieldSpec{
	Allowed: map[string]FieldSpec{
		"schema":  {},
		"extends": {},
		"levels": {
			IsSeq: true,
			Allowed: map[string]FieldSpec{
				"id": {},
				"source": {
					Allowed: map[string]FieldSpec{
						"path": {},
						"git":  {},
						"ref":  {},
					},
				},
				"locked": {},
				"offline": {
					Allowed: map[string]FieldSpec{
						"mode":    {},
						"max_age": {},
					},
				},
				"approvers":      {},
				"editable_rules": {},
				"scope":          {},
			},
		},
		"overlays": {
			IsSeq: true,
			Allowed: map[string]FieldSpec{
				"id": {},
				"source": {
					Allowed: map[string]FieldSpec{
						"path": {},
						"git":  {},
						"ref":  {},
					},
				},
				"strength":   {},
				"activation": {},
			},
		},
		"profile": {}, // Extensible profile vocabulary (§6.4)
		"targets": {},
		"hooks": {
			Allowed: map[string]FieldSpec{
				"manager": {},
			},
		},
		"evidence": {
			Allowed: map[string]FieldSpec{
				"sinks": {
					IsSeq: true,
					Allowed: map[string]FieldSpec{
						"type":    {},
						"path":    {},
						"command": {},
					},
				},
			},
		},
	},
}

var levelFieldSpec = FieldSpec{
	Allowed: map[string]FieldSpec{
		"schema":    {},
		"id":        {},
		"version":   {},
		"owners":    {},
		"approvers": {},
		"charter": {
			Allowed: map[string]FieldSpec{
				"purpose":            {},
				"includes":           {},
				"excludes":           {},
				"examples_in":        {},
				"examples_out":       {},
				"allowed_severities": {},
				"review_cadence":     {},
			},
		},
		"constraints": {
			Allowed: map[string]FieldSpec{
				"require_levels":   {},
				"max_extra_levels": {},
				"allowed_overlays": {},
			},
		},
		"activation_rules": {},
		"exceptions": {
			Allowed: map[string]FieldSpec{
				"max_duration": {},
				"approvers":    {},
				"break_glass":  {},
			},
		},
	},
}

var overlayFieldSpec = FieldSpec{
	Allowed: map[string]FieldSpec{
		"schema":     {},
		"id":         {},
		"version":    {},
		"strength":   {},
		"owners":     {},
		"approvers":  {},
		"activation": {},
	},
}

// ParsePyramidConfig parses and validates gradus.yaml bytes.
func ParsePyramidConfig(data []byte) (ParseResult[PyramidConfig], error) {
	var res ParseResult[PyramidConfig]
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return res, fmt.Errorf("invalid gradus.yaml YAML: %w", err)
	}
	res.Notes = CollectUnknownKeys(&node, pyramidFieldSpec, "", true)

	if err := yaml.Unmarshal(data, &res.Value); err != nil {
		return res, fmt.Errorf("failed to decode gradus.yaml: %w", err)
	}
	if err := ValidatePyramidConfig(&res.Value); err != nil {
		return res, err
	}
	return res, nil
}

// ParseLevelConfig parses and validates level.yaml bytes.
func ParseLevelConfig(data []byte) (ParseResult[LevelConfig], error) {
	var res ParseResult[LevelConfig]
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return res, fmt.Errorf("invalid level.yaml YAML: %w", err)
	}
	res.Notes = CollectUnknownKeys(&node, levelFieldSpec, "", true)

	if err := yaml.Unmarshal(data, &res.Value); err != nil {
		return res, fmt.Errorf("failed to decode level.yaml: %w", err)
	}
	if err := ValidateLevelConfig(&res.Value); err != nil {
		return res, err
	}
	return res, nil
}

// ParseOverlayConfig parses and validates overlay.yaml bytes.
func ParseOverlayConfig(data []byte) (ParseResult[OverlayConfig], error) {
	var res ParseResult[OverlayConfig]
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return res, fmt.Errorf("invalid overlay.yaml YAML: %w", err)
	}
	res.Notes = CollectUnknownKeys(&node, overlayFieldSpec, "", false)

	if err := yaml.Unmarshal(data, &res.Value); err != nil {
		return res, fmt.Errorf("failed to decode overlay.yaml: %w", err)
	}
	if err := ValidateOverlayConfig(&res.Value); err != nil {
		return res, err
	}
	return res, nil
}
