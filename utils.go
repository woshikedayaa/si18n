package si18n

import (
	"container/list"
	"encoding/json"
	"errors"
	"github.com/BurntSushi/toml"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func parseFormat(typ string) (string, error) {
	switch typ {
	case "yaml", "yml", "toml", "json":
		return typ, nil
	default:
		return "", ErrUnsupportedFormat
	}
}

func getUnmarshalFunc(typ string) UnmarshalFunc {
	var umf UnmarshalFunc = nil
	// choose UnmarshalFunc
	switch typ {
	case "yaml", "yml":
		umf = yaml.Unmarshal
	case "toml":
		umf = toml.Unmarshal
	case "json":
		umf = json.Unmarshal
	}
	return umf
}

// SystemLanguage return the system default language
func SystemLanguage() language.Tag {
	return language.MustParse(getSystemLocales())
}

// getSystemLocales return the system locales
// if $LANG(unix like system) not defined , return en_US
// if run command(windows) error , return en_US
func getSystemLocales() string {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("powershell", "Get-Culture | select -exp Name")
		output, err := cmd.Output()
		if err == nil {
			return strings.Trim(string(output), "\r\n")
		} else {
			return language.English.String()
		}
	default:
		lang, ok := os.LookupEnv("LANG")
		if !ok {
			return language.English.String()
		}
		return strings.Split(lang, ".")[0]
	}
}

func DefaultLanguage() language.Tag {
	return SystemLanguage()
}

type LRUCache struct {
	capacity  int
	list      *list.List
	keyToNode map[string]*list.Element
}

type entry struct {
	key string
	val *MessageObject
}

func newLRUCache(capacity int) *LRUCache {
	lru := &LRUCache{
		capacity:  capacity,
		list:      list.New(),
		keyToNode: make(map[string]*list.Element, capacity),
	}
	return lru
}
func (lru *LRUCache) Len() int {
	return lru.list.Len()
}

func (lru *LRUCache) Cap() int {
	return lru.capacity
}

func (lru *LRUCache) Get(key string) (*MessageObject, bool) {
	if lru.Len() == 0 {
		return nil, false
	}
	node, ok := lru.keyToNode[key]
	if !ok {
		return nil, false
	}
	lru.list.MoveToFront(node)
	return node.Value.(entry).val, true
}

func (lru *LRUCache) Put(key string, val *MessageObject) {
	if node, ok := lru.keyToNode[key]; ok {
		// update
		node.Value = entry{key, val}
		lru.list.MoveToFront(node)
		return
	}
	lru.keyToNode[key] = lru.list.PushFront(entry{key, val})
	for lru.Len() > lru.Cap() {
		delete(lru.keyToNode, lru.list.Remove(lru.list.Back()).(entry).key)
	}
}

func (lru *LRUCache) Remove(key string) bool {
	if lru.Len() == 0 {
		return false
	}
	var (
		node *list.Element
		ok   bool
	)
	if node, ok = lru.keyToNode[key]; !ok {
		return false
	}
	delete(lru.keyToNode, lru.list.Remove(node).(entry).key)
	return true
}

func (lru *LRUCache) Resize(capacity int) {
	if capacity <= 0 {
		panic(errors.New("LRUCache capacity must larger than zero"))
	}
	for lru.Len() > capacity {
		delete(lru.keyToNode, lru.list.Remove(lru.list.Back()).(entry).key)
	}
	lru.capacity = capacity
}

func mergeMap(ms ...map[string]any) map[string]any {
	if len(ms) == 0 {
		return nil
	}
	start := 0
	for ; ms[start] == nil; start++ {
	}
	for i := start + 1; i < len(ms); i++ {
		if ms[i] == nil {
			continue
		}
		for k, v := range ms[i] {
			ms[start][k] = v
		}
	}
	return ms[start]
}

func errorWarp(err error) error {
	return fmt.Errorf("%s: %w", "si18n", err)
}
