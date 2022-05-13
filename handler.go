package main

import (
	"errors"
	"log"
	"net"
	"regexp"
)

type GameHandler struct {
}

func NewGameHandler() *GameHandler {
	h := &GameHandler{}
	return h
}

func (h *GameHandler) Handle(raw net.Conn) {
	c := NewConn(raw)
	log.Println("new connection:", c.RemoteAddr())
	user, err := h.Login(c)
	if err != nil {
		return
	}
	log.Println("user login:", user.Id, user.Name)
	if user.Desk == nil {
		c.Println("登录成功! 快速开始请输入k: ")
	} else {
		c.Println("登录成功! 重新加入牌桌: ", user.Desk.Id)
	}
	for {
		msg, err := c.ReadLine()
		if err != nil {
			break
		}
		if user.Desk != nil {
			user.Desk.Post(user, msg)
		} else {
			h.handleIdleMsg(user, msg)
		}
	}
	c.Close()
	if user.Desk != nil {
		user.Desk.Post(user, "off")
		user.Desk = nil
	}
	log.Println("lost connection:", c.RemoteAddr())
}

func (h *GameHandler) Login(c *Conn) (*User, error) {
	for i := 0; i < 3; i++ {
		c.Println("请输入用户名: ")
		s, err := c.ReadLine()
		if err != nil {
			return nil, err
		}
		if !h.ValidUsername(s) {
			c.Println("格式错误, 用户名应为2-10个小写字母.")
			continue
		}
		user := NewUser(c, s)
		return user, nil
	}
	c.Print("重试次数过多")
	return nil, errors.New("x")
}

func (h *GameHandler) ValidUsername(s string) bool {
	r := regexp.MustCompile(`[a-z]{2,10}`)
	return r.MatchString(s)
}

func (h *GameHandler) handleIdleMsg(user *User, msg string) {
	if msg == "k" {
		d := NewDesk()
		go h.processDesk(d)
		d.Post(user, "join")
	} else {
		user.Println("无效的命令, 快速开始请输入k:")
	}
}

func (h *GameHandler) processDesk(d *Desk) {
	go d.AddBots()
	ok := d.WaitPlayers()
	if !ok {
		d.Sendall("等待超时, 牌桌已解散")
		d.Stop()
		return
	}
	d.Play()
	d.Sendall("游戏结束, 按k继续匹配玩家")
	d.Stop()
}
