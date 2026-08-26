package mcp

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// sdkModule is the module the whole of this surface's protocol comes from, by
// its import path's own prefix. It is spelled here rather than derived from the
// import above so that the fence and the thing it fences are two statements: a
// constant read off the import would agree with whatever the import happened to
// be.
const sdkModule = "github.com/modelcontextprotocol/go-sdk"

// TestDependency_TheSDKIsReachableFromThisPackageAndNoOther is the boundary
// this package exists to draw, held mechanically (§9, issue #195).
//
// The SDK owns the transport, the handshake, `tools/list` and its paging, the
// notification plumbing and the JSON-RPC framing. What it may never own is what
// an answer says — and the way that stays true is that no other package can
// express anything in its terms at all. **The day it is replaced is a day one
// package changes**, which is a claim about the import graph and is therefore
// checkable as one.
//
// It walks the source rather than asking the toolchain, because what it holds
// is about the files a reader opens: a `go list -deps` would answer that the
// module is in the build, which it is, and say nothing about who wrote an
// import of it.
func TestDependency_TheSDKIsReachableFromThisPackageAndNoOther(t *testing.T) {
	root := filepath.Join("..", "..")
	here, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	var importing int
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "testdata"):
			return fs.SkipDir
		case entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go"):
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			named, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(named, sdkModule) {
				continue
			}
			importing++
			holding, err := filepath.Abs(filepath.Dir(path))
			if err != nil {
				return err
			}
			if holding != here {
				t.Errorf("%s imports %s; the SDK is reachable from internal/mcp and no other package", path, named)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if importing == 0 {
		t.Fatal("nothing in the tree imports the SDK; the rule was held over nothing")
	}
}
