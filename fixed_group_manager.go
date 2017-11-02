package btox

import (
	"gopp"
	"log"
	"sync"

	tox "github.com/kitech/go-toxcore"
	"github.com/kitech/go-toxcore/xtox"
	"github.com/kitech/godsts/sets/hashset"
)

// 根据存储的数据库群组信息，确定邀请好友到群组中
func (this *Btox) tryInviteFriendToGroups(friendNumber uint32, status int) {
	t := this.i
	pubkey, err := t.FriendGetPublicKey(friendNumber)
	gopp.ErrPrint(err)
	fname, _ := t.FriendGetName(friendNumber)

	if status == tox.CONNECTION_NONE {
		// nothing to do
	} else {
		rooms, err := this.store.getRoomsByMemberId(pubkey)
		gopp.ErrPrint(err)
		if len(rooms) > 0 {
			log.Println("Try invite friend to rooms:", friendNumber, fname, len(rooms))
		} else {
			log.Println("Friend not in any room:", friendNumber, fname)
		}
		for _, room := range rooms {
			gn, found := xtox.ConferenceFind(t, room.RoomName)
			if found {
				_, err = t.ConferenceInvite(friendNumber, gn)
				gopp.ErrPrint(err, friendNumber, fname, room.RoomName)
			} else {
				log.Println("Can't find room:", friendNumber, fname, room.RoomName)
			}
		}
	}
}

var unexpectedLeftRoomFriends sync.Map

// 定时检测好友是否在其加入的群组中，如果不在，则尝试拉进群组
func (this *Btox) checkFriendInRoomOrInvite() {
	t := this.i

	log.Println(this.frndjrman.count())
	gntitles := xtox.ConferenceAllTitles(t)
	fns := xtox.GetAllFriendList(t)
	for _, fn := range fns {
		status, err := t.FriendGetConnectionStatus(fn)
		gopp.ErrPrint(err)
		if status == tox.CONNECTION_NONE {
			continue
		}

		//
		pubkey, _ := t.FriendGetPublicKey(fn)
		fname, _ := t.FriendGetName(fn)

		// nowInRoom(pubkey, room)?
		// shouldInRoom(pubkey, room)?
		for gn, title := range gntitles {
			ok1 := this.frndjrman.shouldInRoom(pubkey, title)
			if !ok1 {
				continue
			}
			ok2 := this.frndjrman.nowInRoom(pubkey, title)
			ok3 := this.frndjrman.nowInRoomByNumber(pubkey, gn)

			if ok1 && !(ok2 || ok3) {
				// invite friendNumber to gn
				log.Println("Friend should but not in room:", fn, fname, pubkey, gn, title,
					gopp.Retn(this.frndjrman.rtFriendJoinedRoomNumbers.Load(pubkey)),
					gopp.Retn(this.frndjrman.rtFriendJoinedRoomNames.Load(pubkey)))
				_, err := t.ConferenceInvite(fn, gn)
				gopp.ErrPrint(err)
			}
		}
	}
}

////// 管理好友所在的群组实时数据
type FriendJoinedRoomsManager struct {
	brg                        *Btox
	rtFriendJoinedRoomNames    sync.Map // peer pubkey => room list
	rtFriendJoinedRoomNumbers  sync.Map // peer pubkey => room numer
	cfgFriendJoinedRoomNames   sync.Map // peer pubkey => room list
	cfgFriendJoinedRoomNumbers sync.Map // peer pubkey => room numer
}

func newFriendJoinedRoomsManager(brg *Btox) *FriendJoinedRoomsManager {
	this := &FriendJoinedRoomsManager{}
	this.brg = brg
	return this
}

func (this *FriendJoinedRoomsManager) loadConfigData() {
	recs, err := this.brg.store.getAllRoomMembers()
	gopp.ErrPrint(err)
	for _, rec := range recs {
		rooms := this.getCfgRoomSetByPubkey(rec.MemberId)
		if rec.Disabled == 0 {
			rooms.Add(rec.RoomName)
		}
	}
	log.Println("Load RoomMember config done:", len(recs))
}

func (this *FriendJoinedRoomsManager) getCfgRoomSetByPubkey(pubkey string) *hashset.Set {
	var rooms *hashset.Set
	roomsx, ok := this.cfgFriendJoinedRoomNames.Load(pubkey)
	if !ok {
		// 不存在，则创建并加入
		rooms = hashset.New()
		this.cfgFriendJoinedRoomNames.Store(pubkey, rooms)
	} else {
		rooms = roomsx.(*hashset.Set)
	}
	return rooms
}

func (this *FriendJoinedRoomsManager) getRtRoomSetByPubkey(pubkey string) *hashset.Set {
	var rooms *hashset.Set
	roomsx, ok := this.rtFriendJoinedRoomNames.Load(pubkey)
	if !ok {
		// 不存在，则创建并加入
		rooms = hashset.New()
		this.rtFriendJoinedRoomNames.Store(pubkey, rooms)
	} else {
		rooms = roomsx.(*hashset.Set)
	}
	return rooms
}

func (this *FriendJoinedRoomsManager) getRtRoomNumberSetByPubkey(pubkey string) *hashset.Set {
	var rooms *hashset.Set
	roomsx, ok := this.rtFriendJoinedRoomNumbers.Load(pubkey)
	if !ok {
		// 不存在，则创建并加入
		rooms = hashset.New()
		this.rtFriendJoinedRoomNumbers.Store(pubkey, rooms)
	} else {
		rooms = roomsx.(*hashset.Set)
	}
	return rooms
}

func (this *FriendJoinedRoomsManager) shouldInRoom(pubkey, name string) bool {
	if roomsx, ok := this.cfgFriendJoinedRoomNames.Load(pubkey); ok {
		return roomsx.(*hashset.Set).Contains(name)
	}
	return false
}

func (this *FriendJoinedRoomsManager) nowInRoom(pubkey, name string) bool {
	if roomsx, ok := this.rtFriendJoinedRoomNames.Load(pubkey); ok {
		// log.Println(pubkey, name, roomsx.(*hashset.Set).Values())
		return roomsx.(*hashset.Set).Contains(name)
	}
	return false
}

func (this *FriendJoinedRoomsManager) nowInRoomByNumber(pubkey string, groupNumber uint32) bool {
	if roomsx, ok := this.rtFriendJoinedRoomNumbers.Load(pubkey); ok {
		// log.Println(pubkey, groupNumber, roomsx.(*hashset.Set).Values())
		return roomsx.(*hashset.Set).Contains(groupNumber)
	}
	return false
}

func (this *FriendJoinedRoomsManager) cfgJoin(pubkey, name string) {
	rooms := this.getCfgRoomSetByPubkey(pubkey)
	if !rooms.Contains(name) {
		rooms.Add(name)
	}
}

func (this *FriendJoinedRoomsManager) cfgLeave(pubkey, name string) {
	rooms := this.getCfgRoomSetByPubkey(pubkey)
	if rooms.Contains(name) {
		rooms.Remove(name)
	}
}

func (this *FriendJoinedRoomsManager) rtJoin(pubkey, name string) {
	if isDeletedGroupName(name) {
		return
	}
	rooms := this.getRtRoomSetByPubkey(pubkey)
	if !rooms.Contains(name) {
		rooms.Add(name)
	}
}

func (this *FriendJoinedRoomsManager) rtLeave(pubkey, name string) {
	rooms := this.getRtRoomSetByPubkey(pubkey)
	if rooms.Contains(name) {
		rooms.Remove(name)
	}
}

func (this *FriendJoinedRoomsManager) rtJoinByNumber(pubkey string, groupNumber uint32) {
	rooms := this.getRtRoomNumberSetByPubkey(pubkey)
	if !rooms.Contains(groupNumber) {
		rooms.Add(groupNumber)
	}
}

func (this *FriendJoinedRoomsManager) rtLeaveByNumber(pubkey string, groupNumer uint32) {
	rooms := this.getRtRoomNumberSetByPubkey(pubkey)
	if rooms.Contains(groupNumer) {
		rooms.Remove(groupNumer)
	}
}

func (this *FriendJoinedRoomsManager) count() map[string]int {
	cfgcnt := 0
	this.cfgFriendJoinedRoomNames.Range(func(key interface{}, value interface{}) bool {
		cfgcnt++
		return true
	})
	rtscnt := 0
	this.rtFriendJoinedRoomNames.Range(func(key interface{}, value interface{}) bool {
		rtscnt++
		return true
	})
	rtdcnt := 0
	this.rtFriendJoinedRoomNames.Range(func(key interface{}, value interface{}) bool {
		rtdcnt++
		return true
	})
	ret := map[string]int{"cfg": cfgcnt, "rts": rtscnt, "rtd": rtdcnt}
	return ret
}
