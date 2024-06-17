package si18n

import (
	"golang.org/x/text/language"
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

func getExampleCorrectMapKeys() []string {
	m := getExampleCorrectMap()
	res := make([]string, 0, len(m))
	for k, _ := range m {
		res = append(res, k)
	}
	return res
}

func TestBundle_LoadFile(t *testing.T) {
	dir := makeTmpDir(t)
	defer dir.RemoveAll()
	file := dir.CreateFile("zh-Hans.yaml")
	notNil(file, t)
	_, err := file.Write([]byte(getExampleFileString()))
	isNil(err, t)
	bundle := New(language.SimplifiedChinese)
	notNil(bundle, t)

	err = bundle.loadFile(file.Name())
	isNil(err, t)
	m := getExampleCorrectMap()
	keys := getExampleCorrectMapKeys()
	for _, v := range keys {
		equals(m[v], bundle.TR(v, nil), t)
	}
}

func TestBundle_LoadDir(t *testing.T) {
	dir := makeTmpDir(t)
	defer dir.RemoveAll()
	langDir := dir.SubTmpDir("zh-Hans", t)
	defer langDir.RemoveAll()
	file := langDir.CreateFile("test.yaml")
	notNil(file, t)
	_, err := file.Write([]byte(getExampleFileString()))
	isNil(err, t)
	bundle := New(language.SimplifiedChinese)
	notNil(bundle, t)

	err = bundle.LoadDir(dir.Path())
	isNil(err, t)
	m := getExampleCorrectMap()
	keys := getExampleCorrectMapKeys()
	for _, v := range keys {
		equals(m[v], bundle.TR(v, nil), t)
	}
}

func TestBundle_LoadMap(t *testing.T) {
	bundle := New(language.SimplifiedChinese)
	bundle.LoadMap(getExampleCorrectMap(), "")
	m := getExampleCorrectMap()
	keys := getExampleCorrectMapKeys()
	for _, v := range keys {
		equals(m[v], bundle.TR(v, nil), t)
	}
	prefix := "0"
	bundle.LoadMap(getExampleCorrectMap(), prefix)
	for _, v := range keys {
		k := strings.Join([]string{prefix, v}, ".")
		equals(m[v], bundle.TR(k, nil), t)
	}
}
