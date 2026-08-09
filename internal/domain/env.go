package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
)

// EnvVar is one environment variable of an application, reduced to what a
// comparison needs. The value itself is never carried: only a fingerprint,
// so two variables can be compared for equality without any secret ever
// reaching the UI, logs or clipboard.
type EnvVar struct {
	Key string
	// Fingerprint is a short, one-way digest of the value. Equal values give
	// equal fingerprints; the value cannot be recovered from it.
	Fingerprint string
	// IsBuildTime marks a variable only available during the build.
	IsBuildTime bool
	// IsPreview marks a variable scoped to preview deployments.
	IsPreview bool
}

// FingerprintEnvValue digests a value for drift comparison. Eight hex
// characters are plenty for telling two configurations apart and useless for
// brute-forcing the value back.
func FingerprintEnvValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:4])
}

// EnvDiff is the outcome of comparing two applications' environments. All
// slices are sorted by key so rendering and tests are deterministic.
type EnvDiff struct {
	// OnlyInA and OnlyInB list keys present on one side only.
	OnlyInA []string
	OnlyInB []string
	// Different lists keys present on both sides with differing values.
	Different []string
	// Same counts keys that match on both sides; the keys themselves are not
	// interesting, the reassurance is.
	Same int
}

// InSync reports that the two environments carry the same keys and values.
func (d EnvDiff) InSync() bool {
	return len(d.OnlyInA) == 0 && len(d.OnlyInB) == 0 && len(d.Different) == 0
}

// DiffEnv compares two environments by key and value fingerprint. Preview
// variables are compared like any other: a drifted preview variable is still
// drift.
func DiffEnv(a, b []EnvVar) EnvDiff {
	byKey := func(vars []EnvVar) map[string]string {
		m := make(map[string]string, len(vars))
		for _, v := range vars {
			m[v.Key] = v.Fingerprint
		}
		return m
	}
	am, bm := byKey(a), byKey(b)

	var d EnvDiff
	for key, fp := range am {
		other, ok := bm[key]
		switch {
		case !ok:
			d.OnlyInA = append(d.OnlyInA, key)
		case other != fp:
			d.Different = append(d.Different, key)
		default:
			d.Same++
		}
	}
	for key := range bm {
		if _, ok := am[key]; !ok {
			d.OnlyInB = append(d.OnlyInB, key)
		}
	}
	slices.Sort(d.OnlyInA)
	slices.Sort(d.OnlyInB)
	slices.Sort(d.Different)
	return d
}
