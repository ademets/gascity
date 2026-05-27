package dolt_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const statusScript = "commands/status/run.sh"

func writeFakeProbe(t *testing.T, cityPath, body string) {
	t.Helper()
	script := filepath.Join(cityPath, ".gc", "system", "packs", "bd", "assets", "scripts", "gc-beads-bd.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDoltStatusPrintsProbeFailure(t *testing.T) {
	root := repoRoot(t)
	cityPath := t.TempDir()
	writeFakeProbe(t, cityPath, "#!/bin/sh\necho fake probe detail >&2\nexit 2\n")

	cmd := exec.Command("sh", filepath.Join(root, statusScript))
	cmd.Env = append(filteredEnv("GC_DOLT_PORT"),
		"GC_CITY_PATH="+cityPath,
		"GC_PACK_DIR="+root,
		"GC_DOLT_PORT=1",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("status unexpectedly succeeded:\n%s", out)
	}
	for _, want := range []string{"Server: not running", "Probe: failed (exit 2)", "fake probe detail"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("status output missing %q:\n%s", want, out)
		}
	}
}

func TestDoltStatusTimesOutWithDiagnosticOutput(t *testing.T) {
	if _, errT := exec.LookPath("timeout"); errT != nil {
		if _, errG := exec.LookPath("gtimeout"); errG != nil {
			if _, errP := exec.LookPath("python3"); errP != nil {
				t.Skip("timeout, gtimeout, and python3 unavailable; cannot exercise bounded probe")
			}
		}
	}

	root := repoRoot(t)
	cityPath := t.TempDir()
	writeFakeProbe(t, cityPath, "#!/bin/sh\nsleep 5\n")

	cmd := exec.Command("sh", filepath.Join(root, statusScript))
	cmd.Env = append(filteredEnv("GC_DOLT_PORT"),
		"GC_CITY_PATH="+cityPath,
		"GC_PACK_DIR="+root,
		"GC_DOLT_PORT=1",
		"GC_DOLT_STATUS_TIMEOUT=1",
	)
	start := time.Now()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("status unexpectedly succeeded:\n%s", out)
	}
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Fatalf("status took %s, want bounded output; out:\n%s", elapsed, out)
	}
	for _, want := range []string{"Server: not running", "Probe: timed out after 1s"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("status output missing %q:\n%s", want, out)
		}
	}
}
