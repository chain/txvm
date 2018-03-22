package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chain/txvm/errors"
)

var wd, _ = os.Getwd()

// FatalErr calls t.Fatal with err and a stack trace.
func FatalErr(t testing.TB, err error) {
	args := []interface{}{err}
	args = append(args, errors.Data(err))
	stack := errors.Stack(err)
	for frame, ok := stack.Next(); ok; frame, ok = stack.Next() {
		file := frame.File
		if rel, err := filepath.Rel(wd, file); err == nil && !strings.HasPrefix(rel, "../") {
			file = rel
		}
		funcname := frame.Function[strings.IndexByte(frame.Function, '.')+1:]
		s := fmt.Sprintf("\n%s:%d: %s", file, frame.Line, funcname)
		args = append(args, s)
	}
	t.Fatal(args...)
}
