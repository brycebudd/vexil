package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"vexil/internal/parser"
	"vexil/pkg/formatter"
	"vexil/pkg/render"
)

const version = "v0.1.0"

const usageText = `Vexil (.vxl) - Deterministic Context-Dense Document Language CLI

Usage: vxl <command> [flags] [file]

Commands:
  compile    Compile .vxl document to target format (html, md, json, chunks)
  fmt        Auto-format .vxl document
  ast        Compile .vxl document to canonical JSON AST
  chunk      Segment .vxl document into semantic RAG chunks (JSON)
  version    Show version information
  help       Show help information

Use "vxl <command> --help" for more information about a command.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func closeResource(c io.Closer) {
	if c != nil {
		_ = c.Close()
	}
}

func printMsg(w io.Writer, text string) {
	_, _ = fmt.Fprint(w, text)
}

func printErrorf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printMsg(stderr, usageText)
		return 1
	}

	cmd := args[0]
	switch cmd {
	case "help", "-help", "--help", "-h":
		printMsg(stdout, usageText)
		return 0

	case "version", "-version", "--version", "-v":
		printErrorf(stdout, "vxl %s\n", version)
		return 0

	case "compile":
		return runCompile(args[1:], stdin, stdout, stderr)

	case "fmt":
		return runFmt(args[1:], stdin, stdout, stderr)

	case "ast":
		return runAST(args[1:], stdin, stdout, stderr)

	case "chunk":
		return runChunk(args[1:], stdin, stdout, stderr)

	default:
		printErrorf(stderr, "unknown subcommand: %q\n\n%s", cmd, usageText)
		return 1
	}
}

func runCompile(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var target string
	var output string
	fs.StringVar(&target, "target", "html", "output target: html, md, json, chunks")
	fs.StringVar(&output, "o", "", "output file path (default stdout)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	inReader, closer, err := resolveInput(fs.Args(), stdin)
	if err != nil {
		printErrorf(stderr, "compile: %v\n", err)
		return 1
	}
	defer closeResource(closer)

	doc, err := parser.NewParser().Parse(inReader)
	if err != nil {
		printErrorf(stderr, "compile error: %v\n", err)
		return 1
	}

	outWriter, outCloser, err := resolveOutput(output, stdout)
	if err != nil {
		printErrorf(stderr, "compile output: %v\n", err)
		return 1
	}
	defer closeResource(outCloser)

	targetNorm := strings.ToLower(strings.TrimSpace(target))
	switch targetNorm {
	case "html":
		if err := render.NewHTMLRenderer().Render(doc, outWriter); err != nil {
			printErrorf(stderr, "render html error: %v\n", err)
			return 1
		}
	case "md", "markdown":
		if err := render.NewMarkdownRenderer().Render(doc, outWriter); err != nil {
			printErrorf(stderr, "render markdown error: %v\n", err)
			return 1
		}
	case "json", "ast":
		if err := render.NewJSONRenderer().Render(doc, outWriter); err != nil {
			printErrorf(stderr, "render json error: %v\n", err)
			return 1
		}
	case "chunks", "chunk", "rag":
		chunks, err := render.NewRAGChunker().Chunk(doc)
		if err != nil {
			printErrorf(stderr, "chunk error: %v\n", err)
			return 1
		}
		data, err := json.MarshalIndent(chunks, "", "  ")
		if err != nil {
			printErrorf(stderr, "marshal chunks error: %v\n", err)
			return 1
		}
		data = append(data, '\n')
		if _, err := outWriter.Write(data); err != nil {
			printErrorf(stderr, "write chunks error: %v\n", err)
			return 1
		}
	default:
		printErrorf(stderr, "unknown target: %q (valid: html, md, json, chunks)\n", target)
		return 1
	}

	return 0
}

func runFmt(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var writeInPlace bool
	var alignTables bool
	fs.BoolVar(&writeInPlace, "w", false, "write result to (source) file instead of stdout")
	fs.BoolVar(&alignTables, "align", false, "align table columns to uniform widths")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	remArgs := fs.Args()
	var filePath string
	if len(remArgs) > 0 && remArgs[0] != "-" {
		filePath = remArgs[0]
	}

	var opts []formatter.Option
	if alignTables {
		opts = append(opts, formatter.WithAlignTableColumns(true))
	}
	fmtr := formatter.NewFormatter(opts...)

	if writeInPlace {
		if filePath == "" {
			printErrorf(stderr, "fmt: cannot use -w with standard input\n")
			return 1
		}
		srcBytes, err := os.ReadFile(filePath)
		if err != nil {
			printErrorf(stderr, "fmt read file: %v\n", err)
			return 1
		}
		var buf bytes.Buffer
		if err := fmtr.Format(bytes.NewReader(srcBytes), &buf); err != nil {
			printErrorf(stderr, "fmt formatting error: %v\n", err)
			return 1
		}
		if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
			printErrorf(stderr, "fmt write file: %v\n", err)
			return 1
		}
		return 0
	}

	inReader, closer, err := resolveInput(remArgs, stdin)
	if err != nil {
		printErrorf(stderr, "fmt: %v\n", err)
		return 1
	}
	defer closeResource(closer)

	if err := fmtr.Format(inReader, stdout); err != nil {
		printErrorf(stderr, "fmt formatting error: %v\n", err)
		return 1
	}

	return 0
}

func runAST(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ast", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var output string
	fs.StringVar(&output, "o", "", "output file path (default stdout)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	inReader, closer, err := resolveInput(fs.Args(), stdin)
	if err != nil {
		printErrorf(stderr, "ast: %v\n", err)
		return 1
	}
	defer closeResource(closer)

	doc, err := parser.NewParser().Parse(inReader)
	if err != nil {
		printErrorf(stderr, "ast parse error: %v\n", err)
		return 1
	}

	outWriter, outCloser, err := resolveOutput(output, stdout)
	if err != nil {
		printErrorf(stderr, "ast output: %v\n", err)
		return 1
	}
	defer closeResource(outCloser)

	if err := render.NewJSONRenderer().Render(doc, outWriter); err != nil {
		printErrorf(stderr, "ast render json error: %v\n", err)
		return 1
	}

	return 0
}

func runChunk(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("chunk", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var output string
	fs.StringVar(&output, "o", "", "output file path (default stdout)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	inReader, closer, err := resolveInput(fs.Args(), stdin)
	if err != nil {
		printErrorf(stderr, "chunk: %v\n", err)
		return 1
	}
	defer closeResource(closer)

	doc, err := parser.NewParser().Parse(inReader)
	if err != nil {
		printErrorf(stderr, "chunk parse error: %v\n", err)
		return 1
	}

	chunks, err := render.NewRAGChunker().Chunk(doc)
	if err != nil {
		printErrorf(stderr, "chunk partitioning error: %v\n", err)
		return 1
	}

	outWriter, outCloser, err := resolveOutput(output, stdout)
	if err != nil {
		printErrorf(stderr, "chunk output: %v\n", err)
		return 1
	}
	defer closeResource(outCloser)

	data, err := json.MarshalIndent(chunks, "", "  ")
	if err != nil {
		printErrorf(stderr, "marshal chunks error: %v\n", err)
		return 1
	}
	data = append(data, '\n')

	if _, err := outWriter.Write(data); err != nil {
		printErrorf(stderr, "write chunks error: %v\n", err)
		return 1
	}

	return 0
}

func resolveInput(args []string, defaultStdin io.Reader) (io.Reader, io.Closer, error) {
	if len(args) == 0 || args[0] == "-" {
		return defaultStdin, nil, nil
	}
	f, err := os.Open(args[0])
	if err != nil {
		return nil, nil, err
	}
	return f, f, nil
}

func resolveOutput(outputPath string, defaultStdout io.Writer) (io.Writer, io.Closer, error) {
	if outputPath == "" || outputPath == "-" {
		return defaultStdout, nil, nil
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return nil, nil, err
	}
	return f, f, nil
}
