package main

import (
	"fmt"
	"math/rand"
)

func proba() float64 {
	return rand.Float64() * 100
}

func (this *Desk) AverageCards() int {
	var n int
	var m int
	for _, p := range this.seats {
		if !p.dead {
			n++
			m += len(p.cards)
		}
	}
	if n <= 0 {
		return 0
	}
	return m / n
}

// 返回mail.Msg
func (this *Desk) DeltaGo(bot *Player) string {
	if this.state.action == ACT_GO {
		ave := this.AverageCards()
		cn := len(bot.cards)
		// 随机索要
		if bot.HasCard(CARD_ASK) && proba() < 20 {
			out := this.DeltaUse(bot, CARD_ASK)
			if out != "" {
				return out
			}
		}
		// 1. 牌不够
		if cn < ave {
			if bot.HasCard(CARD_DISARM) {
				return "m"
			}
			if this.turn == bot.ai.safe+1 {
				return "m"
			}
			if s := this.DeltaTry(bot, []Card{CARD_PREDICT, CARD_PREDICT, CARD_SWAP}); s != "" {
				return s
			}
		}
		// 2.牌太多
		if cn > ave {
			if s := this.DeltaTry(bot, []Card{CARD_PREDICT, CARD_PREDICT, CARD_PASS, CARD_TURN, CARD_SHIRK,
				CARD_SHIRK2, CARD_SHUFFLE, CARD_BOTTOM}); s != "" {
				return s
			}
		}
		// 3. 牌正好
		if s := this.DeltaTry(bot, []Card{CARD_PREDICT, CARD_PREDICT, CARD_PASS, CARD_TURN, CARD_SHIRK,
			CARD_SHIRK2, CARD_SHUFFLE, CARD_BOTTOM}); s != "" {
			return s
		}
		return "m"
	} else if this.state.action == ACT_DISARM {
		next := this.seats[this.getNext()-1]
		if len(next.cards) == 0 {
			return "1"
		}
		if proba() < 30 {
			return "r"
		}
		if len(this.stack) >= 15 {
			return "5"
		}
		return "0"
	} else if this.state.action == ACT_DILIVER {
		priorities := []Card{CARD_SHIRK2, CARD_SHIRK, CARD_PERSP, CARD_PREDICT, CARD_TURN,
			CARD_BOTTOM, CARD_PASS, CARD_SHUFFLE, CARD_DISARM, CARD_SWAP, CARD_ASK}
		for _, c := range priorities {
			if bot.HasCard(c) {
				return c.Shortcut()
			}
		}
	}
	return "c 我傻了"
}

func (this *Desk) DeltaTry(bot *Player, cards []Card) string {
	for _, c := range cards {
		if bot.HasCard(c) {
			return this.DeltaUse(bot, c)
		}
	}
	return ""
}

func (this *Desk) DeltaUse(bot *Player, card Card) string {
	switch card {
	case CARD_PREDICT:
		i := this.findCard(CARD_BOMB)
		if i > 0 {
			bot.ai.safe = this.turn
		}
		return KEY_PREDICT
	case CARD_PERSP:
		i := this.findCard(CARD_BOMB)
		if i > 0 {
			bot.ai.safe = this.turn
		}
		return KEY_PERSP
	case CARD_ASK:
		p := this.PoorPlayer(bot.Id)
		if p == nil {
			return ""
		}
		return fmt.Sprintf("%s %d", KEY_ASK, p.no)
	case CARD_SWAP:
		p := this.RichPlayer(bot.Id)
		return fmt.Sprintf("%s %d", KEY_SWAP, p.no)
	case CARD_SHUFFLE:
		return KEY_SHUFFLE
	case CARD_PASS:
		return KEY_PASS
	case CARD_TURN:
		return KEY_TURN
	case CARD_SHIRK:
		p := this.PoorPlayer(bot.Id)
		return fmt.Sprintf("%s %d", KEY_SHIRK, p.no)
	case CARD_SHIRK2:
		p := this.PoorPlayer(bot.Id)
		return fmt.Sprintf("%s %d", KEY_SHIRK2, p.no)
	case CARD_BOTTOM:
		return KEY_BOTTOM
	}
	return "c 整不会了"
}

func (this *Desk) PoorPlayer(exclude int) *Player {
	var min = 10
	var minp *Player
	for _, p := range this.seats {
		if !p.dead && p.Id != exclude && len(p.cards) > 0 {
			if len(p.cards) < min {
				min = len(p.cards)
				minp = p
			}
		}
	}
	return minp
}

func (this *Desk) RichPlayer(exclude int) *Player {
	var max = 0
	var maxp *Player
	for _, p := range this.seats {
		if !p.dead && p.Id != exclude {
			if len(p.cards) > max {
				max = len(p.cards)
				maxp = p
			}
		}
	}
	return maxp
}
