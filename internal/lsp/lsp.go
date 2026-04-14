package lsp

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type LSPServer struct {
	mu           sync.RWMutex
	workspace    string
	diagnostics  map[string][]Diagnostic
	symbols      map[string][]DocumentSymbol
	completions  map[string][]Completion
	activeFile   string
	analysisMode AnalysisMode
}

type Diagnostic struct {
	Range    Range
	Severity DiagnosticSeverity
	Code     string
	Source   string
	Message  string
	Tags     []DiagnosticTag
}

type Range struct {
	Start Position
	End   Position
}

type Position struct {
	Line      int
	Character int
}

type DiagnosticSeverity int

const (
	SeverityError   DiagnosticSeverity = 1
	SeverityWarning DiagnosticSeverity = 2
	SeverityInfo    DiagnosticSeverity = 3
	SeverityHint    DiagnosticSeverity = 4
)

type DiagnosticTag int

const (
	TagUnnecessary DiagnosticTag = 1
	TagDeprecated  DiagnosticTag = 2
)

type DocumentSymbol struct {
	Name           string
	Kind           SymbolKind
	Range          Range
	SelectionRange Range
	Children       []DocumentSymbol
	Detail         string
}

type SymbolKind int

const (
	KindFile       SymbolKind = 1
	KindModule     SymbolKind = 2
	KindNamespace  SymbolKind = 3
	KindPackage    SymbolKind = 4
	KindClass      SymbolKind = 5
	KindMethod     SymbolKind = 6
	KindProperty   SymbolKind = 7
	KindField      SymbolKind = 8
	KindConstructor SymbolKind = 9
	KindEnum       SymbolKind = 10
	KindInterface  SymbolKind = 11
	KindFunction   SymbolKind = 12
	KindVariable   SymbolKind = 13
	KindConstant   SymbolKind = 14
	KindString     SymbolKind = 15
	KindNumber     SymbolKind = 16
	KindBoolean    SymbolKind = 17
	KindArray      SymbolKind = 18
	KindObject     SymbolKind = 19
	KindKey        SymbolKind = 20
	KindNull       SymbolKind = 21
	KindEnumMember SymbolKind = 22
	KindStruct     SymbolKind = 23
	KindEvent      SymbolKind = 24
	KindOperator   SymbolKind = 25
	KindTypeParameter SymbolKind = 26
)

type Completion struct {
	Label         string
	Kind          CompletionKind
	Detail        string
	Documentation string
	InsertText    string
}

type CompletionKind int

const (
	CompletionText     CompletionKind = 1
	CompletionMethod  CompletionKind = 2
	CompletionFunction CompletionKind = 3
	CompletionConstructor CompletionKind = 4
	CompletionField   CompletionKind = 5
	CompletionVariable CompletionKind = 6
	CompletionClass   CompletionKind = 7
	CompletionInterface CompletionKind = 8
	CompletionModule  CompletionKind = 9
	CompletionProperty CompletionKind = 10
	CompletionUnit    CompletionKind = 11
	CompletionValue   CompletionKind = 12
	CompletionEnum    CompletionKind = 13
	CompletionKeyword CompletionKind = 14
	CompletionSnippet CompletionKind = 15
	CompletionColor   CompletionKind = 16
	CompletionFile    CompletionKind = 17
	CompletionReference CompletionKind = 18
	CompletionFolder  CompletionKind = 19
	CompletionEnumMember CompletionKind = 20
	CompletionConstant CompletionKind = 21
	CompletionStruct  CompletionKind = 22
	CompletionEvent   CompletionKind = 23
	CompletionOperator CompletionKind = 24
	CompletionTypeParameter CompletionKind = 25
)

type AnalysisMode string

const (
	ModeFast    AnalysisMode = "fast"
	ModeDeep    AnalysisMode = "deep"
	ModeExpert  AnalysisMode = "expert"
)

var goKeywords = []string{
	"func", "struct", "interface", "type", "var", "const", "package", "import",
	"if", "else", "for", "range", "switch", "case", "default", "select",
	"return", "break", "continue", "goto", "fallthrough",
	"defer", "go", "chan", "map", "make", "new", "append", "len", "cap",
	"copy", "delete", "close", "panic", "recover", "print", "println",
}

var goSnippets = map[string]Completion{
	"func": {
		Label:      "func",
		Kind:       CompletionKeyword,
		Detail:     "func name() {}",
		InsertText: "func ${1:name}(${2:params}) ${3:error} {\n\t$0\n}",
	},
	"struct": {
		Label:      "struct",
		Kind:       CompletionKeyword,
		Detail:     "type Name struct {}",
		InsertText: "type ${1:Name} struct {\n\t$0\n}",
	},
	"interface": {
		Label:      "interface",
		Kind:       CompletionKeyword,
		Detail:     "type Name interface {}",
		InsertText: "type ${1:Name} interface {\n\t$0\n}",
	},
	"err": {
		Label:      "if err != nil",
		Kind:       CompletionSnippet,
		Detail:     "Error handling",
		InsertText: "if err != nil {\n\treturn fmt.Errorf(\"${1:error}: %w\", err)\n}",
	},
}

func NewLSPServer(workspace string) *LSPServer {
	return &LSPServer{
		workspace:    workspace,
		diagnostics: make(map[string][]Diagnostic),
		symbols:     make(map[string][]DocumentSymbol),
		completions: make(map[string][]Completion),
		analysisMode: ModeFast,
	}
}

func (lsp *LSPServer) AnalyzeFile(ctx context.Context, filePath string) error {
	lsp.mu.Lock()
	defer lsp.mu.Unlock()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors)
	if err != nil {
		return err
	}

	var diags []Diagnostic

	ast.Inspect(node, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.FuncDecl:
			if expr.Body == nil {
				diags = append(diags, Diagnostic{
					Range:    lsp.nodeToRange(fset, expr),
					Severity: SeverityWarning,
					Code:     "no-body",
					Source:   "siby-lsp",
					Message:  fmt.Sprintf("Fonction '%s' sans corps", expr.Name.Name),
				})
			}
		case *ast.Ident:
			if expr.Name == "_" && lsp.analysisMode == ModeExpert {
				diags = append(diags, Diagnostic{
					Range:    lsp.nodeToRange(fset, expr),
					Severity: SeverityInfo,
					Code:     "blank-identifier",
					Source:   "siby-lsp",
					Message:  "Utilisation du blank identifier (_)",
				})
			}
		}
		return true
	})

	diags = append(diags, lsp.checkCommonErrors(node, fset)...)
	diags = append(diags, lsp.checkStyleIssues(node, fset)...)

	lsp.diagnostics[filePath] = diags
	lsp.symbols[filePath] = lsp.extractSymbols(node, fset)

	return nil
}

func (lsp *LSPServer) checkCommonErrors(node *ast.File, fset *token.FileSet) []Diagnostic {
	var diags []Diagnostic

	for _, imp := range node.Imports {
		if imp.Path.Value == `"fmt"` {
			hasFmtUsage := false
			ast.Inspect(node, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "fmt" {
							hasFmtUsage = true
						}
					}
				}
				return true
			})
			if !hasFmtUsage {
				diags = append(diags, Diagnostic{
					Range:    lsp.nodeToRange(fset, imp),
					Severity: SeverityInfo,
					Code:     "unused-import",
					Source:   "siby-lsp",
					Message:  fmt.Sprintf("Import '%s' non utilisé", imp.Path.Value),
					Tags:     []DiagnosticTag{TagUnnecessary},
				})
			}
		}
	}

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "main" && node.Name.Name != "main" && node.Name.Name != "" {
				diags = append(diags, Diagnostic{
					Range:    lsp.nodeToRange(fset, fn),
					Severity: SeverityWarning,
					Code:     "non-main-main",
					Source:   "siby-lsp",
					Message:  "Fonction 'main' dans un package non-main",
				})
			}
		}
	}

	return diags
}

func (lsp *LSPServer) checkStyleIssues(node *ast.File, fset *token.FileSet) []Diagnostic {
	var diags []Diagnostic

	if lsp.analysisMode != ModeExpert {
		return diags
	}

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.IsExported() {
				hasDoc := false
				for _, cm := range node.Comments {
					if cm.End() < fn.Pos() && fn.Pos()-cm.End() < 10 {
						hasDoc = true
						break
					}
				}
				if !hasDoc {
					diags = append(diags, Diagnostic{
						Range:    lsp.nodeToRange(fset, fn),
						Severity: SeverityHint,
						Code:     "no-doc",
						Source:   "siby-lsp",
						Message:  fmt.Sprintf("Fonction exportée '%s' sans documentation", fn.Name.Name),
					})
				}
			}
		}
	}

	return diags
}

func (lsp *LSPServer) extractSymbols(node *ast.File, fset *token.FileSet) []DocumentSymbol {
	var symbols []DocumentSymbol

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			symbols = append(symbols, DocumentSymbol{
				Name:  d.Name.Name,
				Kind:  KindFunction,
				Range: lsp.nodeToRange(fset, d),
				SelectionRange: Range{
					Start: Position{Line: fset.Position(d.Name.Pos()).Line - 1},
					End:   Position{Line: fset.Position(d.Name.End()).Line - 1},
				},
			})
		case *ast.TypeSpec:
			symbols = append(symbols, DocumentSymbol{
				Name:  d.Name.Name,
				Kind:  KindClass,
				Range: lsp.nodeToRange(fset, d),
				SelectionRange: Range{
					Start: Position{Line: fset.Position(d.Name.Pos()).Line - 1},
					End:   Position{Line: fset.Position(d.Name.End()).Line - 1},
				},
			})
		}
	}

	return symbols
}

func (lsp *LSPServer) nodeToRange(fset *token.FileSet, node ast.Node) Range {
	pos := fset.Position(node.Pos())
	end := fset.Position(node.End())
	return Range{
		Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
		End:   Position{Line: end.Line - 1, Character: end.Column - 1},
	}
}

func (lsp *LSPServer) GetDiagnostics(filePath string) []Diagnostic {
	lsp.mu.RLock()
	defer lsp.mu.RUnlock()
	return lsp.diagnostics[filePath]
}

func (lsp *LSPServer) GetSymbols(filePath string) []DocumentSymbol {
	lsp.mu.RLock()
	defer lsp.mu.RUnlock()
	return lsp.symbols[filePath]
}

func (lsp *LSPServer) GetCompletions(filePath string, prefix string) []Completion {
	var completions []Completion

	for _, kw := range goKeywords {
		if strings.HasPrefix(kw, prefix) {
			completions = append(completions, Completion{
				Label: kw,
				Kind:  CompletionKeyword,
			})
		}
	}

	for name, snippet := range goSnippets {
		if strings.HasPrefix(name, prefix) {
			completions = append(completions, snippet)
		}
	}

	return completions
}

func (lsp *LSPServer) AnalyzeProject(ctx context.Context) error {
	return filepath.Walk(lsp.workspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".go" {
			lsp.AnalyzeFile(ctx, path)
		}
		return nil
	})
}

func (lsp *LSPServer) SetAnalysisMode(mode AnalysisMode) {
	lsp.mu.Lock()
	defer lsp.mu.Unlock()
	lsp.analysisMode = mode
}

func (lsp *LSPServer) GetErrorCount(filePath string) (errors, warnings, infos int) {
	diags := lsp.GetDiagnostics(filePath)
	for _, d := range diags {
		switch d.Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warnings++
		case SeverityInfo, SeverityHint:
			infos++
		}
	}
	return
}

func (lsp *LSPServer) RenderDiagnostics(filePath string) string {
	diags := lsp.GetDiagnostics(filePath)
	if len(diags) == 0 {
		return ""
	}

	var sb strings.Builder

	errors, warnings, infos := lsp.GetErrorCount(filePath)

	sb.WriteString(fmt.Sprintf("\n📊 Analyse LSP: %d erreurs, %d avertissements, %d infos\n\n",
		errors, warnings, infos))

	for _, d := range diags {
		icon := "❌"
		color := "\033[91m"
		switch d.Severity {
		case SeverityWarning:
			icon = "⚠️"
			color = "\033[93m"
		case SeverityInfo:
			icon = "ℹ️"
			color = "\033[94m"
		case SeverityHint:
			icon = "💡"
			color = "\033[92m"
		}

		sb.WriteString(fmt.Sprintf("%s%s [%s]%s Ligne %d: %s\n",
			color, icon, d.Code, "\033[0m", d.Range.Start.Line+1, d.Message))
	}

	return sb.String()
}

type HoverInfo struct {
	Range       Range
	Contents    []string
	SyntaxKind  string
}

func (lsp *LSPServer) GetHover(filePath string, pos Position) *HoverInfo {
	lsp.mu.RLock()
	defer lsp.mu.RUnlock()

	symbols := lsp.symbols[filePath]
	for _, sym := range symbols {
		if lsp.isPositionInRange(pos, sym.Range) {
			return &HoverInfo{
				Range:      sym.Range,
				Contents:    []string{fmt.Sprintf("**%s** (%s)", sym.Name, sym.Kind)},
				SyntaxKind: "markdown",
			}
		}
	}

	return nil
}

func (lsp *LSPServer) isPositionInRange(pos Position, r Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character > r.End.Character {
		return false
	}
	return true
}

func (lsp *LSPServer) FindDefinition(filePath string, pos Position) *Range {
	return nil
}

func (lsp *LSPServer) FindReferences(filePath string, pos Position) []Range {
	return nil
}

type CodeAction struct {
	Title       string
	Kind        string
	Edit        *WorkspaceEdit
	IsPreferred bool
}

type WorkspaceEdit struct {
	Changes map[string][]TextEdit
}

type TextEdit struct {
	Range       Range
	NewText     string
}

var commonFixes = map[string][]CodeAction{
	"unused-import": {
		{Title: "Supprimer l'import non utilisé", Kind: "quickfix"},
	},
	"no-body": {
		{Title: "Ajouter un corps de fonction", Kind: "quickfix"},
	},
	"no-doc": {
		{Title: "Ajouter documentation", Kind: "quickfix"},
	},
}

func (lsp *LSPServer) GetCodeActions(filePath string, diag Diagnostic) []CodeAction {
	return commonFixes[diag.Code]
}

func (lsp *LSPServer) RunLint(ctx context.Context, filePath string) ([]Diagnostic, error) {
	lintResults := []Diagnostic{}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.Contains(line, "fmt.Printf") && !strings.Contains(line, "log.Printf") {
			lintResults = append(lintResults, Diagnostic{
				Range: Range{
					Start: Position{Line: i, Character: 0},
					End:   Position{Line: i, Character: len(line)},
				},
				Severity: SeverityWarning,
				Code:     "prefer-log",
				Source:   "linter",
				Message:  "Utilisez log.Printf au lieu de fmt.Printf pour le logging",
			})
		}

		if strings.Contains(line, "time.Sleep") && strings.Contains(line, "time.Second*0") {
			lintResults = append(lintResults, Diagnostic{
				Range: Range{
					Start: Position{Line: i, Character: 0},
					End:   Position{Line: i, Character: len(line)},
				},
				Severity: SeverityInfo,
				Code:     "no-op-sleep",
				Source:   "linter",
				Message:  "Sleep de 0 seconde détecté",
			})
		}

		if regexp.MustCompile(`\bif err != nil \{.*return nil\b`).MatchString(line) {
			lintResults = append(lintResults, Diagnostic{
				Range: Range{
					Start: Position{Line: i, Character: 0},
					End:   Position{Line: i, Character: len(line)},
				},
				Severity: SeverityWarning,
				Code:     "no-error-wrap",
				Source:   "linter",
				Message:  "Enveloppez l'erreur: fmt.Errorf(\"context: %w\", err)",
			})
		}
	}

	return lintResults, nil
}
