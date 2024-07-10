# si18n
si18n（simple-i18n）是一个轻量级的国际化库，旨在简化应用程序的多语言支持。它提供了简便的 API 来处理语言翻译和语言包管理。

# 快速开始

## 基础使用
```go
package main

import (
	"fmt"
	"github.com/woshikedayaa/si18n"
)

func main() {
	// test data
	data := `
a: 
  b:
	1: b1
	c: 
	  1: c1
	  2: c2
`
	// init with system language
	bundle := si18n.New(si18n.SystemLanguage())
	err := bundle.LoadFromBytes([]byte(data), "yaml")
	if err != nil {
		panic(err)
	}
	fmt.Println(bundle.MustTr("a.b.1"))   // b1
	fmt.Println(bundle.MustTr("a.b.c.1")) // c1
	fmt.Println(bundle.MustTr("a.b.c.2")) // c2
}
```
## 使用模板
```go
package main

import (
    "fmt"
    "github.com/woshikedayaa/si18n"
)

func main() {
	// test data
	data := `
a:
  b:
    1: "{{.data}}" # go-template
`   
	// init with system language
	bundle := si18n.New(si18n.SystemLanguage())
	err := bundle.LoadFromBytes([]byte(data), "yaml")
	if err != nil {
		panic(err)
	}
	fmt.Println(bundle.MustTr("a.b.1", map[string]any{
		"data": "test",
	})) // test
}
```

## 指定语言

```go
package main

import (
	"fmt"
	"github.com/woshikedayaa/si18n"
	"golang.org/x/text/language"
)

func main() {
	// test data
	data := `
a: 
  b:
	1: b1
	c: 
	  1: c1
	  2: c2
`
	// init 
	bundle := si18n.New(language.English)
	err := bundle.LoadFromBytes([]byte(data), "yaml")
	if err != nil {
		panic(err)
	}
	fmt.Println(bundle.MustTr("a.b.1"))   // b1
	fmt.Println(bundle.MustTr("a.b.c.1")) // c1
	fmt.Println(bundle.MustTr("a.b.c.2")) // c2
}
```

# 贡献
欢迎提交 Issue 和 Pull Request 以改进 si18n。

# 许可证
si18n 采用 MIT 许可证。详细信息请参见 LICENSE 文件。