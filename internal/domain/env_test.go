package domain

import (
	"reflect"
	"testing"
)

func TestDiffEnv(t *testing.T) {
	a := []EnvVar{
		{Key: "APP_ENV", Fingerprint: FingerprintEnvValue("production")},
		{Key: "DATABASE_URL", Fingerprint: FingerprintEnvValue("postgres://prod")},
		{Key: "LOG_LEVEL", Fingerprint: FingerprintEnvValue("warn")},
		{Key: "SENTRY_DSN", Fingerprint: FingerprintEnvValue("dsn")},
	}
	b := []EnvVar{
		{Key: "APP_ENV", Fingerprint: FingerprintEnvValue("production")},
		{Key: "DATABASE_URL", Fingerprint: FingerprintEnvValue("postgres://staging")},
		{Key: "LOG_LEVEL", Fingerprint: FingerprintEnvValue("debug")},
		{Key: "FEATURE_X", Fingerprint: FingerprintEnvValue("1")},
	}

	d := DiffEnv(a, b)
	if got, want := d.OnlyInA, []string{"SENTRY_DSN"}; !reflect.DeepEqual(got, want) {
		t.Errorf("OnlyInA = %v, want %v", got, want)
	}
	if got, want := d.OnlyInB, []string{"FEATURE_X"}; !reflect.DeepEqual(got, want) {
		t.Errorf("OnlyInB = %v, want %v", got, want)
	}
	if got, want := d.Different, []string{"DATABASE_URL", "LOG_LEVEL"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Different = %v, want %v", got, want)
	}
	if d.Same != 1 {
		t.Errorf("Same = %d, want 1", d.Same)
	}
	if d.InSync() {
		t.Error("InSync() = true for drifted environments")
	}
}

func TestDiffEnvInSync(t *testing.T) {
	vars := []EnvVar{{Key: "A", Fingerprint: "x"}, {Key: "B", Fingerprint: "y"}}
	d := DiffEnv(vars, vars)
	if !d.InSync() || d.Same != 2 {
		t.Errorf("identical environments: InSync=%v Same=%d", d.InSync(), d.Same)
	}
}

func TestFingerprintEnvValueIsStableAndOpaque(t *testing.T) {
	fp := FingerprintEnvValue("secret-value")
	if fp != FingerprintEnvValue("secret-value") {
		t.Error("fingerprint is not deterministic")
	}
	if fp == FingerprintEnvValue("other") {
		t.Error("distinct values share a fingerprint")
	}
	if len(fp) != 8 {
		t.Errorf("fingerprint length = %d, want 8", len(fp))
	}
}
