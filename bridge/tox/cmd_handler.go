package btox

import (
	"fmt"
	"gopp"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	// tox "github.com/kitech/go-toxcore"
	tox "github.com/TokTok/go-toxcore-c"
	"github.com/envsh/go-toxcore/xtox"
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
	} else if msg == "njoined" {
		this.processNotJoinedCmd(friendNumber, msg, pubkey)
	} else if msg == "uptime" {
		this.processUptimeCmd(friendNumber, msg, pubkey)
	}
	// TODO not joined
	// TODO 管理员命令，隐藏的
	// TODO 解散群命令，防止出现群分裂的时候无法合并

	if strings.HasPrefix(msg, "join ") {
		this.processJoinCmd(friendNumber, msg, pubkey)
	} else if strings.HasPrefix(msg, "leave ") {
		this.processLeaveCmd(friendNumber, msg, pubkey)
	} else if strings.HasPrefix(msg, "create ") {
		t.FriendSendMessage(friendNumber, "Online create group coming soon. Or contact ME.")
	} else if strings.HasPrefix(msg, "dissolve ") { //admin
		this.processDissolveCmd(friendNumber, msg, pubkey)
	} else if strings.HasPrefix(msg, "ban ") {
		this.processBanCmd(friendNumber, msg, pubkey)
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
	hmsg += "njoined : You not joined groups\r\n"
	hmsg += "uptime : uptime info of botproc\r\n"
	// hmsg += "ban <name> : ban a user with nick name" // TODO maybe bug is more than one user use the same nick name
	// hmsg += "unban <name> : unban a user with nick name"
	// hmsg += "baned : List baned users public key"
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
		ttype := gopp.IfElseStr(uint8(itype) == tox.CONFERENCE_TYPE_AV, "Audio", "Text")
		isours := gopp.IfElseInt(xtox.IsInvitedGroup(t, gn), 0, 1)
		rmsg += fmt.Sprintf("Group %d | %s | Peers: %d | Ours: %d | Title: %s\n\n",
			gn, ttype, pcnt, isours, title)
	}

	// msgs := gopp.Splitn(rmsg, 1000)
	msgs := gopp.Splitln(rmsg, 1000)
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

func (this *Btox) processDissolveCmd(friendNumber uint32, msg string, pubkey string) {
	t := this.i
	var err error

	groupSymbol := msg[9:]
	groupNumberi, err := strconv.Atoi(groupSymbol)
	gopp.ErrPrint(err)
	if err != nil {
		t.FriendSendMessage(friendNumber, "Invalid group number:"+groupSymbol)
		return
	}
	groupNumber := uint32(groupNumberi)

	_, err = t.ConferenceDelete(groupNumber)
	if err != nil {
		log.Println("Cannot get group title:", groupNumber, err)
		return
	}

	// TODO delete meta info in other struct
	toxaa.removeGroup(groupNumber)
}

func (this *Btox) processJoinedCmd(friendNumber uint32, msg string, pubkey string) {
	t := this.i
	var err error

	recs, err := this.store.getRoomsByMemberId(pubkey)
	gopp.ErrPrint(err)
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

func (this *Btox) processNotJoinedCmd(friendNumber uint32, msg string, pubkey string) {
	t := this.i
	var err error

	allrecs, err := this.store.getAllRoomMembers()
	gopp.ErrPrint(err)
	if err != nil {
		return
	}
	recs, err := this.store.getRoomsByMemberId(pubkey)
	gopp.ErrPrint(err)
	if err != nil {
		return
	}

	if len(recs) == 0 {
		rmsg := "You have not joined any room."
		t.FriendSendMessage(friendNumber, rmsg)
		return
	}

	rmsg := ""
	for idx, allrec := range allrecs {
		found := false
		for _, rec := range recs {
			if rec.MemberId == allrec.MemberId {
				found = true
				break
			}
		}
		if !found {
			rmsg += fmt.Sprintf("%d: %s\n\n", idx+1, allrec.RoomName)
		}
	}
	t.FriendSendMessage(friendNumber, rmsg)
}

var procStartTime = time.Now() //

func (this *Btox) processUptimeCmd(friendNumber uint32, msg string, pubkey string) {
	t := this.i
	now := time.Now()

	rmsg := ""
	rmsg += fmt.Sprintf("Uptime: %s\n", gopp.Dur2hum(now.Sub(procStartTime)))
	fns := xtox.GetAllFriendList(t)
	onlineNum := 0
	for _, fn := range fns {
		st, _ := t.FriendGetConnectionStatus(fn)
		if st > 0 {
			onlineNum += 1
		}
	}
	rmsg += fmt.Sprintf("Friends: %d (%d online)\n", len(fns), onlineNum)
	gns := t.ConferenceGetChatlist()
	allPeerPubkeys := make(map[string]bool, 0)
	for _, gn := range gns {
		peerPubkeys := t.ConferenceGetPeerPubkeys(gn)
		for _, pubkey := range peerPubkeys {
			allPeerPubkeys[pubkey] = true
		}
	}
	rmsg += fmt.Sprintf("Groups: %d, Peers: %d\n", len(gns), len(allPeerPubkeys))

	t.FriendSendMessage(friendNumber, rmsg)
}

// 禁止的用户列，内存中存在，重启重置
// 目前禁止的用户还不影响任何流程
// TODO 记录入库
var banedUserList sync.Map

func isBanedUser(pubkey string) bool {
	_, ok := banedUserList.Load(pubkey)
	return ok
}

// 从群里查找
func FindGroupPeerByName(t *tox.Tox, name string) (pubkey string, found bool) {
	pubkey, found = xtox.ConferenceFindPeer(t, name)
	return
}

func (this *Btox) processBanCmd(friendNumber uint32, msg string, pubkey string) {
	t := this.i

	fname := strings.Trim(msg[4:], " ")
	fpubkey, found := FindGroupPeerByName(t, fname)

	if found {
		banedUserList.Store(fpubkey, fname+"|"+pubkey+"|"+time.Now().String())
	} else {
		log.Println("ban name not exist:", msg)
	}
}

func (this *Btox) processUnbanCmd(friendNumber uint32, msg string, pubkey string) {
	t := this.i

	fname := strings.Trim(msg[4:], " ")
	fpubkey, found := FindGroupPeerByName(t, fname)

	if found {
		banedUserList.Delete(fpubkey)
	} else {
		log.Println("unban name not exist:", msg)
	}

}

func (this *Btox) processBanedCmd(friendNumber uint32, msg string, pubkey string) {
	t := this.i

	cnter := 0
	str := ""
	banedUserList.Range(func(key interface{}, value interface{}) bool {
		str += fmt.Sprintf("%d %v=%v\n", cnter, key, value)
		cnter++
		return true
	})

	if cnter > 0 {
		chunks := gopp.Splitln(str, 1000)
		for _, chunk := range chunks {
			t.FriendSendMessage(friendNumber, chunk)
		}
	}
}
