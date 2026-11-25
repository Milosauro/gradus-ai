package assess

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

// DimensionScore captures the process maturity score (0..20) for one governance dimension (X-13).
type DimensionScore struct {
	Name    string   `json:"name"`
	Score   int      `json:"score"`
	Max     int      `json:"max"`
	Signals []string `json:"signals"`
}

// Report is a process-only governance maturity assessment (Design X-13, Principle P16: never evaluates individuals).
type Report struct {
	Schema          string           `json:"schema"`
	Repository      string           `json:"repository"`
	ViewHash        string           `json:"view_hash"`
	MaturityLevel   string           `json:"maturity_level"`
	TotalScore      int              `json:"total_score"`
	MaxScore        int              `json:"max_score"`
	Dimensions      []DimensionScore `json:"dimensions"`
	Recommendations []string         `json:"recommendations"`
}

// Evaluate inspects the Gradus pyramid, compiled/locked state, and local process evidence
// in rootDir and returns a deterministic process maturity Report.
func Evaluate(rootDir string) (Report, error) {
	var rep Report
	cfgBytes, err := os.ReadFile(filepath.Join(rootDir, "gradus.yaml"))
	if err != nil {
		return rep, fmt.Errorf("failed to read gradus.yaml: %w", err)
	}
	cfgRes, err := schema.ParsePyramidConfig(cfgBytes)
	if err != nil {
		return rep, err
	}
	cfg := cfgRes.Value

	res, err := resolve.Resolve(rootDir, resolve.Options{OfflineOnly: true})
	if err != nil {
		return rep, err
	}
	view := res.View

	var recs []string

	// 1. Pyramid Integrity (0..20)
	d1 := DimensionScore{Name: "pyramid_integrity", Max: 20}
	d1.Score += 5
	d1.Signals = append(d1.Signals, "valid gradus.yaml")
	if _, lockErr := os.Stat(filepath.Join(rootDir, "gradus.lock")); lockErr == nil {
		d1.Score += 5
		d1.Signals = append(d1.Signals, "gradus.lock present")
	} else {
		recs = append(recs, "Run `gradusctl resolve --write` to pin resolved levels in gradus.lock.")
	}
	if len(view.Conflicts) == 0 {
		d1.Score += 5
		d1.Signals = append(d1.Signals, "zero blocking conflicts")
	} else {
		recs = append(recs, "Resolve all blocking conflicts reported by `gradusctl verify`.")
	}
	hasLockedOrSig := false
	for _, l := range cfg.Levels {
		if l.Locked || l.Signature == "required" {
			hasLockedOrSig = true
			break
		}
	}
	if hasLockedOrSig || len(cfg.Levels) >= 2 {
		d1.Score += 5
		d1.Signals = append(d1.Signals, "multi-level or locked level governance")
	} else {
		recs = append(recs, "Adopt a shared locked level (team/org) with optional signature verification.")
	}

	// 2. Enforcement Depth (0..20)
	d2 := DimensionScore{Name: "enforcement_depth", Max: 20}
	hasCheck, hasGuard, hasExamples := false, false, false
	for _, r := range view.Rules {
		if r.Enforcement == "check" {
			hasCheck = true
		}
		if r.Enforcement == "hook" || r.Enforcement == "permission" {
			hasGuard = true
		}
	}
	if pyr, perr := schema.LoadLocalPyramid(rootDir); perr == nil {
		for _, r := range pyr.Rules {
			if len(r.Examples.Compliant) > 0 && len(r.Examples.NonCompliant) > 0 {
				hasExamples = true
				break
			}
		}
	}
	if len(view.Rules) > 0 {
		d2.Score += 5
		d2.Signals = append(d2.Signals, fmt.Sprintf("%d active rule(s)", len(view.Rules)))
	}
	if hasCheck {
		d2.Score += 5
		d2.Signals = append(d2.Signals, "deterministic check rule(s) active")
	} else {
		recs = append(recs, "Promote deterministic conventions to `enforcement: check` rules.")
	}
	if hasGuard {
		d2.Score += 5
		d2.Signals = append(d2.Signals, "runtime hook/permission guard(s) active")
	} else {
		recs = append(recs, "Add runtime `hook` or `permission` rules for high-risk actions.")
	}
	if hasExamples {
		d2.Score += 5
		d2.Signals = append(d2.Signals, "testable Compliant/Non-compliant rule examples")
	} else {
		recs = append(recs, "Add `Compliant` and `Non-compliant` examples to rules for `gradusctl test`.")
	}

	// 3. Change Governance (0..20)
	d3 := DimensionScore{Name: "change_governance", Max: 20}
	hasCharter := false
	for _, l := range view.Levels {
		if l.Charter != nil && (l.Charter.Purpose != "" || len(l.Charter.AllowedSeverities) > 0) {
			hasCharter = true
			break
		}
	}
	if hasCharter {
		d3.Score += 5
		d3.Signals = append(d3.Signals, "level charter(s) defined")
	} else {
		recs = append(recs, "Define `charter` sections in level.yaml to clarify rule ownership.")
	}
	if cfg.AntiTamper != nil && strings.TrimSpace(cfg.AntiTamper.Mode) != "" {
		d3.Score += 5
		d3.Signals = append(d3.Signals, fmt.Sprintf("anti_tamper mode=%s", cfg.AntiTamper.Mode))
	} else {
		recs = append(recs, "Configure explicit `anti_tamper.mode` (`ask` or `deny`) in gradus.yaml.")
	}
	if cfg.Hooks != nil && strings.TrimSpace(cfg.Hooks.Manager) != "" {
		d3.Score += 5
		d3.Signals = append(d3.Signals, fmt.Sprintf("git hooks manager=%s", cfg.Hooks.Manager))
	} else {
		recs = append(recs, "Configure `hooks.manager` and install git pre-commit checks.")
	}
	if len(cfg.Targets) > 0 {
		d3.Score += 5
		d3.Signals = append(d3.Signals, fmt.Sprintf("%d compiled agent target(s)", len(cfg.Targets)))
	}

	// 4. Obligations & Toolchain (0..20)
	d4 := DimensionScore{Name: "obligations_and_toolchain", Max: 20}
	if len(view.Obligations) > 0 {
		d4.Score += 5
		d4.Signals = append(d4.Signals, fmt.Sprintf("%d obligation(s)", len(view.Obligations)))
	}
	if len(view.Bindings) > 0 {
		d4.Score += 5
		d4.Signals = append(d4.Signals, fmt.Sprintf("%d provider binding(s)", len(view.Bindings)))
	}
	if view.Toolchain != nil && (len(view.Toolchain.Required) > 0 || len(view.Toolchain.Forbidden) > 0) {
		d4.Score += 5
		d4.Signals = append(d4.Signals, "toolchain governance declared")
	}
	if view.Models != nil || len(view.Overlays) > 0 {
		d4.Score += 5
		d4.Signals = append(d4.Signals, "model policy or overlays configured")
	}

	// 5. Evidence & Audit (0..20)
	d5 := DimensionScore{Name: "evidence_and_audit", Max: 20}
	if cfg.Evidence != nil && (len(cfg.Evidence.Sinks) > 0 || (cfg.Evidence.CI != nil && len(cfg.Evidence.CI.Sinks) > 0)) {
		d5.Score += 7
		d5.Signals = append(d5.Signals, "evidence sinks configured")
	} else {
		recs = append(recs, "Configure `evidence.sinks` for local and CI verification records.")
	}
	if evRecords, evErr := evidence.ListLocalRecords(rootDir); evErr == nil && len(evRecords) > 0 {
		d5.Score += 7
		d5.Signals = append(d5.Signals, fmt.Sprintf("%d local evidence record(s)", len(evRecords)))
	} else {
		recs = append(recs, "Run `gradusctl verify --write-evidence` or `gradusctl check --write-evidence`.")
	}
	hasControls := false
	for _, r := range view.Rules {
		if len(r.Controls) > 0 {
			hasControls = true
			break
		}
	}
	if hasControls {
		d5.Score += 6
		d5.Signals = append(d5.Signals, "compliance framework controls mapped")
	} else {
		recs = append(recs, "Map key rules to compliance frameworks via `controls:` for `gradusctl audit export`.")
	}

	dims := []DimensionScore{d1, d2, d3, d4, d5}
	total := 0
	for _, d := range dims {
		total += d.Score
	}

	return Report{
		Schema:          schema.SchemaVersion,
		Repository:      view.Scope.Repository,
		ViewHash:        view.ViewHash,
		MaturityLevel:   classifyMaturity(total),
		TotalScore:      total,
		MaxScore:        100,
		Dimensions:      dims,
		Recommendations: recs,
	}, nil
}

func classifyMaturity(score int) string {
	switch {
	case score >= 85:
		return "L5 Verified"
	case score >= 70:
		return "L4 Governed"
	case score >= 50:
		return "L3 Defined"
	case score >= 30:
		return "L2 Managed"
	default:
		return "L1 Initial"
	}
}
