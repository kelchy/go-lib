package main

import (
	"fmt"
	"github.com/kelchy/go-lib/common"
)

func main() {
	fmt.Println(common.SliceHasString([]string{"abc", "def"}, "def"))
	fmt.Println(common.SliceHasString([]string{"abc", "def"}, "efg"))
}
