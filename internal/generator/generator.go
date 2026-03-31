package generator

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/PIRSON21/generator-kafka-client/internal/model"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var funcMap = template.FuncMap{
	"lowerFirst": func(s string) string {
		if s == "" {
			return ""
		}
		return strings.ToLower(s[:1]) + s[1:]
	},
	"fieldValidations":     fieldValidations,
	"hasValidatableFields": hasValidatableFields,
}

// templateData is the data passed to each template.
type templateData struct {
	*model.ServiceDef
	KafkaImport string
}

// Generate renders types.go, client.go, server.go into outDir.
func Generate(def *model.ServiceDef, outDir string, kafkaImport string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
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

	return nil
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

// hasValidatableFields reports whether any struct field has a validate tag.
func hasValidatableFields(structs []model.StructDef) bool {
	for _, s := range structs {
		for _, f := range s.Fields {
			if f.ValidateTag != "" {
				return true
			}
		}
	}
	return false
}

// fieldValidations returns the generated if-block validation code for a single field.
// The returned string is inserted verbatim into the template; go/format normalises indentation.
func fieldValidations(f model.FieldDef) string {
	if f.ValidateTag == "" {
		return ""
	}
	var sb strings.Builder
	for _, rule := range strings.Split(f.ValidateTag, ",") {
		rule = strings.TrimSpace(rule)
		var block string
		switch {
		case rule == "required":
			block = requiredCheck(f)
		case strings.HasPrefix(rule, "gte="):
			block = gteCheck(f, strings.TrimPrefix(rule, "gte="))
		case strings.HasPrefix(rule, "lte="):
			block = lteCheck(f, strings.TrimPrefix(rule, "lte="))
		case strings.HasPrefix(rule, "neq="):
			block = neqCheck(f, strings.TrimPrefix(rule, "neq="))
		case strings.HasPrefix(rule, "eq="):
			block = eqCheck(f, strings.TrimPrefix(rule, "eq="))
		}
		if block != "" {
			sb.WriteString(block)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func isNumericType(t string) bool {
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return true
	}
	return false
}

func requiredCheck(f model.FieldDef) string {
	switch {
	case f.Type == "string":
		return fmt.Sprintf("if req.%s == \"\" {\nreturn errors.New(\"%s is required\")\n}", f.Name, f.Name)
	case f.Type == "bool":
		return fmt.Sprintf("if !req.%s {\nreturn errors.New(\"%s is required\")\n}", f.Name, f.Name)
	case isNumericType(f.Type):
		return fmt.Sprintf("if req.%s == 0 {\nreturn errors.New(\"%s is required\")\n}", f.Name, f.Name)
	}
	return ""
}

func gteCheck(f model.FieldDef, val string) string {
	if !isNumericType(f.Type) {
		return ""
	}
	return fmt.Sprintf("if req.%s < %s {\nreturn errors.New(\"%s must be >= %s\")\n}", f.Name, val, f.Name, val)
}

func lteCheck(f model.FieldDef, val string) string {
	if !isNumericType(f.Type) {
		return ""
	}
	return fmt.Sprintf("if req.%s > %s {\nreturn errors.New(\"%s must be <= %s\")\n}", f.Name, val, f.Name, val)
}

func neqCheck(f model.FieldDef, val string) string {
	switch {
	case f.Type == "string":
		return fmt.Sprintf("if req.%s == \"%s\" {\nreturn errors.New(\"%s must not equal %s\")\n}", f.Name, val, f.Name, val)
	case isNumericType(f.Type):
		return fmt.Sprintf("if req.%s == %s {\nreturn errors.New(\"%s must not equal %s\")\n}", f.Name, val, f.Name, val)
	}
	return ""
}

func eqCheck(f model.FieldDef, val string) string {
	switch {
	case f.Type == "string":
		return fmt.Sprintf("if req.%s != \"%s\" {\nreturn errors.New(\"%s must equal %s\")\n}", f.Name, val, f.Name, val)
	case isNumericType(f.Type):
		return fmt.Sprintf("if req.%s != %s {\nreturn errors.New(\"%s must equal %s\")\n}", f.Name, val, f.Name, val)
	}
	return ""
}
