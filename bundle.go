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
	"strconv"
	"strings"
)

var (
	ErrUnsupportedFileType = errors.New("unsupported file type")
	ErrTargetIsRegular     = errors.New("target path is a regular file")
	ErrTargetIsDir         = errors.New("target path is a dir")
	ErrFileFormatError     = errors.New("file format unknown")
)

type Bundle struct {
	lang  language.Tag
	cache *LRU
	all   map[string]string
}

var (
	global *Bundle = New(SystemLanguage())
)

func (b *Bundle) flatten(prefix string, data any) {
	switch val := data.(type) {
	case string:
		b.all[prefix] = val
	case float64:
		s := ""
		if val == float64(int(val)) {
			s = strconv.FormatInt(int64(val), 10)
		} else {
			s = strconv.FormatFloat(val, 'f', 4, 10)
		}
		b.all[prefix] = s
	case []any:
		for i := 0; i < len(val); i++ {
			b.flatten(strings.Join([]string{prefix, fmt.Sprintf("[%d]", i)}, "."), val[i])
		}
	case map[string]any:
		for k, v := range val {
			pf := strings.Join([]string{prefix, k}, ".")
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
	ss := strings.Split(path, ".")
	if len(ss) <= 1 || len(fileTyp(ss[len(ss)-1])) == 0 {
		if len(ss) > 1 {
			typ = ss[len(ss)-1]
		}
		return fmt.Errorf("%w: %s", ErrUnsupportedFileType, typ)
	}
	typ = ss[len(ss)-1]
	return b.loadBytes(buf, getUnmarshalFunc(typ))
}

func (b *Bundle) loadBytes(bs []byte, unmarshaler UnmarshalFunc) error {
	var val any
	var err error

	//
	err = unmarshaler(bs, val)
	if err != nil {
		return err
	}
	switch v := val.(type) {
	case []any:
		for i := 0; i < len(v); i++ {
			b.flatten(fmt.Sprintf("[%d]", i), v[i])
		}
	case map[string]any:
		b.flatten("", v)
	default:
		return ErrFileFormatError
	}
	return nil
}

func (b *Bundle) LoadMap(m map[string]any, prefix string) {
	b.flatten(prefix, m)
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
			// skip it
			return nil
		}
		typ = ss[len(ss)-1]
		err2 = b.loadBytes(data, getUnmarshalFunc(typ))
		if err2 != nil {
			return fmt.Errorf("si18n: load: %w", err2)
		}
		return nil
	})
}

func (b *Bundle) LoadDir(dir string) error {
	root, err := os.Stat(dir)
	// check root is a dir
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("si18n: load: %w :%s", err, dir)
	}
	if err != nil {
		return fmt.Errorf("si18n: load: %w", err)
	}

	if !root.IsDir() {
		return fmt.Errorf("si18n: load: %w", ErrTargetIsRegular)
	}
	// find if there is a dir named b.lang.String()
	stat, err := os.Stat(filepath.Join(dir, b.lang.String()))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("si18n: load: %w", err)
	}
	if stat.IsDir() {
		return b.loadDir(filepath.Join(dir, b.lang.String()))
	}
	// just read file
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
	slices.DeleteFunc(files, func(entry os.DirEntry) bool {
		if entry.IsDir() {
			return false
		}
		return availableList[entry.Name()]
	})
	for i := 0; i < len(files); i++ {
		err := b.loadFile(files[i].Name())
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
		return b.loadFile(path)
	})
}

func New(tag language.Tag) *Bundle {
	return &Bundle{
		lang:  tag,
		cache: new(LRU),
		all:   make(map[string]string),
	}
}
