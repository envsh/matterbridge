package btox

import (
	"fmt"
	"gopp"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	tox "github.com/kitech/go-toxcore"
	"github.com/kitech/go-toxcore/xtox"
)

var startTime = time.Now()

func (this *Btox) processFriendCmd(friendNumber uint32, msg string) {
	t := this.i

	pubkey, err := t.FriendGetPublicKey(friendNumber)
	gopp.ErrPrint(err)
	_ = pubkey
	friendName, err := t.FriendGetName(friendNumber)
	gopp.ErrPrint(err)
	_ = friendName

	msg = strings.Trim(msg, "\n\t ")
	if msg == "info" {
		this.processInfoCmd(friendNumber, msg)
	} else if msg == "help" {
		this.processHelpCmd(friendNumber, msg, pubkey)
	} else if msg == "id" {
		t.FriendSendMessage(friendNumber, t.SelfGetAddress())
	} else if msg == "joined" {
		this.processJoinedCmd(friendNumber, msg, pubkey)
	}
	// TODO not joined
	// TODO 管理员命令，隐藏的

	if strings.HasPrefix(msg, "join ") {
		this.processJoinCmd(friendNumber, msg, pubkey)
	} else if strings.HasPrefix(msg, "leave ") {
		this.processLeaveCmd(friendNumber, msg, pubkey)
	} else if strings.HasPrefix(msg, "create ") {
		t.FriendSendMessage(friendNumber, "Online create group coming soon. Or contact ME.")
	}
}

func (this *Btox) processHelpCmd(friendNumber uint32, msg string, pubkey string) {
	t := this.i

	hmsg := "help : Print this message\n\n"
	hmsg += "info : Print my current status and list active group chats\n\n"
	hmsg += "id : Print my Tox ID\n\n"
	hmsg += "join <name|number> : Join selected group.Permanent effective\n\n"
	hmsg += "leave <name|number> : Leave selected group.Permanent effective\n\n"
	hmsg += "create <name> : Create a new group\n\n"
	hmsg += "joined : You joined groups\r\n"
	// TODO invite for fix sometimes
	t.FriendSendMessage(friendNumber, hmsg)
}

// TODO 所连接的协议信息
func (this *Btox) processInfoCmd(friendNumber uint32, msg string) {
	t := this.i

	rmsg := ""
	// basic info
	rmsg += fmt.Sprintf("Uptime: %s\n\n", time.Now().Sub(startTime))
	// rmsg += fmt.Sprintf("Friends: %d (%d online)\n\n", 0,0)
	log.Println("get Uptime:", rmsg)

	// groups info
	gntitles := xtox.ConferenceAllTitles(t)
	gns := []int{}
	for gn, _ := range gntitles {
		gns = append(gns, int(gn))
	}
	sort.Ints(gns)
	log.Println("get groups info:", len(gns))

	for _, gn_ := range gns {
		gn := uint32(gn_)
		title := gntitles[gn]
		pcnt := t.ConferencePeerCount(gn)
		itype, _ := t.ConferenceGetType(gn)
		ttype := gopp.IfElseStr(itype == tox.CONFERENCE_TYPE_AV, "Audio", "Text")
		isours := gopp.IfElseInt(xtox.IsInvitedGroup(t, gn), 0, 1)
		rmsg += fmt.Sprintf("Group %d | %s | Peers: %d | Ours: %d | Title: %s\n\n",
			gn, ttype, pcnt, isours, title)
	}

	msgs := gopp.Splitn(rmsg, 1000)
	log.Println("get Group detail:", len(rmsg), len(msgs))
	for _, msg := range msgs {
		_, err := t.FriendSendMessage(friendNumber, msg)
		gopp.ErrPrint(err)
	}
}

// TODO join/leave with number
func (this *Btox) processJoinCmd(friendNumber uint32, msg string, pubkey string) {
	groupSymbol := msg[5:]
	if gopp.IsInteger(groupSymbol) {
		this.processJoinCmdByNumber(friendNumber, msg, pubkey)
	} else {
		this.processJoinCmdByName(friendNumber, msg, pubkey)
	}
}

func (this *Btox) processJoinCmdByName(friendNumber uint32, msg string, pubkey string) {
	t := this.i
	var err error

	groupName := msg[5:]
	log.Println(groupName)
	gns, found := xtox.ConferenceFindAll(t, groupName)
	if found {
		var gn uint32 = math.MaxUint32
		for _, gn_ := range gns {
			if xtox.IsInvitedGroup(t, gn_) {
				gn = gn_
				break
			}
			gn = gn_
		}
		if true {
			_, err = t.ConferenceInvite(friendNumber, gn)
			gopp.ErrPrint(err)
		}

		// save to storage
		err = this.store.join(pubkey, groupName)
		gopp.ErrPrint(err)
		this.frndjrman.rtJoin(pubkey, groupName)
		this.frndjrman.cfgJoin(pubkey, groupName)
	} else {
		log.Println("not found:", groupName)
		rmsg := fmt.Sprintf("Group not found: %s", groupName)
		t.FriendSendMessage(friendNumber, rmsg)
	}
}

func (this *Btox) processJoinCmdByNumber(friendNumber uint32, msg string, pubkey string) {
	t := this.i
	var err error

	groupSymbol := msg[5:]
	groupNumberi, err := strconv.Atoi(groupSymbol)
	if err != nil {
		t.FriendSendMessage(friendNumber, "Invalid group number:"+groupSymbol)
		return
	}
	groupNumber := uint32(groupNumberi)

	groupName, err := t.ConferenceGetTitle(groupNumber)
	if err != nil {
		log.Println("Cannot get group title:", groupNumber, err)
		return
	}
	this.processJoinCmdByName(friendNumber, "join "+groupName, pubkey)
}

func (this *Btox) processLeaveCmd(friendNumber uint32, msg string, pubkey string) {
	groupSymbol := msg[6:]
	if gopp.IsInteger(groupSymbol) {
		this.processLeaveCmdByNumber(friendNumber, msg, pubkey)
	} else {
		this.processLeaveCmdByName(friendNumber, msg, pubkey)
	}
}
func (this *Btox) processLeaveCmdByName(friendNumber uint32, msg string, pubkey string) {
	t := this.i
	var err error

	groupName := msg[6:]
	log.Println(groupName)
	gns, found := xtox.ConferenceFindAll(t, groupName)
	if found {
		var gn uint32 = math.MaxUint32
		for _, gn_ := range gns {
			if xtox.IsInvitedGroup(t, gn_) {
				gn = gn_
				break
			}
			gn = gn_
		}
		if true {
			_ = gn
			// _, err = t.ConferenceInvite(friendNumber, gn)
			// gopp.ErrPrint(err)
			// how kick the group member?
		}

		// save to storage
		err = this.store.leave(pubkey, groupName)
		gopp.ErrPrint(err)
		this.frndjrman.rtLeave(pubkey, groupName)
		this.frndjrman.cfgLeave(pubkey, groupName)
	} else {
		log.Println("not found:", groupName)
		rmsg := fmt.Sprintf("Group not found: %s", groupName)
		t.FriendSendMessage(friendNumber, rmsg)
	}
}

func (this *Btox) processLeaveCmdByNumber(friendNumber uint32, msg string, pubkey string) {
	t := this.i
	var err error

	groupSymbol := msg[6:]
	groupNumberi, err := strconv.Atoi(groupSymbol)
	if err != nil {
		t.FriendSendMessage(friendNumber, "Invalid group number:"+groupSymbol)
		return
	}
	groupNumber := uint32(groupNumberi)

	groupName, err := t.ConferenceGetTitle(groupNumber)
	if err != nil {
		log.Println("Cannot get group title:", groupNumber, err)
		return
	}
	this.processLeaveCmdByName(friendNumber, "leave "+groupName, pubkey)
}

func (this *Btox) processJoinedCmd(friendNumber uint32, msg string, pubkey string) {
	t := this.i
	var err error

	recs, err := this.store.getRoomsByMemberId(pubkey)
	if err != nil {
		return
	}

	if len(recs) == 0 {
		rmsg := "You have not joined any room."
		t.FriendSendMessage(friendNumber, rmsg)
		return
	}

	rmsg := ""
	for n, rec := range recs {
		rmsg += fmt.Sprintf("%d: %s\n\n", n+1, rec.RoomName)
	}
	t.FriendSendMessage(friendNumber, rmsg)
}
