package parser

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/PIRSON21/generator-kafka-client/internal/model"
	kstrconv "github.com/PIRSON21/generator-kafka-client/internal/strconv"
)

// ErrFileNotFound is returned when the schema file does not exist.
var ErrFileNotFound = errors.New("schema file not found")

// Parse parses a schema Go file and returns a ServiceDef.
func Parse(filename string) (*model.ServiceDef, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, ErrFileNotFound
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	if file.Name.Name != "schema" {
		return nil, fmt.Errorf("package must be \"schema\", got %q", file.Name.Name)
	}

	var ifaceSpecs []*ast.TypeSpec
	structMap := make(map[string]*ast.StructType)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			switch t := typeSpec.Type.(type) {
			case *ast.InterfaceType:
				ifaceSpecs = append(ifaceSpecs, typeSpec)
			case *ast.StructType:
				structMap[typeSpec.Name.Name] = t
			}
		}
	}

	if len(ifaceSpecs) == 0 {
		return nil, fmt.Errorf("no interface found in schema")
	}

	var services []model.InterfaceDef
	var allMethods []model.MethodDef

	for _, iface := range ifaceSpecs {
		methods, err := parseMethods(iface)
		if err != nil {
			return nil, err
		}
		services = append(services, model.InterfaceDef{
			Name:    iface.Name.Name,
			Methods: methods,
		})
		allMethods = append(allMethods, methods...)
	}

	structs, err := buildStructs(allMethods, structMap)
	if err != nil {
		return nil, err
	}

	return &model.ServiceDef{
		Services: services,
		Structs:  structs,
	}, nil
}

func parseMethods(iface *ast.TypeSpec) ([]model.MethodDef, error) {
	ifaceType := iface.Type.(*ast.InterfaceType)
	var methods []model.MethodDef

	for _, field := range ifaceType.Methods.List {
		if len(field.Names) == 0 {
			continue // embedded interface
		}
		methodName := field.Names[0].Name

		funcType, ok := field.Type.(*ast.FuncType)
		if !ok {
			return nil, fmt.Errorf("method %q: not a function type", methodName)
		}

		params := funcType.Params.List
		if len(params) != 2 {
			return nil, fmt.Errorf("method %q: expected 2 params (context.Context, XxxRequest), got %d", methodName, len(params))
		}
		if !isContextContext(params[0].Type) {
			return nil, fmt.Errorf("method %q: first param must be context.Context", methodName)
		}

		reqType := exprTypeName(params[1].Type)
		if reqType == "" {
			return nil, fmt.Errorf("method %q: cannot determine second param type", methodName)
		}

		results := funcType.Results
		if results == nil || len(results.List) != 1 {
			return nil, fmt.Errorf("method %q: must return exactly one value (error)", methodName)
		}
		retIdent, ok := results.List[0].Type.(*ast.Ident)
		if !ok || retIdent.Name != "error" {
			return nil, fmt.Errorf("method %q: return type must be error", methodName)
		}

		methods = append(methods, model.MethodDef{
			Name:      methodName,
			EventType: kstrconv.ToSnakeCase(methodName),
			ReqType:   reqType,
		})
	}

	return methods, nil
}

func buildStructs(methods []model.MethodDef, structMap map[string]*ast.StructType) ([]model.StructDef, error) {
	seen := make(map[string]bool)
	var structs []model.StructDef

	for _, m := range methods {
		if seen[m.ReqType] {
			continue
		}
		seen[m.ReqType] = true

		st, ok := structMap[m.ReqType]
		if !ok {
			return nil, fmt.Errorf("struct %q not found for method %q", m.ReqType, m.Name)
		}

		def, err := buildStructDef(m.ReqType, st)
		if err != nil {
			return nil, err
		}
		structs = append(structs, def)
	}

	return structs, nil
}

func buildStructDef(name string, st *ast.StructType) (model.StructDef, error) {
	def := model.StructDef{Name: name}

	for _, field := range st.Fields.List {
		fieldType := exprTypeName(field.Type)
		if fieldType == "" {
			return model.StructDef{}, fmt.Errorf("struct %q: cannot determine field type", name)
		}

		for _, nameIdent := range field.Names {
			var tag string
			if field.Tag != nil {
				tag = strings.Trim(field.Tag.Value, "`")
			} else {
				tag = `json:"` + kstrconv.ToSnakeCase(nameIdent.Name) + `"`
			}
			def.Fields = append(def.Fields, model.FieldDef{
				Name: nameIdent.Name,
				Type: fieldType,
				Tag:  tag,
			})
		}
	}

	return def, nil
}

func isContextContext(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return x.Name == "context" && sel.Sel.Name == "Context"
}

func exprTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return ""
}
