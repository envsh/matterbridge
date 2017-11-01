package btox

import (
	"fmt"
	"gopp"
	"log"
	"math"
	"sort"
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

	if msg == "info" {
		this.processInfoCmd(friendNumber, msg)
	} else if msg == "help" {
		this.processHelpCmd(friendNumber, msg, pubkey)
	} else if msg == "id" {
		t.FriendSendMessage(friendNumber, t.SelfGetAddress())
	}

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
	hmsg += "join <name> : Join selected group\n\n"
	hmsg += "leave <name> : Leave selected group\n\n"
	hmsg += "create <name> : Create a new group\n\n"
	t.FriendSendMessage(friendNumber, hmsg)
}

func (this *Btox) processInfoCmd(friendNumber uint32, msg string) {
	t := this.i

	rmsg := ""
	// basic info
	rmsg += fmt.Sprintf("Uptime: %s\n\n", time.Now().Sub(startTime))
	// rmsg += fmt.Sprintf("Friends: %d (%d online)\n\n", 0,0)

	// group info
	gntitles := xtox.ConferenceAllTitles(t)
	gns := []int{}
	for gn, _ := range gntitles {
		gns = append(gns, int(gn))
	}
	sort.Ints(gns)

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
	t.FriendSendMessage(friendNumber, rmsg)
}

func (this *Btox) processJoinCmd(friendNumber uint32, msg string, pubkey string) {
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
	} else {
		log.Println("not found:", groupName)
		rmsg := fmt.Sprintf("Group not found: %s", groupName)
		t.FriendSendMessage(friendNumber, rmsg)
	}
}

func (this *Btox) processLeaveCmd(friendNumber uint32, msg string, pubkey string) {
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
	} else {
		log.Println("not found:", groupName)
		rmsg := fmt.Sprintf("Group not found: %s", groupName)
		t.FriendSendMessage(friendNumber, rmsg)
	}
}
