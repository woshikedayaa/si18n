package si18n

import (
	"embed"
	"errors"
	"fmt"
	"golang.org/x/text/language"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var (
	ErrEmpty                       = errors.New("empty")
	ErrUnsupportedFileType         = errors.New("unsupported file type")
	ErrTargetIsRegular             = errors.New("target path is a regular file")
	ErrTargetIsDir                 = errors.New("target path is a dir")
	ErrIncorrectBytesUnmarshalFunc = errors.New("incorrect bytes unmarshaler")
	ErrKeyType                     = errors.New("key type incorrect,except: float int string")
)

type Bundle struct {
	lang  language.Tag
	cache *LRUCache
	all   map[string]string
}

var (
	global = New(SystemLanguage())
)

func LoadFile(path string) error {
	return global.LoadFile(path)
}

func LoadFS(fs *embed.FS) error {
	return global.LoadFs(fs)
}

func LoadMap(m map[string]any, prefix string) {
	global.LoadMap(m, prefix)
}

func LoadDir(dir string) error {
	return global.loadDir(dir)
}

func GetAllTR() map[string]string {
	return global.GetAllTR()
}

func TR(key string, ms ...map[string]any) string {
	return global.TR(key, ms...)
}

func Language() language.Tag {
	return global.Language()
}

func (b *Bundle) flatten(prefix string, data any) {
	switch val := data.(type) {
	case string:
		b.all[prefix] = val
	case float64, int:
		s := fmt.Sprint(val)
		b.all[prefix] = s
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

func (b *Bundle) LoadFile(path string) error {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("si18n: load: %w:%s", err, path)
	}
	if stat.IsDir() {
		return fmt.Errorf("si18n: load: %w:%s", ErrTargetIsDir, path)
	}
	err = b.loadFile(path)
	if err != nil {
		return fmt.Errorf("si18n: load: %w", err)
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
	typ := ""
	ss := strings.Split(filepath.Base(path), ".")
	if len(ss) <= 1 || len(fileTyp(ss[len(ss)-1])) == 0 {
		if len(ss) > 1 {
			typ = ss[len(ss)-1]
		}
		return fmt.Errorf("%w: %s", ErrUnsupportedFileType, typ)
	}
	typ = ss[len(ss)-1]
	return b.loadBytes(buf, GetUnmarshalFunc(typ))
}

func (b *Bundle) loadBytes(bs []byte, unmarshaler UnmarshalFunc) error {
	if len(bs) == 0 {
		return ErrEmpty
	}

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

func (b *Bundle) LoadString(s string, umf UnmarshalFunc) error {
	return b.LoadBytes([]byte(s), umf)
}

func (b *Bundle) LoadBytes(bs []byte, umf UnmarshalFunc) error {
	err := b.loadBytes(bs, umf)
	if err != nil {
		return fmt.Errorf("si18n: load: %w", err)
	}
	return nil
}

func (b *Bundle) LoadMap(m map[string]any, prefix string) {
	for k, v := range m {
		if len(prefix) == 0 {
			b.flatten(k, v)
		} else {
			b.flatten(strings.Join([]string{prefix, k}, "."), v)
		}
	}
}

func (b *Bundle) LoadFs(f *embed.FS) error {
	var targetDir string
	_, err := f.ReadDir(b.lang.String())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("si18n: load: %w", err)
	}
	if errors.Is(err, os.ErrNotExist) {
		targetDir = "."
	} else {
		targetDir = b.lang.String()
	}
	return fs.WalkDir(f, targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err2 := f.ReadFile(path)
		if err2 != nil {
			return err2
		}
		typ := ""
		ss := strings.Split(path, ".")
		if len(ss) <= 1 || len(fileTyp(ss[len(ss)-1])) == 0 {
			if len(ss) > 1 {
				typ = ss[len(ss)-1]
			}
			// skip
			return nil
		}
		typ = ss[len(ss)-1]
		err2 = b.loadBytes(data, GetUnmarshalFunc(typ))
		if err2 != nil {
			return fmt.Errorf("si18n: load: %w", err2)
		}
		return nil
	})
}

func (b *Bundle) LoadDir(dir string) error {
	rootStat, err := os.Stat(dir)
	// check rootStat is a dir
	if rootStat != nil && !rootStat.IsDir() {
		return fmt.Errorf("si18n: load: %w: %s", ErrTargetIsRegular, dir)
	}

	if err != nil {
		return fmt.Errorf("si18n: load: %w: %s", err, dir)
	}

	// find if there is a dir named b.lang.String()
	stat, err := os.Stat(filepath.Join(dir, b.lang.String()))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("si18n: load: %w", err)
	}
	if stat != nil && stat.IsDir() {
		err = b.loadDir(filepath.Join(dir, b.lang.String()))
		if err != nil {
			return fmt.Errorf("si18n: load: %w", err)
		}
		return nil
	}
	// just read files
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("si18n: load: %w", err)
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
			return fmt.Errorf("si18n: load: %w", err)
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

func (b *Bundle) TR(key string, arg ...map[string]any) string {
	che, ok := b.cache.Get(key)
	if ok {
		return che
	}
	res, ok := b.all[key]
	if ok {
		b.cache.Put(key, res)
	}
	return res
}

func (b *Bundle) GetAllTR() map[string]string {
	return b.all
}

func (b *Bundle) Language() language.Tag {
	return b.lang
}

func New(tag language.Tag) *Bundle {
	return &Bundle{
		lang:  tag,
		cache: newLRUCache(64),
		all:   make(map[string]string),
	}
}
