package main

import "sync/atomic"

type User struct {
	*Conn
	Id   int // user id
	Name string
	Desk *Desk
}

var _user_id int64

func NewUser(c *Conn, username string) *User {
	atomic.AddInt64(&_user_id, 1)
	u := &User{
		Conn: c,
		Id:   int(_user_id),
		Name: username,
	}
	return u
}
