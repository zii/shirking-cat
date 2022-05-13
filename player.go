package main

type Player struct {
	*User
	bot   bool   // 是不是机器人
	cards []Card // 手牌
	dead  bool   // 被炸死
	off   bool   // 离开牌桌
	no    int    // 座位号 1-5
}

func NewPlayer(user *User, no int) *Player {
	p := &Player{
		User: user,
		off:  false,
		no:   no,
		bot:  user.Conn == nil,
	}
	return p
}

func (p *Player) Println(a ...any) {
	if p.User == nil {
		return
	}
	if p.bot {
		return
	}
	if p.off {
		return
	}
	if p.User.Conn == nil {
		return
	}
	if p.Desk != nil {
		p.User.Printf("[desk%d] ", p.Desk.Id)
	}
	p.User.Println(a...)
}

func (p *Player) Printf(format string, a ...any) {
	if p.User == nil {
		return
	}
	if p.bot {
		return
	}
	if p.off {
		return
	}
	if p.User.Conn == nil {
		return
	}
	if p.Desk != nil {
		p.User.Printf("[desk%d] ", p.Desk.Id)
	}
	p.User.Printf(format, a...)
}

func (p *Player) HasDisarm() bool {
	for _, c := range p.cards {
		if c == CARD_DISARM {
			return true
		}
	}
	return false
}

func (p *Player) PutCard(card Card) {
	p.cards = append(p.cards, card)
}

func (p *Player) RemoveCard(card Card) bool {
	for i, c := range p.cards {
		if c == card {
			out := p.cards[:i]
			if i < len(p.cards)-1 {
				out = append(out, p.cards[i+1:]...)
			}
			p.cards = out
			return true
		}
	}
	return false
}
