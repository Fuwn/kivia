package collect

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Context struct {
	EnclosingFunction string `json:"enclosingFunction,omitempty"`
	Type              string `json:"type,omitempty"`
	ValueExpression   string `json:"valueExpression,omitempty"`
	ParentType        string `json:"parentType,omitempty"`
}

type Identifier struct {
	Name    string  `json:"name"`
	Kind    string  `json:"kind"`
	File    string  `json:"file"`
	Line    int     `json:"line"`
	Column  int     `json:"column"`
	Context Context `json:"context"`
}

func FromPath(path string) ([]Identifier, error) {
	files, err := discoverFiles(path)

	if err != nil {
		return nil, err
	}

	fileSet := token.NewFileSet()
	identifiers := make([]Identifier, 0, 128)

	var parseErrors []error

	for _, filePath := range files {
		fileNode, parseErr := parser.ParseFile(fileSet, filePath, nil, parser.SkipObjectResolution)

		if parseErr != nil {
			parseErrors = append(parseErrors, fmt.Errorf("Failed to parse %s: %w", filePath, parseErr))

			continue
		}

		collector := visitor{
			fileSet: fileSet,
			file:    filePath,
		}

		ast.Walk(&collector, fileNode)

		identifiers = append(identifiers, collector.identifiers...)
	}

	if len(parseErrors) > 0 {
		return nil, parseErrors[0]
	}

	return identifiers, nil
}

type visitor struct {
	fileSet       *token.FileSet
	file          string
	identifiers   []Identifier
	functionStack []string
	typeStack     []string
}

func (identifierVisitor *visitor) Visit(node ast.Node) ast.Visitor {
	switch typedNode := node.(type) {
	case *ast.FuncDecl:
		identifierVisitor.addIdentifier(typedNode.Name, "function", Context{})

		identifierVisitor.functionStack = append(identifierVisitor.functionStack, typedNode.Name.Name)

		identifierVisitor.captureFieldList(typedNode.Recv, "receiver")
		identifierVisitor.captureFieldList(typedNode.Type.Params, "parameter")
		identifierVisitor.captureFieldList(typedNode.Type.Results, "result")

		return leaveScope(identifierVisitor, func() {
			identifierVisitor.functionStack = identifierVisitor.functionStack[:len(identifierVisitor.functionStack)-1]
		})
	case *ast.TypeSpec:
		identifierVisitor.addIdentifier(typedNode.Name, "type", Context{})

		identifierVisitor.typeStack = append(identifierVisitor.typeStack, typedNode.Name.Name)

		identifierVisitor.captureTypeMembers(typedNode.Name.Name, typedNode.Type)

		return leaveScope(identifierVisitor, func() { identifierVisitor.typeStack = identifierVisitor.typeStack[:len(identifierVisitor.typeStack)-1] })
	case *ast.ValueSpec:
		declaredType := renderExpression(identifierVisitor.fileSet, typedNode.Type)
		rightHandValue := renderExpressionList(identifierVisitor.fileSet, typedNode.Values)

		for _, name := range typedNode.Names {
			identifierVisitor.addIdentifier(name, "variable", Context{Type: declaredType, ValueExpression: rightHandValue})
		}
	case *ast.AssignStmt:
		if typedNode.Tok != token.DEFINE {
			break
		}

		rightHandValue := renderExpressionList(identifierVisitor.fileSet, typedNode.Rhs)

		for index, left := range typedNode.Lhs {
			identifierNode, ok := left.(*ast.Ident)

			if !ok {
				continue
			}

			assignmentContext := Context{ValueExpression: rightHandValue}

			if index < len(typedNode.Rhs) {
				assignmentContext.Type = inferTypeFromExpression(typedNode.Rhs[index])
			}

			identifierVisitor.addIdentifier(identifierNode, "variable", assignmentContext)
		}
	case *ast.RangeStmt:
		if typedNode.Tok != token.DEFINE {
			break
		}

		if typedNode.Key != nil {
			if keyIdentifier, ok := typedNode.Key.(*ast.Ident); ok {
				identifierVisitor.addIdentifier(keyIdentifier, "rangeKey", Context{ValueExpression: renderExpression(identifierVisitor.fileSet, typedNode.X)})
			}
		}

		if typedNode.Value != nil {
			if valueIdentifier, ok := typedNode.Value.(*ast.Ident); ok {
				identifierVisitor.addIdentifier(valueIdentifier, "rangeValue", Context{ValueExpression: renderExpression(identifierVisitor.fileSet, typedNode.X)})
			}
		}
	}

	return identifierVisitor
}

type scopeExit struct {
	parent  *visitor
	onLeave func()
}

func leaveScope(parent *visitor, onLeave func()) ast.Visitor {
	return &scopeExit{parent: parent, onLeave: onLeave}
}

func (scopeExitVisitor *scopeExit) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		scopeExitVisitor.onLeave()

		return nil
	}

	return scopeExitVisitor.parent
}

func (identifierVisitor *visitor) captureFieldList(fields *ast.FieldList, kind string) {
	if fields == nil {
		return
	}

	for _, field := range fields.List {
		declaredType := renderExpression(identifierVisitor.fileSet, field.Type)

		for _, name := range field.Names {
			identifierVisitor.addIdentifier(name, kind, Context{Type: declaredType})
		}
	}
}

func (identifierVisitor *visitor) captureTypeMembers(typeName string, typeExpression ast.Expr) {
	switch typedType := typeExpression.(type) {
	case *ast.StructType:
		if typedType.Fields == nil {
			return
		}

		for _, field := range typedType.Fields.List {
			memberType := renderExpression(identifierVisitor.fileSet, field.Type)

			for _, fieldName := range field.Names {
				identifierVisitor.addIdentifier(fieldName, "field", Context{Type: memberType, ParentType: typeName})
			}
		}
	case *ast.InterfaceType:
		if typedType.Methods == nil {
			return
		}

		for _, method := range typedType.Methods.List {
			memberType := renderExpression(identifierVisitor.fileSet, method.Type)

			for _, methodName := range method.Names {
				identifierVisitor.addIdentifier(methodName, "interfaceMethod", Context{Type: memberType, ParentType: typeName})
			}
		}
	}
}

func (identifierVisitor *visitor) addIdentifier(identifier *ast.Ident, kind string, context Context) {
	if identifier == nil || identifier.Name == "_" {
		return
	}

	position := identifierVisitor.fileSet.Position(identifier.NamePos)
	context.EnclosingFunction = currentFunction(identifierVisitor.functionStack)
	identifierVisitor.identifiers = append(identifierVisitor.identifiers, Identifier{
		Name:    identifier.Name,
		Kind:    kind,
		File:    identifierVisitor.file,
		Line:    position.Line,
		Column:  position.Column,
		Context: context,
	})
}

func currentFunction(stack []string) string {
	if len(stack) == 0 {
		return ""
	}

	return stack[len(stack)-1]
}

func discoverFiles(path string) ([]string, error) {
	searchRoot := path
	recursive := false

	if strings.HasSuffix(path, "/...") {
		searchRoot, _ = strings.CutSuffix(path, "/...")
		recursive = true
	}

	if searchRoot == "" {
		searchRoot = "."
	}

	pathFileDetails, err := os.Stat(searchRoot)

	if err != nil {
		return nil, err
	}

	if !pathFileDetails.IsDir() {
		if strings.HasSuffix(searchRoot, ".go") {
			return []string{searchRoot}, nil
		}

		return nil, fmt.Errorf("Path %q is not a Go file.", searchRoot)
	}

	files := make([]string, 0, 64)
	walkErr := filepath.WalkDir(searchRoot, func(candidate string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}

		if entry.IsDir() {
			name := entry.Name()

			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}

			if !recursive && candidate != searchRoot {
				return fs.SkipDir
			}

			return nil
		}

		if strings.HasSuffix(candidate, ".go") {
			files = append(files, candidate)
		}

		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	sort.Strings(files)

	return files, nil
}

func renderExpression(fileSet *token.FileSet, expression ast.Expr) string {
	if expression == nil {
		return ""
	}

	var buffer bytes.Buffer

	if err := printer.Fprint(&buffer, fileSet, expression); err != nil {
		return ""
	}

	return buffer.String()
}

func renderExpressionList(fileSet *token.FileSet, expressions []ast.Expr) string {
	if len(expressions) == 0 {
		return ""
	}

	parts := make([]string, 0, len(expressions))

	for _, expression := range expressions {
		parts = append(parts, renderExpression(fileSet, expression))
	}

	return strings.Join(parts, ", ")
}

func inferTypeFromExpression(expression ast.Expr) string {
	switch typedExpression := expression.(type) {
	case *ast.CallExpr:
		switch functionExpression := typedExpression.Fun.(type) {
		case *ast.Ident:
			return functionExpression.Name
		case *ast.SelectorExpr:
			return functionExpression.Sel.Name
		}

		return ""
	default:
		return ""
	}
}
