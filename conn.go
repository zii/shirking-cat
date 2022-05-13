package main

import (
	"bufio"
	"fmt"
	"net"
	"sync/atomic"
)

type Conn struct {
	net.Conn
	Id     int
	reader *bufio.Reader
	Closed bool
}

var _id int64

func NewConn(c net.Conn) *Conn {
	atomic.AddInt64(&_id, 1)
	n := &Conn{
		Conn:   c,
		Id:     int(_id),
		reader: bufio.NewReader(c),
	}
	return n
}

func (c *Conn) Close() error {
	if c.Closed {
		return nil
	}
	c.Closed = true
	return c.Close()
}

func (c *Conn) ReadLine() (string, error) {
	l, _, err := c.reader.ReadLine()
	return string(l), err
}

func (c *Conn) Print(a ...any) {
	if c.Closed {
		return
	}
	fmt.Fprint(c, a...)
}

func (c *Conn) Printf(format string, a ...any) {
	if c.Closed {
		return
	}
	fmt.Fprintf(c, format, a...)
}

func (c *Conn) Println(a ...any) {
	if c.Closed {
		return
	}
	fmt.Fprintln(c, a...)
}
