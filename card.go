package main

import (
	"fmt"
	"strings"
)

type Card int

const (
	CARD_PREDICT = 1  // 预言
	CARD_PERSP   = 2  // 透视
	CARD_ASK     = 3  // 索要
	CARD_SWAP    = 4  // 交换
	CARD_SHUFFLE = 5  // 洗牌
	CARD_PASS    = 6  // 跳过
	CARD_TURN    = 7  // 转向
	CARD_SHIRK   = 8  // 甩锅
	CARD_SHIRK2  = 9  // 双甩
	CARD_BOTTOM  = 10 // 抽底
	CARD_DISARM  = 11 // 拆除
	CARD_BOMB    = 12 // 炸弹
)

const (
	KEY_DRAW    = "m" // 摸牌
	KEY_PREDICT = "y" // 预言
	KEY_PERSP   = "t" // 透视
	KEY_ASK     = "a" // 索要
	KEY_SWAP    = "e" // 交换
	KEY_SHUFFLE = "x" // 洗牌
	KEY_PASS    = "j" // 跳过
	KEY_TURN    = "d" // 转向
	KEY_SHIRK   = "s" // 甩锅
	KEY_SHIRK2  = "f" // 双甩
	KEY_BOTTOM  = "b" // 抽底
	KEY_DISARM  = ""  // 拆除
	KEY_BOMB    = ""  // 炸弹
)

func (c Card) String() string {
	names := []string{
		"预言", "透视", "索要", "交换", "洗牌", "跳过", "转向", "甩锅", "双甩", "抽底", "拆除", "炸弹",
	}
	if int(c) > len(names) {
		return fmt.Sprintf("%d", c)
	}
	return names[c-1]
}

func (c Card) Shortcut() string {
	keys := []string{
		KEY_PREDICT,
		KEY_PERSP,
		KEY_ASK,
		KEY_SWAP,
		KEY_SHUFFLE,
		KEY_PASS,
		KEY_TURN,
		KEY_SHIRK,
		KEY_SHIRK2,
		KEY_BOTTOM,
		KEY_DISARM,
		KEY_BOMB,
	}
	if int(c) >= len(keys) {
		return "?"
	}
	return keys[c-1]
}

func joinCards(cards []Card) string {
	if len(cards) == 0 {
		return "[]"
	}
	var out string
	for i, c := range cards {
		out += fmt.Sprintf("%s", c.String())
		if i < len(cards)-1 {
			out += ", "
		}
	}
	return out
}

func groupCards(cards []Card) string {
	var m = make(map[Card]int)
	for _, c := range cards {
		m[c]++
	}
	var out []string
	for c, n := range m {
		if n <= 1 {
			out = append(out, fmt.Sprintf("%s%s", c.String(), c.Shortcut()))
		} else {
			hans := []string{"两", "三", "四", "五", "六", "七", "八"}
			out = append(out, fmt.Sprintf("%s张%s%s", hans[n-2], c.String(), c.Shortcut()))
		}
	}
	return strings.Join(out, ", ")
}
