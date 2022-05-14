package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const MaxSeats = 5
const TurnTimeout = 10 // 回合超时时间

// 行动模式
const (
	ACT_GO      = 1 // 等待出摸牌
	ACT_DISARM  = 2 // 等待拆除炸弹
	ACT_DILIVER = 3 // 等待交牌
)

type Argument string

func (a Argument) Int() int {
	i, _ := strconv.Atoi(string(a))
	return i
}

type Mail struct {
	User *User
	Msg  string // join:加入 off:离开
}

func (m *Mail) Command() (string, Argument) {
	a := strings.SplitN(m.Msg, " ", 2)
	if len(a) <= 0 {
		return "", ""
	}
	if len(a) == 1 {
		return a[0], ""
	}
	return a[0], Argument(a[1])
}

// 牌型: 预言1 透视2 索要3 交换4 洗牌5 跳过6 转向7 甩锅8 双甩9 抽底10 拆除11 炸弹12
// 牌数: 预言4 透视4 索要4 交换3 洗牌4 跳过8 转向5 甩锅5 双甩3 抽底3 拆除6 炸弹4 共53张
// 快捷键: 预言y 透视t 索要a 交换e 洗牌x 跳过j 转向d 甩锅s 双甩f 抽底b 拆除r 炸弹*

type Desk struct {
	Id       int
	mailbox  chan *Mail
	players  map[int]*Player
	seats    [MaxSeats]*Player
	stopch   chan struct{}
	turn     int // 第几回合
	bot_id   int
	namebook []string

	state struct {
		direct  int // 方向 1从小到大 2从大到小
		current int // 当前出牌人座位号
		action  int // 当前行动类型: 1等待出摸牌 2等待拆除 3等待交牌
		from_id int // 攻击者ID
		combo   int // 剩余行动次数, 连击点数
	}
	stack     []Card // 牌堆
	expire_at int
}

var _desk_id int64

func NewDesk() *Desk {
	atomic.AddInt64(&_desk_id, 1)
	d := &Desk{
		Id:       int(_desk_id),
		stopch:   make(chan struct{}, 1),
		mailbox:  make(chan *Mail, 1),
		players:  make(map[int]*Player, MaxSeats),
		namebook: []string{"后端", "美工", "前端", "产品", "运营", "人事", "市场", "测试", "领导"},
	}
	d.state.direct = 1
	rand.Shuffle(len(d.namebook), func(i, j int) {
		d.namebook[i], d.namebook[j] = d.namebook[j], d.namebook[i]
	})
	return d
}

func (this *Desk) Stoped() bool {
	select {
	case <-this.stopch:
		return true
	default:
		return false
	}
}

func (this *Desk) Post(user *User, msg string) {
	m := &Mail{
		User: user,
		Msg:  msg,
	}
	if this.Stoped() {
		return
	}
	select {
	case <-this.stopch:
		break
	case this.mailbox <- m:
		break
	}
}

// 超时返回nil
func (this *Desk) Read(timeout time.Duration) *Mail {
	select {
	case m := <-this.mailbox:
		return m
	case <-time.After(timeout):
		return nil
	}
}

// 返回0表坐满
func (this *Desk) findSeat() int {
	no := len(this.players) + 1
	if no > MaxSeats {
		return 0
	}
	return no
}

// 返回false表示已坐满
func (this *Desk) AddPlayer(user *User) (*Player, bool) {
	if p, ok := this.players[user.Id]; ok {
		p.User = user
		p.off = false
		p.bot = user.Conn == nil
		return p, true
	}

	no := this.findSeat()
	if no <= 0 {
		return nil, false
	}
	p := NewPlayer(user, no)
	this.seats[no-1] = p
	this.players[p.Id] = p
	p.Desk = this
	return p, true
}

func (this *Desk) Sendall(msg ...any) {
	for _, p := range this.players {
		if !p.off {
			p.Println(msg...)
		}
	}
}

func (this *Desk) Sendallf(format string, a ...any) {
	for _, p := range this.players {
		if !p.off {
			p.Printf(format, a...)
		}
	}
}

func (this *Desk) Sendothers(user_id int, msg ...any) {
	for _, p := range this.players {
		if p.Id != user_id && !p.off {
			p.Println(msg...)
		}
	}
}

func (this *Desk) Sendothersf(user_id int, format string, a ...any) {
	for _, p := range this.players {
		if p.Id != user_id && !p.off {
			p.Printf(format, a...)
		}
	}
}

func (this *Desk) Stop() {
	if this.Stoped() {
		return
	}
	close(this.stopch)
	for i, p := range this.seats {
		if p != nil {
			this.seats[i] = nil
			this.RemovePlayer(p.User)
		}
	}
	close(this.mailbox)
}

func (this *Desk) RemovePlayer(user *User) bool {
	p, ok := this.players[user.Id]
	if !ok {
		return false
	}
	p.off = true
	if user.Desk != nil && user.Desk.Id == this.Id {
		user.Desk = nil
	}
	//delete(this.players, user.Id)
	//this.seats[p.no-1] = nil
	return true
}

func (this *Desk) AddBot() {
	this.bot_id++

	name := this.namebook[(this.bot_id-1)%len(this.namebook)]
	user := NewUser(nil, name)
	this.Post(user, "join")
}

func (this *Desk) AddBots() {
	for !this.Stoped() {
		for _, p := range this.seats {
			if p == nil {
				this.AddBot()
				//break
			}
		}
		time.Sleep(time.Duration(rand.Intn(5)+1) * time.Second)
	}
}

// wait for filling all the seats
func (this *Desk) WaitPlayers() bool {
	for len(this.players) < MaxSeats {
		m := this.Read(10 * time.Second)
		if m == nil {
			return false
		}
		if m.User == nil {
			continue
		}
		user := m.User
		player := this.players[user.Id]
		if m.Msg == "join" {
			p, ok := this.AddPlayer(m.User)
			if !ok {
				p.Println("已坐满, 请重试")
			} else {
				p.Println("成功加入牌桌, 请等待玩家. q:退出等待, l:查看当前状态")
				this.Sendothers(user.Id, "来人了:", p.Name)
			}
		} else if player != nil && m.Msg == "q" {
			this.RemovePlayer(user)
			player.Println("您已退出牌桌:", this.Id)
		} else if player != nil && m.Msg == "l" {
			player.Printf("牌桌ID: %d, 人数: %d\n", this.Id, len(this.players))
		} else if player != nil {
			player.Println("无效的命令. q:退出等待, l:查看当前状态")
		} else {
			user.Println("?")
		}
	}
	return true
}

func (this *Desk) Alives() int {
	var n int
	for _, p := range this.seats {
		if p != nil && !p.dead {
			n++
		}
	}
	return n
}

func (this *Desk) Deads() int {
	var n int
	for _, p := range this.seats {
		if p != nil && p.dead {
			n++
		}
	}
	return n
}

// 运行一局游戏
func (this *Desk) Play() {
	this.Sendall("游戏开始!")
	this.Sendall("提示: 摸一张牌(m), 打出一张牌(参考帮助), 聊天(c <文字>), 查看状态(l), 帮助(?)")
	this.init()
	for _, p := range this.seats {
		this.SendStatus(p)
	}
	for this.Alives() > 1 {
		this.OneTurn()
	}
	for _, p := range this.seats {
		if !p.dead {
			this.Sendallf("恭喜🎉%s坚持到了最后!\n", p.Name)
			break
		}
	}
}

func (this *Desk) init() {
	this.stack = []Card{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 5, 5, 5, 5, 6, 6, 6, 6, 6, 6, 6, 6,
		7, 7, 7, 7, 7, 8, 8, 8, 8, 8, 9, 9, 9, 10, 10, 10}
	rand.Shuffle(len(this.stack), func(i, j int) {
		this.stack[i], this.stack[j] = this.stack[j], this.stack[i]
	})
	// 发牌
	for _, p := range this.seats {
		for i := 0; i < 4; i++ {
			card := this.PopCard()
			p.cards = append(p.cards, card)
		}
		p.cards = append(p.cards, 11)
	}
	this.stack = append(this.stack, 11, 12, 12, 12, 12)
	rand.Shuffle(len(this.stack), func(i, j int) {
		this.stack[i], this.stack[j] = this.stack[j], this.stack[i]
	})
	this.state.current = rand.Intn(5) + 1
	this.state.combo = 0
	this.state.action = ACT_GO
	this.expire_at = int(time.Now().Unix()) + TurnTimeout
}

func (this *Desk) PopCard() Card {
	if len(this.stack) == 0 {
		return 0
	}
	out := this.stack[0]
	this.stack = this.stack[1:]
	return out
}

// 放牌 index: 0第一张 1第二张 -1牌底 -2随机
func (this *Desk) PutCard(c Card, index int) {
	if index < -1 {
		index = rand.Intn(len(this.stack) + 1)
	}
	if index < 0 || index >= len(this.stack) {
		this.stack = append(this.stack, c)
		return
	}
	var out = make([]Card, 0, len(this.stack)+1)
	out = append(out, this.stack[:index]...)
	out = append(out, c)
	out = append(out, this.stack[index:]...)
	this.stack = out
}

func (this *Desk) Bombs() int {
	var n int
	for _, c := range this.stack {
		if c == CARD_BOMB {
			n++
		}
	}
	return n
}

func (this *Desk) SendStatus(me *Player) {
	if me.bot {
		return
	}
	s := fmt.Sprintf("")
	s += "------------------------------ 当前状态 --------------------------\n"
	s += fmt.Sprintf("  stack: %d cards, bomb: %d\n\n", len(this.stack), this.Bombs())
	for i, p := range this.seats {
		var arrow = " "
		var timetext string
		var card_text string
		if p.no == this.state.current {
			arrow = ">"
			timetext = fmt.Sprintf("(Time Left: %ds)", this.expire_at-int(time.Now().Unix()))
		}
		var name = p.Name
		if p.Id == me.Id {
			card_text = groupCards(p.cards)
			name = "You"
		} else {
			card_text = fmt.Sprintf("%d cards", len(p.cards))
		}
		if p.dead {
			card_text = "DEAD"
		}

		s += fmt.Sprintf("%s %d(%s):\t %s %s\n", arrow, i+1, name, card_text, timetext)
	}
	s += "-------------------------------------------------------------------"
	me.User.Println(s)
}

func (this *Desk) OneTurn() {
	this.turn++
	this.expire_at = int(time.Now().Unix()) + TurnTimeout
	expire_t := time.Now().Add(TurnTimeout * time.Second)

	p := this.seats[this.state.current-1]
	if p.bot {
		go func() {
			time.Sleep(time.Duration(rand.Intn(5)+1) * time.Second)
			msg := this.DeltaGo(p)
			this.Post(p.User, msg)
		}()
	}
	switch this.state.action {
	case ACT_GO:
		p.Printf("现在由你出牌或摸牌(m)..\n")
		this.Sendothersf(p.Id, "现在由%s出牌或摸牌..\n", p.Name)
	case ACT_DISARM:
		p.Printf("现在由你拆除炸弹(输入数字, 放在第几张, 0牌底, r随机)..\n")
		this.Sendothersf(p.Id, "现在等%s拆除炸弹..\n", p.Name)
	case ACT_DILIVER:
		var from_name = "?"
		f := this.players[this.state.from_id]
		if f != nil {
			from_name = f.Name
		}
		this.Sendallf("现在等%s给%s一张牌(牌的快捷键)..\n", p.Name, from_name)
	default:
		this.Sendallf("现在由%s行动..\n", p.Name)
	}

	for {
		m := this.Read(time.Until(expire_t))
		if m == nil {
			this.Sendall("(超时)")
			this.AutoGo()
			break
		} else {
			ok := this.handleMail(m)
			if ok {
				break
			}
		}
	}
}

func (this *Desk) AutoGo() {
	player := this.seats[this.state.current-1]
	switch this.state.action {
	case ACT_GO:
		this.cmdDraw(player)
	case ACT_DISARM:
		this.cmdDisarm(player, -2)
	case ACT_DILIVER:
		if len(player.cards) > 0 {
			this.cmdDiliver(player, player.cards[0])
		}
	}
}

// 摸牌
func (this *Desk) cmdDraw(player *Player) {
	c := this.PopCard()
	if c == CARD_BOMB {
		if player.HasDisarm() {
			this.state.combo = 0
			this.state.action = ACT_DISARM
			this.state.from_id = 0
			player.Println("你不小心摸到一颗炸弹, 请立即拆除! (选择将炸弹放第几张: 1-5, 0牌底, r随机)")
			this.Sendothersf(player.Id, "%s摸到一颗炸弹, 等待拆除..\n", player.Name)
		} else {
			player.dead = true
			this.state.combo = 0
			this.state.action = ACT_GO
			this.state.from_id = 0
			this.Next()
			player.Printf("你摸到炸弹, 没有拆除, 光荣牺牲! 得到第%d名.\n", this.Alives()+1)
			this.Sendothersf(player.Id, "DUANG! 摸到炸弹, %s骂骂咧咧退出游戏.\n", player.Name)
		}
	} else {
		player.PutCard(c)
		this.state.combo--
		this.state.action = ACT_GO
		this.state.from_id = 0
		if this.state.combo > 0 {
			player.Printf("你摸到一张%s, 再摸%d次\n", c, this.state.combo)
		} else {
			player.Printf("你摸到一张%s\n", c)
		}

		this.Sendothersf(player.Id, "%s摸了一张牌\n", player.Name)
		if this.state.combo <= 0 {
			this.state.combo = 0
			this.Next()
		}
	}
}

// 拆除
func (this *Desk) cmdDisarm(player *Player, index int) {
	this.PutCard(CARD_BOMB, index)
	player.RemoveCard(CARD_DISARM)
	this.state.combo = 0
	this.state.action = ACT_GO
	this.state.from_id = 0
	this.Sendallf("%s拆除了炸弹.\n", player.Name)
	this.Next()
}

// 交牌
func (this *Desk) cmdDiliver(player *Player, card Card) bool {
	ok := player.RemoveCard(card)
	if !ok {
		player.Println("错误: 你没有那张牌")
		return false
	}
	from := this.players[this.state.from_id]
	if from == nil {
		player.Println("异常: from==nil")
		return false
	}
	from.PutCard(card)
	this.state.current = from.no
	this.state.action = ACT_GO
	this.state.from_id = 0
	for _, p := range this.seats {
		if p.Id == player.Id {
			p.Printf("你给了%s一张%s.\n", from.Name, card)
		} else if p.Id == from.Id {
			p.Printf("%s给了你一张%s.\n", player.Name, card)
		} else {
			p.Printf("%s给了%s一张牌.\n", player.Name, from.Name)
		}
	}
	return true
}

func (this *Desk) getNext() int {
	current := this.state.current
	for i := 0; i < MaxSeats; i++ {
		if this.state.direct == 1 {
			current = (current)%MaxSeats + 1
		} else {
			current = (current+MaxSeats-2)%MaxSeats + 1
		}
		p := this.seats[current-1]
		if !p.dead {
			break
		}
	}
	return current
}

func (this *Desk) Next() {
	this.state.current = this.getNext()
}

// 返回true表示本轮可以结束
func (this *Desk) handleMail(m *Mail) bool {
	log.Println("recv mail:", m)
	if m.User == nil {
		return false
	}
	p := this.players[m.User.Id]
	if p == nil {
		return false
	}
	cmd, arg := m.Command()
	if cmd == "l" {
		this.SendStatus(p)
		return false
	} else if cmd == "c" {
		this.Sendallf("%s: %s\n", p.Name, arg)
		return false
	} else if cmd == "?" {
		p.Println(this.helpInfo())
		return false
	} else if cmd == "q" {
		this.Sendallf("%s离开了牌桌\n", p.Name)
		this.RemovePlayer(p.User)
		return false
	}
	if this.state.current != p.no {
		return false
	}
	if this.state.action == ACT_GO {
		switch cmd {
		case KEY_DRAW:
			this.cmdDraw(p)
			return true
		case KEY_PREDICT:
			this.cmdPredict(p)
			return true
		case KEY_PERSP:
			this.cmdPersp(p)
			return true
		case KEY_ASK:
			ok := this.cmdAsk(p, arg.Int())
			return ok
		case KEY_SWAP:
			ok := this.cmdSwap(p, arg.Int())
			return ok
		case KEY_SHUFFLE:
			this.cmdShuffle(p)
			return true
		case KEY_PASS:
			this.cmdPass(p)
			return true
		case KEY_TURN:
			this.cmdTurn(p)
			return true
		case KEY_SHIRK:
			ok := this.cmdShirk(p, arg.Int(), 1)
			return ok
		case KEY_SHIRK2:
			ok := this.cmdShirk(p, arg.Int(), 2)
			return ok
		case KEY_BOTTOM:
			this.cmdBottom(p)
			return true
		default:
			p.Println("?")
		}
	} else if this.state.action == ACT_DISARM {
		var index int
		if cmd == "r" {
			index = -2
		} else {
			index, _ = strconv.Atoi(cmd)
			index--
		}
		if index > 4 {
			p.Println("错误: 位置应为0-5或r")
			return false
		}
		this.cmdDisarm(p, index)
		return true
	} else if this.state.action == ACT_DILIVER {
		c, _ := strconv.Atoi(cmd)
		ok := this.cmdDiliver(p, Card(c))
		return ok
	}
	return false
}

// 预言
func (this *Desk) cmdPredict(player *Player) {
	player.RemoveCard(CARD_PREDICT)
	n := this.findCard(CARD_BOMB) + 1
	player.Printf("预言: 炸弹在第%d张\n", n)
	this.Sendothersf(player.Id, "%s使用了预言", player.Name)
}

func (this *Desk) cmdPersp(player *Player) {
	player.RemoveCard(CARD_PERSP)
	var cards []Card
	if len(this.stack) < 3 {
		cards = this.stack
	} else {
		cards = this.stack[:3]
	}
	tip := joinCards(cards)
	player.Printf("透视: %s\n", tip)
	this.Sendothersf(player.Id, "%s使用了透视", player.Name)
}

func (this *Desk) cmdAsk(player *Player, no int) bool {
	if no < 1 || no > MaxSeats {
		player.Println("错误的参数, 座位号必须是1-5")
		return false
	}
	if no == player.no {
		player.Println("错误: 不能索要自己")
		return false
	}
	peer := this.seats[no-1]
	if peer.dead {
		player.Println("错误: 对方已死")
		return false
	}
	player.RemoveCard(CARD_ASK)
	this.state.action = ACT_DILIVER
	this.state.from_id = player.Id
	this.state.current = peer.no
	this.Sendallf("%s索要%s\n", player.Name, peer.Name)
	return true
}

func (this *Desk) cmdSwap(player *Player, no int) bool {
	if no < 1 || no > MaxSeats {
		player.Println("错误的参数, 座位号必须是1-5")
		return false
	}
	if no == player.no {
		player.Println("错误: 不能交换自己")
		return false
	}
	peer := this.seats[no-1]
	if peer.dead {
		player.Println("错误: 对方已死")
		return false
	}
	player.RemoveCard(CARD_SWAP)
	player.cards, peer.cards = peer.cards, player.cards
	this.Sendallf("%s换牌%s\n", player.Name, peer.Name)
	return true
}

// 返回牌在第几张
func (this *Desk) findCard(card Card) int {
	for i, c := range this.stack {
		if c == card {
			return i
		}
	}
	return -1
}

func (this *Desk) cmdShuffle(player *Player) {
	player.RemoveCard(CARD_SHUFFLE)
	rand.Shuffle(len(this.stack), func(i, j int) {
		this.stack[i], this.stack[j] = this.stack[j], this.stack[i]
	})
	this.Sendallf("%s洗牌\n", player.Name)
}

func (this *Desk) cmdPass(player *Player) {
	player.RemoveCard(CARD_PASS)
	this.state.combo--
	if this.state.combo < 1 {
		this.state.combo = 1
		this.state.action = ACT_GO
		this.state.from_id = 0
		this.Next()
	}
	this.Sendallf("%s跳过\n", player.Name)
}

func (this *Desk) cmdTurn(player *Player) {
	player.RemoveCard(CARD_TURN)
	if this.state.direct == 1 {
		this.state.direct = 2
	} else {
		this.state.direct = 1
	}
	this.state.combo--
	if this.state.combo < 1 {
		this.state.combo = 0
		this.state.action = ACT_GO
		this.state.from_id = 0
		this.Next()
	}
	this.Sendallf("%s转向\n", player.Name)
}

func (this *Desk) cmdShirk(player *Player, no int, hits int) bool {
	if no < 1 || no > MaxSeats {
		player.Println("错误的参数, 座位号必须是1-5")
		return false
	}
	peer := this.seats[no-1]
	if peer.dead {
		player.Println("错误: 对方已死")
		return false
	}
	if hits <= 1 {
		player.RemoveCard(CARD_SHIRK)
	} else {
		player.RemoveCard(CARD_SHIRK2)
	}
	this.state.current = peer.no
	this.state.action = ACT_GO
	this.state.combo += hits
	this.state.from_id = player.Id
	if player.no == no {
		this.Sendallf("%s甩锅自己, 连击x%d\n", player.Name, this.state.combo)
	} else {
		this.Sendallf("%s甩锅%s, 连击x%d\n", player.Name, peer.Name, this.state.combo)
	}
	return true
}

func (this *Desk) cmdBottom(player *Player) {
	player.RemoveCard(CARD_BOTTOM)
	if len(this.stack) == 0 {
		player.Println("错误: 没牌了")
		return
	}

	c := this.stack[len(this.stack)-1]
	this.stack = this.stack[:len(this.stack)-1]
	if c == CARD_BOMB {
		if player.HasDisarm() {
			this.state.combo = 0
			this.state.action = ACT_DISARM
			this.state.from_id = 0
			player.Println("你不小心摸到一颗炸弹, 请立即拆除! (选择将炸弹放第几张: 1-5, 0牌底, r随机)")
			this.Sendothersf(player.Id, "%s摸到一颗炸弹, 等待拆除..\n", player.Name)
		} else {
			player.dead = true
			this.state.combo = 0
			this.state.action = ACT_GO
			this.state.from_id = 0
			this.Next()
			player.Printf("你摸到炸弹, 没有拆除, 光荣牺牲! 得到第%d名.\n", this.Alives()+1)
			this.Sendothersf(player.Id, "DUANG! 摸到炸弹, %s骂骂咧咧退出游戏.\n", player.Name)
		}
	} else {
		player.PutCard(c)
		player.Printf("你抽底得到一张%s\n", c)
		this.state.combo--
		if this.state.combo < 1 {
			this.state.combo = 0
			this.state.action = ACT_GO
			this.state.from_id = 0
			this.Next()
		}
		this.Sendothersf(player.Id, "%s抽底\n", player.Name)
	}
}

func (this *Desk) helpInfo() string {
	s := ""
	s += `
	摸一张牌(m), 聊天(c), 查看状态(l), 帮助(?), 离开(q)

	[出牌快捷键]
	预言: y
	透视: t
	索要: a <座位号1-5>
	交换: e <座位号1-5>
	洗牌: x
	跳过: j
	转向: d
	甩锅: s <座位号1-5>
	双甩: f <座位号1-5>
	抽底: b
	拆除: r <将炸弹放第几张, 1-5, 0牌底, r随机>

	[牌型介绍]
	预言: 查看最近一张炸弹位置
	透视: 查看底牌最上方三张牌
	索要: 指定一个玩家给你一张牌
	交换: 指定一名玩家, 与他交换手牌
	洗牌: 打乱底牌顺序
	跳过: 结束本回合, 到下一个玩家
	转向: 结束本回合, 改变出牌方向
	甩锅: 结束本回合, 指定玩家回合+1
	双甩: 结束本回合, 指定玩家回合+2
	抽底: 摸起底牌最下方一张牌
	拆除: 拆除炸弹化解危机
`
	return s
}
