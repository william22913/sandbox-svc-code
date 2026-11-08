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
	"regexp"
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

// TestEveryNexcommonErrorCodeIsBundled closes the hole the hand-maintained list
// left open: this service ships its own error bundle in place of nexcommon's, so
// every code nexcommon can emit must be present. The list is read out of the
// vendored library rather than written down here, because the previous version
// listed an invented E-3-SBX-AUT-001 while the real 401 rendered nexcommon's
// E-1-CMD-AUT-001 as its own message — a gap the guard could not see.
func TestEveryNexcommonErrorCodeIsBundled(t *testing.T) {
	const messageGo = "../vendor/github.com/nexsoft-git/nexcommon/error/message.go"

	raw, err := os.ReadFile(messageGo)
	if err != nil {
		t.Fatalf("reading %s: %v — the derived code list is the point of this test", messageGo, err)
	}

	codes := nexcommonErrorCodes.FindAllString(string(raw), -1)
	if len(codes) < 20 {
		t.Fatalf("found only %d codes in %s — the extraction regex has drifted from the library", len(codes), messageGo)
	}

	assertKeysPresent(t, "common/error", codes)
}

// nexcommonErrorCodes matches the E-<layer>-<DOMAIN>-<AREA>-<n> shape of every
// sentinel in nexcommon's error package.
var nexcommonErrorCodes = regexp.MustCompile(`E-[0-9]+-[A-Z]+-[A-Z]+-[0-9]+`)
