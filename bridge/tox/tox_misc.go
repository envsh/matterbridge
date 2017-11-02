package btox

import (
	"gopp"
	"log"

	tox "github.com/kitech/go-toxcore"
	"github.com/kitech/go-toxcore/xtox"
)

func (this *Btox) updatePeerState(groupNumber uint32, peerNumber uint32, change uint8) {
	t := this.i

	peerPubkeyd, errpk := t.ConferencePeerGetPublicKey(groupNumber, peerNumber)
	groupTitled, errgt := t.ConferenceGetTitle(groupNumber)
	peerPubkey, foundpk := xtox.ConferencePeerGetPubkey(t, groupNumber, peerNumber)
	groupTitle, foundgt := xtox.ConferenceGetTitle(t, groupNumber)
	gopp.ErrPrint(errpk, groupNumber, peerPubkeyd, peerNumber, change, groupTitle)
	gopp.ErrPrint(errgt, groupNumber, groupTitled, peerNumber, change, peerPubkeyd,
		errpk, peerPubkey, groupTitle)
	if !foundgt || !foundpk {
		log.Println("lack info:", foundgt, groupTitle, peerNumber, change,
			peerPubkey, foundpk, peerPubkeyd, groupTitled)
	}
	if errpk != nil && foundpk != true {
		// can not get pubkey
	}
	if errgt != nil && foundgt != true {
		// can not get title
	}
	switch change {
	case tox.CHAT_CHANGE_PEER_ADD:
		if foundgt == true && foundpk == true {
			this.frndjrman.rtJoin(peerPubkey, groupTitle)
		}
		if foundpk == true {
			this.frndjrman.rtJoinByNumber(peerPubkey, groupNumber)
		}
		if errpk == nil {
			this.frndjrman.rtJoinByNumber(peerPubkeyd, groupNumber)
		}
	case tox.CHAT_CHANGE_PEER_DEL:
		if foundgt == true && foundpk == true {
			this.frndjrman.rtLeave(peerPubkey, groupTitle)
		}
		if foundpk == true {
			this.frndjrman.rtLeaveByNumber(peerPubkey, groupNumber)
		}
		if errpk == nil {
			this.frndjrman.rtLeaveByNumber(peerPubkeyd, groupNumber)
		}
	}
}
