package spec_v01_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"vexil/internal/parser"
	"vexil/pkg/render"
)

var update = flag.Bool("update", false, "update golden files")

func TestGoldenFiles(t *testing.T) {
	vxlPath := filepath.Join(".", "spec.vxl")
	astPath := filepath.Join(".", "spec.ast.json")
	htmlPath := filepath.Join(".", "spec.html")

	vxlBytes, err := os.ReadFile(vxlPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", vxlPath, err)
	}

	doc, err := parser.NewParser().Parse(bytes.NewReader(vxlBytes))
	if err != nil {
		t.Fatalf("failed to parse spec.vxl: %v", err)
	}

	// Render AST to JSON
	var jsonBuf bytes.Buffer
	if err := render.NewJSONRenderer().Render(doc, &jsonBuf); err != nil {
		t.Fatalf("failed to render JSON: %v", err)
	}
	actualJSON := jsonBuf.Bytes()

	// Render AST to HTML
	var htmlBuf bytes.Buffer
	if err := render.NewHTMLRenderer().Render(doc, &htmlBuf); err != nil {
		t.Fatalf("failed to render HTML: %v", err)
	}
	actualHTML := htmlBuf.Bytes()

	if *update {
		if err := os.WriteFile(astPath, actualJSON, 0644); err != nil {
			t.Fatalf("failed to update %s: %v", astPath, err)
		}
		if err := os.WriteFile(htmlPath, actualHTML, 0644); err != nil {
			t.Fatalf("failed to update %s: %v", htmlPath, err)
		}
		t.Logf("golden files successfully updated")
		return
	}

	// Verify JSON AST golden file
	expectedJSON, err := os.ReadFile(astPath)
	if err != nil {
		t.Fatalf("failed to read golden AST %s: %v", astPath, err)
	}
	if !bytes.Equal(actualJSON, expectedJSON) {
		t.Errorf("AST JSON does not match golden file %s.\nActual:\n%s\nExpected:\n%s",
			astPath, string(actualJSON), string(expectedJSON))
	}

	// Verify HTML golden file
	expectedHTML, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("failed to read golden HTML %s: %v", htmlPath, err)
	}
	if !bytes.Equal(actualHTML, expectedHTML) {
		t.Errorf("HTML output does not match golden file %s.\nActual:\n%s\nExpected:\n%s",
			htmlPath, string(actualHTML), string(expectedHTML))
	}
}
