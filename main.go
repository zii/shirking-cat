package main

import (
	"log"
	"math/rand"
	"net"
	"runtime/debug"
	"time"
)

type Server struct {
	handler Handler
	Addr    string
}

type Handler interface {
	Handle(c net.Conn)
}

func (s *Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	return s.Serve(l)
}

func (s *Server) Serve(l net.Listener) error {
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go s.handle(c, s.handler)
	}
}

func (s *Server) handle(c net.Conn, h Handler) {
	defer c.Close()

	defer func() {
		if r := recover(); r != nil {
			log.Println("recover:", r)
			log.Println(string(debug.Stack()))
		}
	}()

	n := NewConn(c)
	h.Handle(n)
}

func raise(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	rand.Seed(time.Now().Unix())
	h := NewGameHandler()
	s := &Server{
		handler: h,
		Addr:    ":5555",
	}
	log.Println("server listen on:", s.Addr)
	err := s.ListenAndServe()
	raise(err)
}
