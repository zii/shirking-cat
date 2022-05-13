package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	d := NewDesk()
	go func() {
		time.Sleep(1 * time.Second)
		d.Post(nil, "hi")
	}()
	m := d.Read(time.Second * 3)
	fmt.Println("m:", m)
}

func Test2(t *testing.T) {
	rand.Seed(time.Now().Unix())
	d := NewDesk()
	d.stack = []Card{1, 2, 3}
	d.PutCard(4, -2)
	fmt.Println("stack:", d.stack)
}
