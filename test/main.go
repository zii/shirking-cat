package main

import (
	"fmt"
	"regexp"
)

func test1() {
	r := regexp.MustCompile(`[a-z]{2,8}`)
	fmt.Println(r.MatchString("se"))
}

func test2() {
	d := make(chan string)
	m := <-d
	go func() {
		d <- "hi"
	}()
	fmt.Println("m:", m)
}

func main() {
	test2()
}
