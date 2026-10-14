package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vexil/internal/ports"
)

func TestCLI_HelpAndVersion(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantExit   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "help subcommand",
			args:       []string{"help"},
			wantExit:   0,
			wantStdout: "Vexil (.vxl)",
		},
		{
			name:       "--help flag",
			args:       []string{"--help"},
			wantExit:   0,
			wantStdout: "Usage: vxl",
		},
		{
			name:       "-h flag",
			args:       []string{"-h"},
			wantExit:   0,
			wantStdout: "Usage: vxl",
		},
		{
			name:       "version command",
			args:       []string{"version"},
			wantExit:   0,
			wantStdout: "vxl v0.1.0",
		},
		{
			name:       "--version flag",
			args:       []string{"--version"},
			wantExit:   0,
			wantStdout: "vxl v0.1.0",
		},
		{
			name:       "no arguments returns error code",
			args:       []string{},
			wantExit:   1,
			wantStderr: "Usage: vxl",
		},
		{
			name:       "unknown subcommand",
			args:       []string{"nonexistent"},
			wantExit:   1,
			wantStderr: "unknown subcommand",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := run(tc.args, strings.NewReader(""), &stdout, &stderr)
			if exitCode != tc.wantExit {
				t.Fatalf("expected exit code %d, got %d (stderr: %q)", tc.wantExit, exitCode, stderr.String())
			}
			if tc.wantStdout != "" && !strings.Contains(stdout.String(), tc.wantStdout) {
				t.Errorf("stdout %q does not contain %q", stdout.String(), tc.wantStdout)
			}
			if tc.wantStderr != "" && !strings.Contains(stderr.String(), tc.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr.String(), tc.wantStderr)
			}
		})
	}
}

func TestCLI_CompileSubcommand(t *testing.T) {
	sampleVexil := "@title: Test Doc\n\n# Heading One\n\nHello *world*.\n"

	t.Run("compile html default from stdin", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"compile"}, strings.NewReader(sampleVexil), &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("run compile failed with code %d: %s", exitCode, stderr.String())
		}
		out := stdout.String()
		if !strings.Contains(out, "<h1>Heading One</h1>") {
			t.Errorf("expected HTML heading in %q", out)
		}
		if !strings.Contains(out, "<p>Hello <strong>world</strong>.</p>") {
			t.Errorf("expected HTML paragraph in %q", out)
		}
	})

	t.Run("compile md from stdin", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"compile", "-target=md"}, strings.NewReader(sampleVexil), &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("run compile -target=md failed with code %d: %s", exitCode, stderr.String())
		}
		out := stdout.String()
		if !strings.Contains(out, "# Heading One") {
			t.Errorf("expected markdown heading in %q", out)
		}
		if !strings.Contains(out, "Hello **world**.") {
			t.Errorf("expected markdown strong in %q", out)
		}
	})

	t.Run("compile json from stdin", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"compile", "-target=json"}, strings.NewReader(sampleVexil), &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("run compile -target=json failed with code %d: %s", exitCode, stderr.String())
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
			t.Fatalf("failed to unmarshal JSON output: %v", err)
		}
		if parsed["type"] != "document" {
			t.Errorf("expected document type in json AST, got %v", parsed["type"])
		}
	})

	t.Run("compile chunks from stdin", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"compile", "-target=chunks"}, strings.NewReader(sampleVexil), &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("run compile -target=chunks failed with code %d: %s", exitCode, stderr.String())
		}
		var chunks []ports.Chunk
		if err := json.Unmarshal(stdout.Bytes(), &chunks); err != nil {
			t.Fatalf("failed to unmarshal chunks JSON output: %v", err)
		}
		if len(chunks) == 0 {
			t.Fatalf("expected at least 1 chunk, got 0")
		}
		if chunks[0].Title != "Heading One" {
			t.Errorf("expected chunk title 'Heading One', got %q", chunks[0].Title)
		}
	})

	t.Run("compile invalid target", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"compile", "-target=bogus"}, strings.NewReader(sampleVexil), &stdout, &stderr)
		if exitCode == 0 {
			t.Fatalf("expected non-zero exit code for invalid target, got 0")
		}
		if !strings.Contains(stderr.String(), "unknown target") {
			t.Errorf("expected 'unknown target' error in %q", stderr.String())
		}
	})

	t.Run("compile from file with -o output file", func(t *testing.T) {
		tmpDir := t.TempDir()
		inFile := filepath.Join(tmpDir, "input.vxl")
		outFile := filepath.Join(tmpDir, "output.html")

		if err := os.WriteFile(inFile, []byte(sampleVexil), 0644); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"compile", "-target=html", "-o", outFile, inFile}, strings.NewReader(""), &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("run compile with -o failed: %s", stderr.String())
		}

		outData, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}
		if !strings.Contains(string(outData), "<h1>Heading One</h1>") {
			t.Errorf("output file does not contain expected HTML: %s", string(outData))
		}
	})

	t.Run("compile non-existent input file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"compile", "missing.vxl"}, strings.NewReader(""), &stdout, &stderr)
		if exitCode == 0 {
			t.Fatalf("expected non-zero exit code for non-existent file")
		}
	})
}

func TestCLI_ASTSubcommand(t *testing.T) {
	sampleVexil := "# AST Test\n\nParagraph text.\n"
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"ast"}, strings.NewReader(sampleVexil), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run ast failed: %s", stderr.String())
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("failed to unmarshal ast output: %v", err)
	}
	if doc["type"] != "document" {
		t.Errorf("expected type 'document', got %v", doc["type"])
	}
}

func TestCLI_ChunkSubcommand(t *testing.T) {
	sampleVexil := "# Chunk Test\n\nChunk content.\n"
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"chunk"}, strings.NewReader(sampleVexil), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run chunk failed: %s", stderr.String())
	}
	var chunks []ports.Chunk
	if err := json.Unmarshal(stdout.Bytes(), &chunks); err != nil {
		t.Fatalf("failed to unmarshal chunk output: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatalf("expected at least 1 chunk")
	}
	if chunks[0].Title != "Chunk Test" {
		t.Errorf("expected title 'Chunk Test', got %q", chunks[0].Title)
	}
}

func TestCLI_FmtSubcommand(t *testing.T) {
	unformatted := "   \n\n#  Heading   \n\n-  item 1\n-  item 2\n\n\n"
	expected := "# Heading\n\n- item 1\n- item 2\n"

	t.Run("fmt from stdin to stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"fmt"}, strings.NewReader(unformatted), &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("fmt failed: %s", stderr.String())
		}
		if stdout.String() != expected {
			t.Errorf("got %q, want %q", stdout.String(), expected)
		}
	})

	t.Run("fmt -w with stdin errors out", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"fmt", "-w"}, strings.NewReader(unformatted), &stdout, &stderr)
		if exitCode == 0 {
			t.Fatalf("expected non-zero exit code for fmt -w with stdin")
		}
		if !strings.Contains(stderr.String(), "cannot use -w with standard input") {
			t.Errorf("expected error message about -w with stdin, got: %s", stderr.String())
		}
	})

	t.Run("fmt -w modifies file in place", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "sample.vxl")
		if err := os.WriteFile(filePath, []byte(unformatted), 0644); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"fmt", "-w", filePath}, strings.NewReader(""), &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("fmt -w failed: %s", stderr.String())
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != expected {
			t.Errorf("got %q, want %q", string(content), expected)
		}
	})

	t.Run("fmt without -w writes to stdout leaving file unchanged", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "sample.vxl")
		if err := os.WriteFile(filePath, []byte(unformatted), 0644); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"fmt", filePath}, strings.NewReader(""), &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("fmt failed: %s", stderr.String())
		}
		if stdout.String() != expected {
			t.Errorf("stdout got %q, want %q", stdout.String(), expected)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != unformatted {
			t.Errorf("file was modified unexpectedly")
		}
	})
}
