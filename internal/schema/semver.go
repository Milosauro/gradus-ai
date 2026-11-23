package schema

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer represents a parsed Semantic Version (major.minor.patch[-prerelease][+build]).
type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

// ParseSemVer parses a SemVer string with an optional leading "v" prefix.
func ParseSemVer(raw string) (SemVer, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return SemVer{}, fmt.Errorf("empty semver string")
	}
	s = strings.TrimPrefix(s, "v")

	if plusIdx := strings.IndexByte(s, '+'); plusIdx != -1 {
		if plusIdx == len(s)-1 {
			return SemVer{}, fmt.Errorf("invalid semver %q: empty build metadata", raw)
		}
		s = s[:plusIdx]
	}

	var pre string
	if dashIdx := strings.IndexByte(s, '-'); dashIdx != -1 {
		pre = s[dashIdx+1:]
		if pre == "" {
			return SemVer{}, fmt.Errorf("invalid semver %q: empty prerelease", raw)
		}
		s = s[:dashIdx]
	}

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return SemVer{}, fmt.Errorf("invalid semver %q: expected major.minor.patch", raw)
	}

	maj, err := parseNonNegInt(parts[0], raw)
	if err != nil {
		return SemVer{}, err
	}
	min, err := parseNonNegInt(parts[1], raw)
	if err != nil {
		return SemVer{}, err
	}
	pat, err := parseNonNegInt(parts[2], raw)
	if err != nil {
		return SemVer{}, err
	}

	return SemVer{
		Major:      maj,
		Minor:      min,
		Patch:      pat,
		Prerelease: pre,
	}, nil
}

func parseNonNegInt(part, raw string) (int, error) {
	if part == "" {
		return 0, fmt.Errorf("invalid semver %q: empty numeric component", raw)
	}
	if len(part) > 1 && part[0] == '0' {
		return 0, fmt.Errorf("invalid semver %q: leading zero in component %q", raw, part)
	}
	for i := 0; i < len(part); i++ {
		if part[i] < '0' || part[i] > '9' {
			return 0, fmt.Errorf("invalid semver %q: non-numeric component %q", raw, part)
		}
	}
	v, err := strconv.Atoi(part)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("invalid semver %q: invalid component %q", raw, part)
	}
	return v, nil
}

// CompareSemVer compares a and b according to SemVer 2.0.0 precedence (-1, 0, +1).
func CompareSemVer(a, b SemVer) int {
	if a.Major != b.Major {
		return cmpInt(a.Major, b.Major)
	}
	if a.Minor != b.Minor {
		return cmpInt(a.Minor, b.Minor)
	}
	if a.Patch != b.Patch {
		return cmpInt(a.Patch, b.Patch)
	}
	if a.Prerelease == b.Prerelease {
		return 0
	}
	if a.Prerelease == "" {
		return 1
	}
	if b.Prerelease == "" {
		return -1
	}
	return comparePrerelease(a.Prerelease, b.Prerelease)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	return 1
}

func comparePrerelease(a, b string) int {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	n := len(pa)
	if len(pb) < n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		if pa[i] == pb[i] {
			continue
		}
		ai, aErr := strconv.Atoi(pa[i])
		bi, bErr := strconv.Atoi(pb[i])
		switch {
		case aErr == nil && bErr == nil:
			return cmpInt(ai, bi)
		case aErr == nil:
			return -1
		case bErr == nil:
			return 1
		case pa[i] < pb[i]:
			return -1
		default:
			return 1
		}
	}
	if len(pa) < len(pb) {
		return -1
	}
	if len(pa) > len(pb) {
		return 1
	}
	return 0
}
