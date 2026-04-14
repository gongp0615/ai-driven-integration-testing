package codeanalyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ModuleInfo struct {
	Name       string
	File       string
	Structs    []StructInfo
	Functions  []FuncInfo
	Publishes  []EventRef
	Subscribes []EventRef
	Constants  []ConstInfo
	Bugs       []BugHint
}

type StructInfo struct {
	Name   string
	Fields []string
}

type FuncInfo struct {
	Receiver string
	Name     string
	Params   string
}

type EventRef struct {
	EventType string
	Function  string
	Line      int
}

type ConstInfo struct {
	Name  string
	Value string
}

type BugHint struct {
	Description string
	Location    string
	Line        int
}

func Analyze(projectDir string) ([]ModuleInfo, error) {
	internalDir := filepath.Join(projectDir, "internal")
	entries, err := os.ReadDir(internalDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read internal dir: %w", err)
	}

	var modules []ModuleInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		goFile := filepath.Join(internalDir, e.Name(), e.Name()+".go")
		if _, err := os.Stat(goFile); os.IsNotExist(err) {
			continue
		}
		mod, err := analyzeFile(goFile, projectDir)
		if err != nil {
			continue
		}
		modules = append(modules, mod)
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})
	return modules, nil
}

func analyzeFile(filePath, projectDir string) (ModuleInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return ModuleInfo{}, err
	}

	relPath, _ := filepath.Rel(projectDir, filePath)
	mod := ModuleInfo{
		Name: strings.TrimSuffix(filepath.Base(filePath), ".go"),
		File: relPath,
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if st, ok := x.Type.(*ast.StructType); ok {
				si := StructInfo{Name: x.Name.Name}
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						si.Fields = append(si.Fields, name.Name)
					}
				}
				mod.Structs = append(mod.Structs, si)
			}

		case *ast.FuncDecl:
			fi := FuncInfo{Name: x.Name.Name}
			if x.Recv != nil && len(x.Recv.List) > 0 {
				if star, ok := x.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := star.X.(*ast.Ident); ok {
						fi.Receiver = ident.Name
					}
				} else if ident, ok := x.Recv.List[0].Type.(*ast.Ident); ok {
					fi.Receiver = ident.Name
				}
			}
			var paramStrs []string
			if x.Type.Params != nil {
				for _, param := range x.Type.Params.List {
					for _, name := range param.Names {
						paramStrs = append(paramStrs, name.Name)
					}
				}
			}
			fi.Params = strings.Join(paramStrs, ", ")
			mod.Functions = append(mod.Functions, fi)

		case *ast.CallExpr:
			line := fset.Position(n.Pos()).Line
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if sel.Sel.Name == "Publish" {
						if len(x.Args) > 0 {
							if eventType := extractStringLiteral(x.Args[0]); eventType != "" {
								mod.Publishes = append(mod.Publishes, EventRef{
									EventType: eventType,
									Function:  ident.Name,
									Line:      line,
								})
							}
						}
					}
					if sel.Sel.Name == "Subscribe" {
						if len(x.Args) > 0 {
							if eventType := extractStringLiteral(x.Args[0]); eventType != "" {
								handlerName := "?"
								if len(x.Args) > 1 {
									handlerName = extractIdentName(x.Args[1])
								}
								mod.Subscribes = append(mod.Subscribes, EventRef{
									EventType: eventType,
									Function:  fmt.Sprintf("%s.%s", ident.Name, handlerName),
									Line:      line,
								})
							}
						}
					}
				}
			}
		}
		return true
	})

	return mod, nil
}

func extractStringLiteral(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return strings.Trim(lit.Value, `"`)
	}
	return ""
}

func extractIdentName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}
	return "?"
}

func FormatSummary(modules []ModuleInfo) string {
	var sb strings.Builder

	sb.WriteString("## Project Structure\n")
	for _, m := range modules {
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", m.Name, m.File))
	}

	sb.WriteString("\n## Module Details\n")
	for _, m := range modules {
		sb.WriteString(fmt.Sprintf("\n### %s (%s)\n", m.Name, m.File))

		if len(m.Structs) > 0 {
			sb.WriteString("**Structs:**\n")
			for _, s := range m.Structs {
				sb.WriteString(fmt.Sprintf("- %s { %s }\n", s.Name, strings.Join(s.Fields, ", ")))
			}
		}

		if len(m.Functions) > 0 {
			sb.WriteString("**Functions:**\n")
			for _, f := range m.Functions {
				if f.Receiver != "" {
					sb.WriteString(fmt.Sprintf("- (%s) %s(%s)\n", f.Receiver, f.Name, f.Params))
				} else {
					sb.WriteString(fmt.Sprintf("- %s(%s)\n", f.Name, f.Params))
				}
			}
		}

		if len(m.Publishes) > 0 {
			sb.WriteString("**Publishes events:**\n")
			for _, p := range m.Publishes {
				sb.WriteString(fmt.Sprintf("- \"%s\" (in %s, line %d)\n", p.EventType, p.Function, p.Line))
			}
		}

		if len(m.Subscribes) > 0 {
			sb.WriteString("**Subscribes to events:**\n")
			for _, s := range m.Subscribes {
				sb.WriteString(fmt.Sprintf("- \"%s\" → %s (line %d)\n", s.EventType, s.Function, s.Line))
			}
		}
	}

	sb.WriteString("\n## Event Flow Map (auto-generated from Publish/Subscribe)\n")
	pubMap := make(map[string][]string)
	for _, m := range modules {
		for _, p := range m.Publishes {
			pubMap[p.EventType] = append(pubMap[p.EventType], m.Name+".Publish")
		}
	}
	for _, m := range modules {
		for _, s := range m.Subscribes {
			if pubs, ok := pubMap[s.EventType]; ok {
				for _, pub := range pubs {
					publisher := strings.Split(pub, ".")[0]
					if publisher != m.Name {
						sb.WriteString(fmt.Sprintf("- %s ──\"%s\"──→ %s.%s\n", publisher, s.EventType, m.Name, s.Function))
					}
				}
			}
		}
	}

	return sb.String()
}
