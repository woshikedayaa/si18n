package si18n

import (
	"container/list"
	"encoding/json"
	"github.com/BurntSushi/toml"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func fileTyp(typ string) string {
	switch typ {
	case "yaml", "yml", "toml", "json":
		return typ
	default:
		return ""
	}
}

func GetUnmarshalFunc(typ string) UnmarshalFunc {
	var umf UnmarshalFunc = nil
	// choose UnmarshalFunc
	switch typ {
	case "yaml", "yml":
		umf = yaml.Unmarshal
	case "toml":
		umf = toml.Unmarshal
	case "json", "json5":
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

type LRUCache struct {
	capacity  int
	list      *list.List
	keyToNode map[string]*list.Element
}

type entry struct {
	key string
	val string
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

func (lru *LRUCache) Get(key string) (string, bool) {
	if lru.Len() == 0 {
		return "", false
	}
	node, ok := lru.keyToNode[key]
	if !ok {
		return "", false
	}
	lru.list.MoveToFront(node)
	return node.Value.(entry).val, true
}

func (lru *LRUCache) Put(key, val string) {
	if node, ok := lru.keyToNode[key]; ok {
		// update
		node.Value = entry{key, val}
		lru.list.MoveToFront(node)
		return
	}
	lru.keyToNode[key] = lru.list.PushFront(entry{key, val})
	if lru.Len() > lru.Cap() {
		delete(lru.keyToNode, lru.list.Remove(lru.list.Back()).(entry).key)
	}
}
