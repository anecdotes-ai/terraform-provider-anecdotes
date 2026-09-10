// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestResourceStateShapeConsistency guards the state-shape contract: every model field
// Read populates from an API response must also be populated by Create and by Update.
// Read defines the shape a resource settles into after a refresh, so a field Create or
// Update leaves at its planned value while Read sets it from the response produces a
// difference on the next plan.
//
// A resource satisfies the contract either by routing all three methods through the same
// mapping helper, or by assigning at least the same fields inline. Methods may populate
// more than Read does; only Read's fields are required everywhere.
//
// Resources that cannot satisfy this on Update are listed in updateExempt, with the
// reason: either every settable attribute is RequiresReplace, so the framework replaces
// the resource rather than calling Update, or the update endpoint returns no body.
func TestResourceStateShapeConsistency(t *testing.T) {
	// Resources whose Update cannot populate state from a response. Either Update is
	// unreachable because every settable attribute forces replacement, or the endpoint
	// it calls returns no representation to map.
	updateExempt := map[string]string{
		"mapping_control_requirement":  "control_id and requirement_id are RequiresReplace; framework_id is Computed",
		"mapping_requirement_evidence": "requirement_id and evidence_id are RequiresReplace",
		"control_category":             "UpdateControlCategory returns no body, so Update has no response to map",
	}

	files, err := filepath.Glob("*_resource.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no resource files found")
	}

	var violations []string
	for _, file := range files {
		name := strings.TrimSuffix(file, "_resource.go")

		methods, perr := parseStatePopulation(file)
		if perr != nil {
			t.Fatalf("parse %s: %v", file, perr)
		}
		read, ok := methods["Read"]
		if !ok {
			continue // data-source-like or read-less resource
		}

		for _, method := range []string{"Create", "Update"} {
			if method == "Update" {
				if _, skip := updateExempt[name]; skip {
					continue
				}
			}
			missing := read.missingFrom(methods[method])
			if len(missing) == 0 {
				continue
			}
			violations = append(violations, fmt.Sprintf(
				"%s: %s does not populate %s, which Read sets from the API response",
				file, method, strings.Join(missing, ", ")))
		}
	}

	if len(violations) > 0 {
		t.Fatalf("found %d state-shape divergence(s):\n  %s\n\nRoute Create, Read and Update through one mapping helper, "+
			"or assign the same fields in each.", len(violations), strings.Join(violations, "\n  "))
	}
}

// statePopulation records how one method fills the resource model: the fields it assigns
// directly, and the helpers it passes the model to.
type statePopulation struct {
	fields  map[string]bool
	mappers map[string]bool
}

// missingFrom returns the fields of r that other populates neither through a shared
// mapper nor by direct assignment.
func (r statePopulation) missingFrom(other statePopulation) []string {
	for mapper := range r.mappers {
		if !other.mappers[mapper] {
			return []string{"the result of " + mapper}
		}
	}

	var missing []string
	for field := range r.fields {
		if !other.fields[field] {
			missing = append(missing, field)
		}
	}
	sort.Strings(missing)
	return missing
}

// modelVars are the local names the resource methods bind the model to.
var modelVars = map[string]bool{"data": true, "state": true, "plan": true}

// parseStatePopulation reports, for Create, Read and Update, which model fields each one
// assigns and which helpers it hands the model to.
func parseStatePopulation(file string) (map[string]statePopulation, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		return nil, err
	}

	out := map[string]statePopulation{}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Body == nil {
			continue
		}
		switch fn.Name.Name {
		case "Create", "Read", "Update":
			out[fn.Name.Name] = scanStatePopulation(fn)
		}
	}
	return out, nil
}

func scanStatePopulation(fn *ast.FuncDecl) statePopulation {
	found := statePopulation{fields: map[string]bool{}, mappers: map[string]bool{}}

	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.AssignStmt:
			for _, lhs := range n.Lhs {
				if field, ok := modelFieldName(lhs); ok {
					found.fields[field] = true
				}
			}
		case *ast.CallExpr:
			if name, ok := mapperName(n); ok {
				found.mappers[name] = true
			}
		}
		return true
	})
	return found
}

// modelFieldName returns the field name of a `data.Field` style assignment target.
func modelFieldName(expr ast.Expr) (string, bool) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || !modelVars[ident.Name] {
		return "", false
	}
	return sel.Sel.Name, true
}

// mapperName returns the name of a call that is handed the model itself, which is how a
// resource delegates state population to a shared helper.
func mapperName(call *ast.CallExpr) (string, bool) {
	var name string
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		name = fun.Name
	case *ast.SelectorExpr:
		name = fun.Sel.Name
	default:
		return "", false
	}
	// Framework plumbing reads the model rather than populating it.
	if name == "Get" || name == "Set" || name == "Append" || strings.HasPrefix(name, "Get") {
		return "", false
	}
	for _, arg := range call.Args {
		if isModelExpr(arg) {
			return name, true
		}
	}
	return "", false
}

func isModelExpr(expr ast.Expr) bool {
	if unary, ok := expr.(*ast.UnaryExpr); ok {
		expr = unary.X
	}
	ident, ok := expr.(*ast.Ident)
	return ok && modelVars[ident.Name]
}
