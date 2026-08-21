package run_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestEngine_ReachesNoProcessFactOfItsOwn is the property the whole package is
// shaped around, held over its own source (§6, issue #136).
//
// **internal/run reads no environment variable, no clock and no randomness for
// itself.** Every one of them is threaded through Request, which is what makes
// the engine exercisable without a subprocess and what makes every Store path
// and every terminal line a Run writes a checked-in constant rather than a
// value normalised out of a golden (§8, ADR-0047).
//
// It is asserted over the source rather than by driving anything, because that
// is the only reading that catches the regression it is about: a `time.Now()`
// added inside a rarely-reached branch would pass every case in the corpus that
// does not reach it, and would be a Run whose entry nobody can assert the day
// it does.
func TestEngine_ReachesNoProcessFactOfItsOwn(t *testing.T) {
	// The readings, spelled as a reader would type them. `os/exec` and
	// `time` are imported by this package and stay imported — Request names
	// a child launcher and a clock in its own signature — so what is fenced
	// is the **call** and never the import.
	reached := map[string]string{
		"os.Getenv":       "an environment variable",
		"os.LookupEnv":    "an environment variable",
		"os.Environ":      "the environment",
		"os.Hostname":     "the machine's name",
		"time.Now":        "the clock",
		"time.Since":      "the clock",
		"rand.Read":       "randomness",
		"rand.Int":        "randomness",
		"rand.Intn":       "randomness",
		"store.MintRunID": "a Run id, which is randomness and a clock at once",
	}

	files, err := parser.ParseDir(token.NewFileSet(), ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	var judged int
	for name, pkg := range files {
		if strings.HasSuffix(name, "_test") {
			continue
		}
		ast.Inspect(pkg, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall {
				return true
			}
			judged++
			selector, isSelector := call.Fun.(*ast.SelectorExpr)
			if !isSelector {
				return true
			}
			pkgName, isIdent := selector.X.(*ast.Ident)
			if !isIdent {
				return true
			}
			if what, reaches := reached[pkgName.Name+"."+selector.Sel.Name]; reaches {
				t.Errorf("the engine calls %s.%s, which reads %s for itself; every process fact is threaded through Request",
					pkgName.Name, selector.Sel.Name, what)
			}
			return true
		})
	}

	if judged == 0 {
		t.Fatal("the walk found no call in the package at all; the fence held vacuously")
	}
}
