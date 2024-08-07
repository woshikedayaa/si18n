# si18n
si18n (simple-i18n) is a lightweight internationalization library designed to simplify multilingual support in applications. It offers an easy-to-use API for handling translations and managing language packs.  

[zh-cn](https://github.com/woshikedayaa/si18n/blob/main/docs/README.zh-CN.md)
# Quick-start

## basic usage
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
## use go-template
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

## order a special language

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

# Contribution
We welcome issues and pull requests to improve si18n.

# License
si18n is licensed under the MIT License. For more details, please refer to the [LICENSE](https://github.com/woshikedayaa/si18n/blob/main/LICENSE) file.
