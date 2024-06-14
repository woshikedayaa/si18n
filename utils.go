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

func getUnmarshalFunc(typ string) UnmarshalFunc {
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

type LRU struct {
	list list.List
	m    map[string]string
}
