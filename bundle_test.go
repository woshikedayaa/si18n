package si18n

import (
	"net/http"
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
		equals(v, bundle.TryTr(k), t)
		equals(v, bundle.MustTr(k), t)
		tr, err := bundle.Tr(k)
		isNil(err, t)
		equals(v, tr, t)
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

	err = bundle.LoadFromDir(dir.Path())
	isNil(err, t)
	m := getExampleCorrectMap()
	for k, v := range m {
		equals(v, bundle.TryTr(k), t)
		equals(v, bundle.MustTr(k), t)
		tr, err := bundle.Tr(k)
		isNil(err, t)
		equals(v, tr, t)
	}
}

func TestBundle_LoadMap(t *testing.T) {
	bundle := New(SystemLanguage())
	bundle.LoadFromMap(getExampleCorrectMap(), "")
	m := getExampleCorrectMap()
	for k, v := range m {
		equals(v, bundle.TryTr(k), t)
	}
	prefix := "0"
	bundle.LoadFromMap(getExampleCorrectMap(), prefix)
	for key, v := range m {
		k := strings.Join([]string{prefix, key}, ".")
		equals(v, bundle.TryTr(k), t)
	}
}

func TestBundle_TrWithTemplate(t *testing.T) {
	s := `
a: 
  b:
    1: "{{.test}}"
    2: "{{.test}}"
    3: "{{.test}}"
    c: 
      1: "{{.test}}"
      2: "{{.test}}"
`

	dir := makeTmpDir(t)
	defer dir.RemoveAll()
	file := dir.CreateFile(SystemLanguage().String() + ".yaml")
	notNil(file, t)
	_, err := file.Write([]byte(s))
	isNil(err, t)
	bundle := New(SystemLanguage())
	notNil(bundle, t)

	err = bundle.loadFile(file.Name())
	isNil(err, t)
	m := getExampleCorrectMap()
	for k, v := range m {
		ms := map[string]any{
			"test": v,
		}
		equals(v, bundle.TryTr(k, ms), t)
		equals(v, bundle.MustTr(k, ms), t)
		tr, err := bundle.Tr(k, ms)
		isNil(err, t)
		equals(v, tr, t)
	}
}

func TestBundle_LoadFromHttp(t *testing.T) {
	finish := make(chan struct{})
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /test", func(resp http.ResponseWriter, req *http.Request) {
			_, _ = resp.Write([]byte(getExampleFileString()))
		})
		mux.HandleFunc("GET /test.yaml", func(resp http.ResponseWriter, req *http.Request) {
			_, _ = resp.Write([]byte(getExampleFileString()))
		})
		finish <- struct{}{}
		err := http.ListenAndServe("localhost:8080", mux)
		isNil(err, t)
	}()
	<-finish

	bundle := New(SystemLanguage())
	err := bundle.LoadFromHttp("http://localhost:8080/test", "yaml")
	isNil(err, t)
	m := getExampleCorrectMap()
	for k, v := range m {
		equals(v, bundle.TryTr(k), t)
		equals(v, bundle.MustTr(k), t)
		tr, err := bundle.Tr(k)
		isNil(err, t)
		equals(v, tr, t)
	}
}

func TestBundle_LoadFromHttpWithSuffix(t *testing.T) {
	finish := make(chan struct{})
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /test.yaml", func(resp http.ResponseWriter, req *http.Request) {
			_, _ = resp.Write([]byte(getExampleFileString()))
		})
		finish <- struct{}{}
		err := http.ListenAndServe("localhost:8081", mux)
		isNil(err, t)
	}()
	<-finish

	bundle := New(SystemLanguage())
	err := bundle.LoadFromHttp("http://localhost:8081/test.yaml", "")
	isNil(err, t)
	m := getExampleCorrectMap()
	for k, v := range m {
		equals(v, bundle.TryTr(k), t)
		equals(v, bundle.MustTr(k), t)
		tr, err := bundle.Tr(k)
		isNil(err, t)
		equals(v, tr, t)
	}
}
