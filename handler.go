package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
)

type GameHandler struct {
	desks sync.Map
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
		c.Println("登录成功! 快速开始a, 查看l, 退出q")
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

func (h *GameHandler) findDesk() *Desk {
	var out *Desk
	h.desks.Range(func(key, value any) bool {
		d := value.(*Desk)
		if len(d.players) < MaxSeats {
			out = d
			return false
		}
		return true
	})
	return out
}

func ParseMsg(msg string) (string, Argument) {
	a := strings.SplitN(msg, " ", 2)
	if len(a) <= 0 {
		return "", ""
	}
	if len(a) == 1 {
		return a[0], ""
	}
	return a[0], Argument(a[1])
}

func (h *GameHandler) handleIdleMsg(user *User, msg string) {
	cmd, arg := ParseMsg(msg)
	if cmd == "a" {
		d := h.findDesk()
		if d == nil {
			d = NewDesk()
			h.desks.Store(d.Id, d)
			go h.processDesk(d)
		}
		d.Post(user, "join")
	} else if cmd == "l" {
		buf := strings.Builder{}
		buf.WriteString("加入游戏: j <牌桌ID>\n")
		var n int
		h.desks.Range(func(key, value any) bool {
			d := value.(*Desk)
			buf.WriteString(fmt.Sprintf("#%d: %d人\n", d.Id, len(d.players)))
			n++
			return true
		})
		if n == 0 {
			user.Println("当前没有牌桌.")
		} else {
			user.Printf(buf.String())
		}
	} else if cmd == "j" {
		id := arg.Int()
		h.cmdJoin(user, id)
	} else if cmd == "q" {
		user.Close()
	} else {
		user.Println("无效的命令, 快速开始a, 查看l, 退出q")
	}
}

func (h *GameHandler) cmdJoin(user *User, id int) {
	if id == 0 {
		user.Println("参数错误, 必须是数字")
		return
	}
	v, ok := h.desks.Load(id)
	if !ok {
		user.Println("无此牌桌")
		return
	}
	d := v.(*Desk)
	if len(d.players) >= MaxSeats {
		user.Println("牌桌已坐满.")
		return
	}
	d.Post(user, "join")
}

func (h *GameHandler) processDesk(d *Desk) {
	defer func() {
		d.Stop()
		h.desks.Delete(d.Id)
	}()
	//go d.AddBots()
	ok := d.WaitPlayers()
	if !ok {
		d.Sendall("等待超时, 牌桌已解散")
		return
	}
	d.Play()
	d.Sendall("游戏结束, 按a继续匹配玩家")
}
