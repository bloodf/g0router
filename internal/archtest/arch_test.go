package archtest

import (
	"encoding/json"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureConformance(t *testing.T) {
	modRoot := moduleRoot(t)

	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = modRoot
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("go list failed: %v\n%s", err, ee.Stderr)
		}
		t.Fatalf("go list failed: %v", err)
	}

	type pkgInfo struct {
		ImportPath  string   `json:"ImportPath"`
		Imports     []string `json:"Imports"`
		TestImports []string `json:"TestImports"`
	}

	const modulePrefix = "github.com/bloodf/g0router"

	var violations []string

	dec := json.NewDecoder(strings.NewReader(string(out)))
	for {
		var p pkgInfo
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("decode go list output: %v", err)
		}

		// Only check packages under our module.
		if !strings.HasPrefix(p.ImportPath, modulePrefix) {
			continue
		}

		// Rule 1: No internal/<domain> package imports api/.
		// Allowlist: internal/cli is a transport/entry-point package, not domain logic.
		if strings.HasPrefix(p.ImportPath, modulePrefix+"/internal/") && p.ImportPath != modulePrefix+"/internal/cli" {
			for _, imp := range p.Imports {
				if strings.HasPrefix(imp, modulePrefix+"/api") {
					violations = append(violations, p.ImportPath+" imports "+imp+": violated domain→transport rule")
				}
			}
			for _, imp := range p.TestImports {
				if strings.HasPrefix(imp, modulePrefix+"/api") {
					violations = append(violations, p.ImportPath+" [test] imports "+imp+": violated domain→transport rule")
				}
			}
		}

		// Rule 2: No domain package imports fasthttp except transport-adjacent packages.
		// Allowlist: api/* and internal/providers/* (HTTP client adapters, not domain logic).
		isProvider := strings.HasPrefix(p.ImportPath, modulePrefix+"/internal/providers/")
		isAPI := strings.HasPrefix(p.ImportPath, modulePrefix+"/api")
		if !isAPI && !isProvider {
			for _, imp := range p.Imports {
				if imp == "github.com/valyala/fasthttp" {
					violations = append(violations, p.ImportPath+" imports fasthttp: violated domain→transport rule")
				}
			}
			for _, imp := range p.TestImports {
				if imp == "github.com/valyala/fasthttp" {
					violations = append(violations, p.ImportPath+" [test] imports fasthttp: violated domain→transport rule")
				}
			}
		}

		// Rule 3: internal/store imports no domain package.
		// Allowlist: internal/mcp and internal/usage are known pre-existing couplings
		// that are out of scope for Phase 12B (store stays one package per explicit non-goals).
		if p.ImportPath == modulePrefix+"/internal/store" || strings.HasPrefix(p.ImportPath, modulePrefix+"/internal/store/") {
			for _, imp := range p.Imports {
				if strings.HasPrefix(imp, modulePrefix+"/internal/") && imp != modulePrefix+"/internal/store" && !strings.HasPrefix(imp, modulePrefix+"/internal/store/") && imp != modulePrefix+"/internal/mcp" && imp != modulePrefix+"/internal/usage" {
					violations = append(violations, p.ImportPath+" imports "+imp+": violated repository→domain rule")
				}
			}
			for _, imp := range p.TestImports {
				if strings.HasPrefix(imp, modulePrefix+"/internal/") && imp != modulePrefix+"/internal/store" && !strings.HasPrefix(imp, modulePrefix+"/internal/store/") && imp != modulePrefix+"/internal/mcp" && imp != modulePrefix+"/internal/usage" {
					violations = append(violations, p.ImportPath+" [test] imports "+imp+": violated repository→domain rule")
				}
			}
		}
	}

	for _, v := range violations {
		t.Error(v)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("go", "env", "GOMOD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env GOMOD failed: %v", err)
	}
	modFile := strings.TrimSpace(string(out))
	if modFile == "" {
		t.Fatal("not inside a Go module")
	}
	return filepath.Dir(modFile)
}
