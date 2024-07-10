package si18n

import (
	"embed"
	"errors"
	"fmt"
	"golang.org/x/text/language"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"text/template"
)

var (
	ErrEmpty                       = errors.New("empty")
	ErrUnsupportedFormat           = errors.New("unsupported format")
	ErrUnknownFormat               = errors.New("unknown format")
	ErrTargetIsRegular             = errors.New("target path is a regular file")
	ErrTargetIsDir                 = errors.New("target path is a dir")
	ErrIncorrectBytesUnmarshalFunc = errors.New("incorrect bytes unmarshaler")
	ErrNotFound                    = errors.New("can not found translations")
	ErrIncorrectRemoteProtocol     = errors.New("incorrect remote protocol")
)

type MessageObject struct {
	updated bool
	key     string
	val     string
	tmpl    *template.Template
}

func NewMessage(key, val string) *MessageObject {
	return &MessageObject{
		updated: true,
		key:     key,
		val:     val,
		tmpl:    nil,
	}
}

func (m *MessageObject) Update() {
	if !m.updated {
		return
	}
	m.tmpl = nil
	m.updated = false
}

func (m *MessageObject) Template() (*template.Template, error) {
	if m.tmpl != nil && !m.updated {
		return m.tmpl, nil
	}
	parse, err := template.New(m.key).Parse(m.val)
	if err != nil {
		return nil, err
	}
	m.tmpl = parse
	return parse, nil
}

func (m *MessageObject) MustTemplate() *template.Template {
	if tmpl, err := m.Template(); err != nil {
		panic(err)
	} else {
		return tmpl
	}
}

type NotFoundHandler func(key string, writer io.Writer, ms ...map[string]any)

type Bundle struct {
	lang            language.Tag
	cache           *LRUCache
	all             map[string]*MessageObject
	lock            *sync.RWMutex
	updateList      []string
	notFoundHandler NotFoundHandler
}

func (b *Bundle) NotFoundHandler(handler NotFoundHandler) {
	b.notFoundHandler = handler
}

func (b *Bundle) flatten(prefix string, data any) {
	switch val := data.(type) {
	case string:
		b.all[prefix] = nil
		b.all[prefix] = NewMessage(prefix, val)
		b.updateList = append(b.updateList, prefix)
	case float64, int:
		s := fmt.Sprint(val)
		b.all[prefix] = nil
		b.all[prefix] = NewMessage(prefix, s)
		b.updateList = append(b.updateList, prefix)
	case []any:
		for i := 0; i < len(val); i++ {
			pf := strings.Join([]string{prefix, fmt.Sprintf("[%d]", i)}, ".")
			b.flatten(pf, val[i])
		}
	case map[string]any:
		for k, v := range val {
			pf := strings.Join([]string{prefix, k}, ".")
			b.flatten(pf, v)
		}
	case map[any]any:
		for k, v := range val {
			pf := strings.Join([]string{prefix, fmt.Sprint(k)}, ".")
			b.flatten(pf, v)
		}
	default:
		// skip
	}
}

func (b *Bundle) LoadFromFile(path string) error {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return errorWarp(fmt.Errorf("LoadFromFile: %w: %s", err, path))
	}
	if stat.IsDir() {
		return errorWarp(fmt.Errorf("LoadFromFile: %w: %s", ErrTargetIsDir, path))
	}
	err = b.loadFile(path)
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromFile: %w", err))
	}
	return nil
}

type UnmarshalFunc func(data []byte, obj any) error

func (b *Bundle) loadFile(path string) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// read the suffix
	ss := strings.Split(filepath.Base(path), ".")
	format, err := parseFormat(ss[len(ss)-1])
	if err != nil {
		return err
	}
	return b.loadBytes(buf, getUnmarshalFunc(format))
}

func (b *Bundle) loadBytes(bs []byte, unmarshaler UnmarshalFunc) error {
	if len(bs) == 0 {
		return ErrEmpty
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	var val any
	var err error

	//
	err = unmarshaler(bs, &val)
	if err != nil {
		return err
	}
	switch v := val.(type) {
	case []any:
		for i := 0; i < len(v); i++ {
			b.flatten(fmt.Sprintf("[%d]", i), v[i])
		}
	case map[string]any:
		for key, val := range v {
			b.flatten(key, val)
		}
	case map[any]any:
		for key, val := range v {
			b.flatten(fmt.Sprint(key), val)
		}
	default:
		return ErrIncorrectBytesUnmarshalFunc
	}
	return nil
}

func (b *Bundle) LoadFromBytes(bs []byte, format string) error {
	ft, err := parseFormat(format)
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromBytes: %w: %s", err, format))
	}
	err = b.loadBytes(bs, getUnmarshalFunc(ft))
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromBytes: %w", err))
	}
	return nil
}

func (b *Bundle) LoadFromMap(m map[string]any, prefix string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	for k, v := range m {
		if len(prefix) == 0 {
			b.flatten(k, v)
		} else {
			b.flatten(strings.Join([]string{prefix, k}, "."), v)
		}
	}
}

func (b *Bundle) LoadFromHttp(u string, format string) error {
	up, err := url.Parse(u)
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromHttp: parseurl: %w", err))
	}
	if len(up.Scheme) == 0 {
		up.Scheme = "http"
	}

	if up.Scheme != "http" && up.Scheme != "https" {
		return errorWarp(fmt.Errorf("LoadFromHttp: %w: %s .except %s,%s", ErrIncorrectRemoteProtocol, up.Scheme, "http", "https"))
	}
	if len(format) == 0 {
		// auto get format from url path
		base := path.Base(up.Path)
		ss := strings.Split(base, ".")
		if len(ss) <= 1 {
			return errorWarp(fmt.Errorf("LoadFromHttp: %w", ErrUnknownFormat))
		}
		format = ss[len(ss)-1]
	}
	ft, err := parseFormat(format)
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromHttp: %w: %s", err, format))
	}
	u = up.String()
	resp, err := http.Get(u)
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromHttp: get: %s error %w", u, err))
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromHttp: read: %w", err))
	}
	err = b.loadBytes(bytes, getUnmarshalFunc(ft))
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromHttp: parse: %w", err))
	}
	return nil
}

func (b *Bundle) LoadFromReader(r io.Reader, format string) error {
	all, err := io.ReadAll(r)
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromReader: %w", err))
	}
	f, err := parseFormat(format)
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromReader: %w: %s", err, format))
	}
	err = b.loadBytes(all, getUnmarshalFunc(f))
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromReader: %w", err))
	}
	return nil
}

func (b *Bundle) LoadFromFs(f *embed.FS) error {
	var targetDir string
	_, err := f.ReadDir(b.lang.String())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errorWarp(fmt.Errorf("LoadFromFs: %w", err))
	}
	if errors.Is(err, os.ErrNotExist) {
		targetDir = "."
	} else {
		targetDir = b.lang.String()
	}
	err = fs.WalkDir(f, targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ss := strings.Split(filepath.Base(path), ".")
		if len(ss) <= 1 {
			return nil
		}
		format, err := parseFormat(ss[len(ss)-1])
		if err != nil {
			// skip this file
			return nil
		}
		data, err2 := f.ReadFile(path)
		if err2 != nil {
			return err2
		}
		err2 = b.loadBytes(data, getUnmarshalFunc(format))
		if err2 != nil {
			return err2
		}
		return nil
	})
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromFs: %w", err))
	}
	return nil
}

func (b *Bundle) LoadFromDir(dir string) error {
	rootStat, err := os.Stat(dir)
	// check rootStat is a dir
	if rootStat != nil && !rootStat.IsDir() {
		return errorWarp(fmt.Errorf("LoadFromDir: %w: %s", ErrTargetIsRegular, dir))
	}

	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromDir: %w: %s", err, dir))
	}

	// find if there is a dir named b.lang.String()
	stat, err := os.Stat(filepath.Join(dir, b.lang.String()))
	if err != nil && !os.IsNotExist(err) {
		return errorWarp(fmt.Errorf("LoadFromDir: %w", err))
	}
	if stat != nil && stat.IsDir() {
		err = b.loadDir(filepath.Join(dir, b.lang.String()))
		if err != nil {
			return errorWarp(fmt.Errorf("LoadFromDir: %w", err))
		}
		return nil
	}
	// just read files
	files, err := os.ReadDir(dir)
	if err != nil {
		return errorWarp(fmt.Errorf("LoadFromDir: %w", err))
	}

	availableList := map[string]bool{
		b.lang.String() + "." + "yaml": true,
		b.lang.String() + "." + "yml":  true,
		b.lang.String() + "." + "json": true,
		b.lang.String() + "." + "toml": true,
	}
	files = slices.DeleteFunc(files, func(entry os.DirEntry) bool {
		if entry.IsDir() {
			return true
		}
		return !availableList[entry.Name()]
	})
	for i := 0; i < len(files); i++ {
		err := b.loadFile(filepath.Join(dir, files[i].Name()))
		if err != nil {
			return errorWarp(fmt.Errorf("LoadFromDir: %w", err))
		}
	}
	return nil
}

func (b *Bundle) loadDir(dir string) error {
	// just read the dir files and bind to map
	return filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		err2 := b.loadFile(path)
		if err2 != nil {
			return fmt.Errorf("%w:%s", err2, path)
		}
		return nil
	})
}

func (b *Bundle) Tr2Writer(key string, writer io.Writer, ms ...map[string]any) error {
	b.update()
	messageObj, ok := b.cache.Get(key)
	if !ok {
		messageObj, ok = b.all[key]
		if !ok {
			return errorWarp(fmt.Errorf("Tr2Writer: %w: lang: %s key: %s", ErrNotFound, b.lang, key))
		}
		b.cache.Put(key, messageObj)
	}
	t, err := messageObj.Template()
	if err != nil {
		return errorWarp(fmt.Errorf("Tr2Writer: exec: %w", err))
	}
	err = t.Execute(writer, mergeMap(ms...))
	if err != nil {
		return errorWarp(fmt.Errorf("Tr2Writer: exec: %w", err))
	}
	return nil
}

func (b *Bundle) Tr(key string, ms ...map[string]any) (string, error) {
	res := &strings.Builder{}
	err := b.Tr2Writer(key, res, ms...)
	return res.String(), err
}

func (b *Bundle) TryTr(key string, ms ...map[string]any) string {
	res := &strings.Builder{}
	b.TryTr2Writer(key, res, ms...)
	return res.String()
}

func (b *Bundle) MustTr(key string, ms ...map[string]any) string {
	res := &strings.Builder{}
	b.MustTr2Writer(key, res, ms...)
	return res.String()
}

func (b *Bundle) TryTr2Writer(key string, writer io.Writer, ms ...map[string]any) {
	err := b.Tr2Writer(key, writer, ms...)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			b.notFoundHandler(key, writer, ms...)
		} else {
			// noop
		}
	}
}

func (b *Bundle) MustTr2Writer(key string, writer io.Writer, ms ...map[string]any) {
	if err := b.Tr2Writer(key, writer, ms...); err != nil {
		panic(err)
	}
}

func (b *Bundle) update() {
	if len(b.updateList) == 0 {
		return
	}
	for i := 0; i < len(b.updateList); i++ {
		b.all[b.updateList[i]].Update()
		// remove it from cache
		b.cache.Remove(b.updateList[i])
	}
	// resize the cache
	if len(b.all) <= 1024 {
		if len(b.all)/3 >= 64 {
			b.cache.Resize(len(b.all) / 3)
		}
	} else {
		b.cache.Resize(len(b.all) / 4)
	}
	b.updateList = []string{}
}

func (b *Bundle) GetAllTR() map[string]*MessageObject {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.all
}

func (b *Bundle) Language() language.Tag {
	return b.lang
}

func New(tag language.Tag) *Bundle {
	b := &Bundle{
		lang:  tag,
		cache: newLRUCache(64),
		all:   make(map[string]*MessageObject),
		lock:  &sync.RWMutex{},
	}
	b.notFoundHandler = func(key string, writer io.Writer, ms ...map[string]any) {
		_, _ = writer.Write([]byte(""))
	}
	return b
}
