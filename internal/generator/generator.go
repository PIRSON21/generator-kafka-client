package generator

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/PIRSON21/generator-kafka-client/internal/model"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

//go:embed kafkasrc/*.go.src
var kafkaSrcFS embed.FS

var funcMap = template.FuncMap{
	"lowerFirst": func(s string) string {
		if s == "" {
			return ""
		}
		return strings.ToLower(s[:1]) + s[1:]
	},
}

// templateData is the data passed to each template.
type templateData struct {
	*model.ServiceDef
	KafkaImport string
}

// Generate renders types.go, client.go, server.go into outDir and copies the
// kafka wrapper sources into outDir/kafka/.
func Generate(def *model.ServiceDef, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	kafkaImport, err := resolveKafkaImport(outDir)
	if err != nil {
		return fmt.Errorf("resolve kafka import: %w", err)
	}

	data := templateData{ServiceDef: def, KafkaImport: kafkaImport}

	targets := []struct {
		tmpl string
		out  string
	}{
		{"templates/types.go.tmpl", "types.go"},
		{"templates/client.go.tmpl", "client.go"},
		{"templates/server.go.tmpl", "server.go"},
	}

	for _, t := range targets {
		if err := renderFile(data, t.tmpl, filepath.Join(outDir, t.out)); err != nil {
			return fmt.Errorf("generate %s: %w", t.out, err)
		}
	}

	if err := copyKafkaSrc(filepath.Join(outDir, "kafka")); err != nil {
		return fmt.Errorf("copy kafka sources: %w", err)
	}

	return nil
}

// resolveKafkaImport finds the go.mod above outDir, derives the module import
// path and returns "<module>/<rel-outDir>/kafka".
func resolveKafkaImport(outDir string) (string, error) {
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return "", fmt.Errorf("abs path: %w", err)
	}

	modRoot, modName, err := findModule(absOut)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(modRoot, absOut)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}

	// Use forward slashes for Go import paths.
	rel = filepath.ToSlash(rel)
	return modName + "/" + rel + "/kafka", nil
}

// findModule walks up from dir until it finds a go.mod file and returns the
// module root directory and module name declared in that file.
func findModule(dir string) (root, name string, err error) {
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, statErr := os.Stat(candidate); statErr == nil {
			name, err := readModuleName(candidate)
			if err != nil {
				return "", "", err
			}
			return dir, name, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}

// readModuleName parses the module directive from a go.mod file.
func readModuleName(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open go.mod: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in %s", path)
}

// copyKafkaSrc copies the embedded kafka source files into destDir.
func copyKafkaSrc(destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create kafka dir: %w", err)
	}

	return fs.WalkDir(kafkaSrcFS, "kafkasrc", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := kafkaSrcFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}

		// Strip ".src" suffix: "producer.go.src" → "producer.go"
		name := strings.TrimSuffix(filepath.Base(path), ".src")
		destPath := filepath.Join(destDir, name)
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", destPath, err)
		}

		return nil
	})
}

func renderFile(data templateData, tmplPath, outPath string) error {
	src, err := templateFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Funcs(funcMap).Parse(string(src))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format source (%s): %w\n--- raw ---\n%s", outPath, err, buf.String())
	}

	if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
