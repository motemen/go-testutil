package dataloc

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

func logf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

const debug = false

func debugf(format string, args ...interface{}) {
	if debug {
		log.Printf("debug: "+format, args...)
	}
}

// L returns the source code location of the test case identified by its name.
// See Example.
// It attempts runtime source code analysis to find the location
// by using the expression passed to dataloc.L().
// So some restrictions apply:
// - The function must be invoked as "dataloc.L".
// - The argument must be an expression of the form "testcase.key",
// - where "testcase" is a variable declared as "for _, testcase := range testcases",
// - and "testcases" is a slice of a struct type,
// - whose "key" field is a string which is passsed to L().
func L(name string) string {
	s, _ := loc(name)
	return s
}

// TODO: allow testcases to be a map, eg
//
//	for name, testcase := range testcases {
//	  ... dataloc.L(name) ...
//	}
func loc(value string) (string, error) {
	_, file, line, _ := runtime.Caller(2)

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	file, err = filepath.Rel(cwd, file)
	if err != nil {
		return "", err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		return "", err
	}

	objToTypeDecl := make(map[*ast.Object]ast.Expr)
	objToVarInit := make(map[*ast.Object]ast.Expr)
	// make [ v ↦ expr ] mapping from "for k, v := range expr"
	objToRangeExprForValue := make(map[*ast.Object]ast.Expr)
	objToRangeExprForKey := make(map[*ast.Object]ast.Expr)

	ast.Inspect(f, func(n ast.Node) bool {
		if rangeStmt, ok := n.(*ast.RangeStmt); ok {
			if ident, ok := rangeStmt.Value.(*ast.Ident); ok {
				objToRangeExprForValue[ident.Obj] = rangeStmt.X
			}
			if ident, ok := rangeStmt.Key.(*ast.Ident); ok {
				objToRangeExprForKey[ident.Obj] = rangeStmt.X
			}
		} else if decl, ok := n.(ast.Decl); ok {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				debugf("declStmt: %+v", genDecl)
				if genDecl.Tok == token.VAR {
					for _, spec := range genDecl.Specs {
						if valueSpec, ok := spec.(*ast.ValueSpec); ok {
							for i, name := range valueSpec.Names {
								objToVarInit[name.Obj] = valueSpec.Values[i]
							}
						}
					}
				} else if genDecl.Tok == token.TYPE {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							objToTypeDecl[typeSpec.Name.Obj] = typeSpec.Type
						}
					}
				}
			}
		} else if assignStmt, ok := n.(*ast.AssignStmt); ok {
			debugf("assignStmt: %+v", assignStmt)
			for i, expr := range assignStmt.Lhs {
				if ident, ok := expr.(*ast.Ident); ok {
					if len(assignStmt.Lhs) == len(assignStmt.Rhs) {
						objToVarInit[ident.Obj] = assignStmt.Rhs[i]
					} else if len(assignStmt.Rhs) == 1 {
						objToVarInit[ident.Obj] = assignStmt.Rhs[0]
					} else {
						debugf("unreachable: len(assignStmt.Lhs)=%d, len(assignStmt.Rhs)=%d", len(assignStmt.Lhs), len(assignStmt.Rhs))
					}
				}
			}
		}

		return true
	})

	loc := "(unknown)"
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		pos := fset.Position(n.Pos())
		if pos.Line != line {
			return true
		}

		// for example:
		//   testcases := []struct{}{...}
		//   for _, testdata := range testcases {
		//     dataloc.L(testdata.name)
		//   }
		if call, ok := isMethodCall(n, "dataloc", "L"); ok {
			debugf("found %+v", n)
			arg := call.Args[0]
			// ident = testdata, key = name
			if ident, key, ok := isSelector(arg); ok {
				debugf("arg[0]: %s.%s", ident, key)

				// expr = testcases
				if expr, ok := objToRangeExprForValue[ident.Obj]; ok {
					debugf("range: %s", expr)

					if sourceIdent, ok := expr.(*ast.Ident); ok {
						// varInit = []struct{}{...}
						varInit := objToVarInit[sourceIdent.Obj]
						debugf("varInit: %+v", varInit)
						node := findTestCaseItem(varInit, key, value, objToTypeDecl)
						if node != nil {
							debugf("⭐ pos: %s", fset.Position(node.Pos()))
							pos := fset.Position(node.Pos())
							loc = fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
							return false
						}
					}
				}
			} else if ident, ok := arg.(*ast.Ident); ok {
				// for k, v := range testcases {
				//   dataloc.L(k)
				// }
				if expr, ok := objToRangeExprForKey[ident.Obj]; ok {
					if sourceIdent, ok := expr.(*ast.Ident); ok {
						varInit := objToVarInit[sourceIdent.Obj]
						node := findTestCaseItem(varInit, ident.Name, value, objToTypeDecl)
						if node != nil {
							debugf("⭐ pos: %s", fset.Position(node.Pos()))
							pos := fset.Position(node.Pos())
							loc = fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
							return false
						}
					}
				}
			}
		}

		return true
	})

	return loc, nil
}

func isMethodCall(n ast.Node, obj, fun string) (*ast.CallExpr, bool) {
	if call, ok := n.(*ast.CallExpr); ok {
		if ident, name, ok := isSelector(call.Fun); ok {
			if ident.Name == obj && name == fun {
				return call, true
			}
		}
	}
	return nil, false
}

func isSelector(n ast.Node) (*ast.Ident, string, bool) {
	if sel, ok := n.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident, sel.Sel.Name, true
		}
	}
	return nil, "", false
}

func findTestCaseItem(init ast.Expr, key, value string, objToTypeDecl map[*ast.Object]ast.Expr) ast.Node {
	testcases, ok := init.(*ast.CompositeLit)
	if !ok {
		return nil
	}

	var testcaseType ast.Expr
	if t, ok := testcases.Type.(*ast.ArrayType); ok {
		testcaseType = t.Elt
		if ident, ok := testcaseType.(*ast.Ident); ok {
			testcaseType = objToTypeDecl[ident.Obj]
			if testcaseType == nil {
				logf("could not resolve type of %s", ident.Name)
				return nil
			}
		}
	} else if m, ok := testcases.Type.(*ast.MapType); ok {
		testcaseType = m
	} else {
		// testcases should be an array eg.
		//   testcases := []testcase{ ... }
		// or a map eg.
		//   testcases := map[string]testcase{ ... }
		debugf("unexpected testcase type: %#v", testcases.Type)
		return nil
	}

	for _, testcase := range testcases.Elts {
		debugf("testacse: %v (%T)", testcase, testcase)

		if kv, ok := testcase.(*ast.KeyValueExpr); ok {
			if basic, ok := kv.Key.(*ast.BasicLit); ok {
				debugf("item basic=%v (%T) vs %q", basic, basic, value)
				if isStringLiteral(basic, value) {
					return kv
				}
			}
		}

		testcase, ok := testcase.(*ast.CompositeLit)
		if !ok {
			// testcase should be a struct literal eg.
			//   { name: "foo", ... }
			// or
			//   { "foo", ... }
			continue
		}

		for i, field := range testcase.Elts {
			if kv, ok := field.(*ast.KeyValueExpr); ok {
				// { <key>: <value>, ... }
				if ident, ok := kv.Key.(*ast.Ident); ok {
					if ident.Name == key {
						debugf("item value=%v (%T) vs %q", kv.Value, kv.Value, value)
						if isStringLiteral(kv.Value, value) {
							return testcase
						}
					}
				}
			} else if basic, ok := field.(*ast.BasicLit); ok {
				// { <value>, ...}
				debugf("item basic=%v (%T) vs %q", basic, basic, value)
				if findStructFieldIndex(testcaseType, key) == i {
					if isStringLiteral(basic, value) {
						return testcase
					}
				}
			}
		}
	}

	return nil
}

func isStringLiteral(n ast.Expr, s string) bool {
	lit, ok := n.(*ast.BasicLit)
	if !ok {
		return false
	}
	if lit.Kind != token.STRING {
		return false
	}
	return lit.Value == strconv.Quote(s)
}

func findStructFieldIndex(t ast.Expr, name string) int {
	typ, ok := t.(*ast.StructType)
	if !ok {
		return -1
	}

	for i, field := range typ.Fields.List {
		for _, ident := range field.Names {
			if ident.Name == name {
				return i
			}
		}
	}

	return -1
}
