package main

import (
	"fmt"
)

func main() {
	reg := "ab|cd"
	post := re2post(reg)
	start := post2nfa(post)
	test := "ab"
	res := match(start, test)
	fmt.Println(res)
}
