package si18n

import (
	"strings"
	"testing"
)

func getExampleFileString() string {
	return `
a: 
  b:
    1: b1
    2: b2
    3: b3
    c: 
      1: c1
      2: c2
`
}

func getExampleCorrectMap() map[string]any {
	return map[string]any{
		"a.b.1":   "b1",
		"a.b.2":   "b2",
		"a.b.3":   "b3",
		"a.b.c.1": "c1",
		"a.b.c.2": "c2",
	}
}

func TestBundle_LoadFile(t *testing.T) {
	dir := makeTmpDir(t)
	defer dir.RemoveAll()
	file := dir.CreateFile(SystemLanguage().String() + ".yaml")
	notNil(file, t)
	_, err := file.Write([]byte(getExampleFileString()))
	isNil(err, t)
	bundle := New(SystemLanguage())
	notNil(bundle, t)

	err = bundle.loadFile(file.Name())
	isNil(err, t)
	m := getExampleCorrectMap()
	for k, v := range m {
		equals(v, bundle.TR(k, nil), t)
	}
}

func TestBundle_LoadDir(t *testing.T) {
	dir := makeTmpDir(t)
	defer dir.RemoveAll()
	langDir := dir.SubTmpDir(SystemLanguage().String(), t)
	defer langDir.RemoveAll()
	file := langDir.CreateFile("test.yaml")
	notNil(file, t)
	_, err := file.Write([]byte(getExampleFileString()))
	isNil(err, t)
	bundle := New(SystemLanguage())
	notNil(bundle, t)

	err = bundle.LoadDir(dir.Path())
	isNil(err, t)
	m := getExampleCorrectMap()
	for k, v := range m {
		equals(v, bundle.TR(k, nil), t)
	}
}

func TestBundle_LoadMap(t *testing.T) {
	bundle := New(SystemLanguage())
	bundle.LoadMap(getExampleCorrectMap(), "")
	m := getExampleCorrectMap()
	for k, v := range m {
		equals(v, bundle.TR(k, nil), t)
	}
	prefix := "0"
	bundle.LoadMap(getExampleCorrectMap(), prefix)
	for key, v := range m {
		k := strings.Join([]string{prefix, key}, ".")
		equals(v, bundle.TR(k, nil), t)
	}
}
