// Package tests holds cross-package guard tests.
//
// This is the bundle-coverage guard: nexcommon renders every field label, rule
// name and error message by looking a key up in the service's own i18n JSON
// bundles. Nothing verifies a key exists — bundles.ReadMessageBundle recovers
// from the lookup panic and returns the raw key instead. So a missing entry
// ships silently and only surfaces when that exact error path is exercised in
// production. This test is the only thing that catches it.
//
// It deliberately does NOT import the config package: config's init() calls
// log.Fatal on a bad environment, which would kill the test binary at import
// time with no output.
package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/william22913/sandbox-svc-code/constanta"
	svcerror "github.com/william22913/sandbox-svc-code/error"
)

const bundleRoot = "../i18n"

// loadBundle returns every file's key set, keyed by "<bundle>/<lang>".
func loadBundle(t *testing.T, bundle string) map[string]map[string]bool {
	t.Helper()

	dir := filepath.Join(bundleRoot, bundle)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	if len(entries) == 0 {
		t.Fatalf("%s has no bundle files — the service would answer every message with a raw key", dir)
	}

	byLang := make(map[string]map[string]bool)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}

		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}

		// Values are message templates, not keys; only the keys matter here, and
		// some are deliberately not valid Go templates for a caller without params.
		var parsed map[string]interface{}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("parsing %s: %v", e.Name(), err)
		}

		keys := make(map[string]bool, len(parsed))
		for k := range parsed {
			keys[k] = true
		}
		byLang[bundle+"/"+e.Name()] = keys
	}

	return byLang
}

func assertKeysPresent(t *testing.T, bundle string, keys []string) {
	t.Helper()

	for lang, present := range loadBundle(t, bundle) {
		for _, key := range keys {
			if !present[key] {
				t.Errorf("%s: missing key %q — the client would receive the raw key instead of a message", lang, key)
			}
		}
	}
}

func TestEveryFieldLabelKeyIsBundled(t *testing.T) {
	assertKeysPresent(t, "common/constanta", constanta.FieldLabelKeys)
}

func TestEveryRuleNameKeyIsBundled(t *testing.T) {
	assertKeysPresent(t, "common/constanta", constanta.RuleNameKeys)
}

func TestEveryErrorCodeIsBundled(t *testing.T) {
	assertKeysPresent(t, "common/error", svcerror.BundledCodes)
}
