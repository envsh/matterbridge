package btox

import (
	"fmt"
	"gopp"
	"log"
	"math"
	"strings"

	tox "github.com/kitech/go-toxcore"
	"github.com/kitech/go-toxcore/xtox"
)

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
	}

	if strings.HasPrefix(msg, "join ") {
		this.processJoinCmd(friendNumber, msg, pubkey)
	} else if strings.HasPrefix(msg, "leave ") {
		this.processLeaveCmd(friendNumber, msg, pubkey)
	}
}

func (this *Btox) processInfoCmd(friendNumber uint32, msg string) {
	t := this.i

	gntitles := xtox.ConferenceAllTitles(t)
	rmsg := ""
	for gn, title := range gntitles {
		pcnt := t.ConferencePeerCount(gn)
		itype, _ := t.ConferenceGetType(gn)
		ttype := gopp.IfElseStr(itype == tox.CONFERENCE_TYPE_AV, "AV", "Text")
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
	} else {
		log.Println("not found:", groupName)
		rmsg := fmt.Sprintf("Group not found: %s", groupName)
		t.FriendSendMessage(friendNumber, rmsg)
	}
}
