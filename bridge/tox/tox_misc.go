package btox

import (
	"gopp"
	"log"

	// tox "github.com/kitech/go-toxcore"
	// "github.com/kitech/go-toxcore/xtox"

	"github.com/envsh/go-toxcore/xtox"
)

func (this *Btox) updatePeerState(groupNumber uint32, peerNumber uint32, change uint8) {
	t := this.i

	peerPubkeyd, errpk := t.ConferencePeerGetPublicKey(groupNumber, peerNumber)
	groupTitled, errgt := t.ConferenceGetTitle(groupNumber)
	peerPubkey, foundpk := xtox.ConferencePeerGetPubkey(t, groupNumber, peerNumber)
	groupTitle, foundgt := xtox.ConferenceGetTitle(t, groupNumber)
	groupId, foundid := xtox.ConferenceGetIdentifier(t, groupNumber)
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
	case xtox.CHAT_CHANGE_PEER_ADD:
		if foundgt == true && foundpk == true {
			this.frndjrman.rtJoin(peerPubkey, groupTitle)
		}
		if foundpk == true {
			// this.frndjrman.rtJoinByNumber(peerPubkey, groupNumber)
		}
		if foundid == nil {
			this.frndjrman.rtJoinByIdentifer(peerPubkey, groupId)
		}
		if errpk == nil {
			// this.frndjrman.rtJoinByNumber(peerPubkeyd, groupNumber)
		}
	case xtox.CHAT_CHANGE_PEER_DEL:
		if foundgt == true && foundpk == true {
			this.frndjrman.rtLeave(peerPubkey, groupTitle)
		}
		if foundpk == true {
			// this.frndjrman.rtLeaveByNumber(peerPubkey, groupNumber)
		}
		if errpk == nil {
			// this.frndjrman.rtLeaveByNumber(peerPubkeyd, groupNumber)
		}
		if foundid == nil {
			this.frndjrman.rtLeaveByIdentifier(peerPubkeyd, groupId)
		}
	}
}

func (this *Btox) updatePeerState2(groupNumber uint32, peerPubkey string, change uint8) {
	t := this.i

	groupTitled, errgt := t.ConferenceGetTitle(groupNumber)
	groupTitle, foundgt := xtox.ConferenceGetTitle(t, groupNumber)
	groupId, foundid := xtox.ConferenceGetIdentifier(t, groupNumber)
	gopp.ErrPrint(errgt, groupNumber, peerPubkey[:20], change, groupTitle, foundid)
	if !foundgt {
		log.Println("lack info:", foundgt, groupTitle, change,
			peerPubkey[:20], groupTitled)
	}

	switch change {
	case xtox.CHAT_CHANGE_PEER_ADD:
		this.frndjrman.rtJoin(peerPubkey, groupTitle)
		// this.frndjrman.rtJoinByNumber(peerPubkey, groupNumber)
		this.frndjrman.rtJoinByIdentifer(peerPubkey, groupId)
	case xtox.CHAT_CHANGE_PEER_DEL:
		this.frndjrman.rtLeave(peerPubkey, groupTitle)
		// this.frndjrman.rtLeaveByNumber(peerPubkey, groupNumber)
		this.frndjrman.rtLeaveByIdentifier(peerPubkey, groupId)
	}
}
