package si18n

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// assert will log the given message if condition is false.
func assert(condition bool, t testing.TB, msg string, v ...interface{}) {
	assertUp(condition, t, 1, msg, v...)
}

// assertUp is like assert, but used inside helper functions, to ensure that
// the file and line number reported by failures corresponds to one or more
// levels up the stack.
func assertUp(condition bool, t testing.TB, caller int, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(caller + 1)
		v = append([]interface{}{filepath.Base(file), line}, v...)
		fmt.Printf("%s:%d: "+msg+"\n", v...)
		t.FailNow()
	}
}

// equals tests that the two values are equal according to reflect.DeepEqual.
func equals(exp, act interface{}, t testing.TB) {
	equalsUp(exp, act, t, 1)
}

// equalsUp is like equals, but used inside helper functions, to ensure that the
// file and line number reported by failures corresponds to one or more levels
// up the stack.
func equalsUp(exp, act interface{}, t testing.TB, caller int) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(caller + 1)
		fmt.Printf("%s:%d: exp: %v (%T), got: %v (%T)\n",
			filepath.Base(file), line, exp, exp, act, act)
		t.FailNow()
	}
}

// isNil reports a failure if the given value is not nil.  Note that values
// which cannot be nil will always fail this check.
func isNil(obtained interface{}, t testing.TB) {
	isNilUp(obtained, t, 1)
}

// isNilUp is like isNil, but used inside helper functions, to ensure that the
// file and line number reported by failures corresponds to one or more levels
// up the stack.
func isNilUp(obtained interface{}, t testing.TB, caller int) {
	if !_isNil(obtained) {
		_, file, line, _ := runtime.Caller(caller + 1)
		fmt.Printf("%s:%d: expected nil, got: %v\n", filepath.Base(file), line, obtained)
		t.FailNow()
	}
}

// notNil reports a failure if the given value is nil.
func notNil(obtained interface{}, t testing.TB) {
	notNilUp(obtained, t, 1)
}

// notNilUp is like notNil, but used inside helper functions, to ensure that the
// file and line number reported by failures corresponds to one or more levels
// up the stack.
func notNilUp(obtained interface{}, t testing.TB, caller int) {
	if _isNil(obtained) {
		_, file, line, _ := runtime.Caller(caller + 1)
		fmt.Printf("%s:%d: expected non-nil, got: %v\n", filepath.Base(file), line, obtained)
		t.FailNow()
	}
}

func _isNil(i any) bool {
	if i == nil {
		return true
	}

	type Nil interface {
		IsNil() bool
	}
	if in, ok := i.(Nil); ok {
		return in.IsNil()
	}
	vo := reflect.ValueOf(i)
	k := vo.Kind()
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return vo.IsNil()
	default:
		return false
	}
}

type TmpDir struct {
	path    string
	handles []*os.File
}

func (t *TmpDir) Name() string {
	return filepath.Base(t.Path())
}

func (t *TmpDir) Path() string {
	return t.path
}

func (t *TmpDir) RemoveAll() {
	_ = os.RemoveAll(t.path)
	for i := 0; i < len(t.handles); i++ {
		_ = t.handles[i].Close()
		t.handles[i] = nil
	}
}

func (t *TmpDir) CreateFile(name string) *os.File {
	if len(name) == 0 {
		return nil
	}
	name = filepath.Join(t.path, name)
	file, err := os.Create(name)
	if file != nil && err == nil {
		t.handles = append(t.handles, file)
		return file
	}

	return nil
}

func (t *TmpDir) SubTmpDir(name string, tb testing.TB) *TmpDir {
	assert(len(name) != 0, tb, "exp: non-zero name")
	path := filepath.Join(t.path, name)
	err := os.Mkdir(path, 0700)
	isNilUp(err, tb, 1)
	td := &TmpDir{
		path:    path,
		handles: nil,
	}
	return td
}

func makeTmpDir(tb testing.TB) *TmpDir {
	temp, err := os.MkdirTemp(os.TempDir(), "si18n-tmp-*")
	isNilUp(err, tb, 1)
	err = os.Chmod(temp, 0700)
	isNilUp(err, tb, 1)
	td := &TmpDir{
		path:    temp,
		handles: nil,
	}
	return td
}
