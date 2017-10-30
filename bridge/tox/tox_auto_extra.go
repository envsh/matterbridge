package btox

import (
	"fmt"
	"gopp"
	"log"
	"strings"
	"time"

	tox "github.com/kitech/go-toxcore"
	"github.com/kitech/go-toxcore/xtox"
)

var groupbot = "56A1ADE4B65B86BCD51CC73E2CD4E542179F47959FE3E0E21B4B0ACDADE51855D34D34D37CB5"

// 帮助改进p2p网络稳定的bot列表
var nethlpbots = []string{
	groupbot, //groupbot@toxme.io
	"76518406F6A9F2217E8DC487CC783C25CC16A15EB36FF32E335A235342C48A39218F515C39A6", //echobot@toxme.io
	"7F3948BDF42F2DA68468ADA46783B392EF8ADD60E8BDE3CD04981766B5A7883747824B7108D7", //toxme@toxme.io
	"DD7A68B345E0AA918F3544AA916B5CA6AED6DE80389BFF1EF7342DACD597943D62BDEED1FC67", // my echobot
	//, // kalinaBot@toxme.io,
	//, // LainBot@toxme.io,
}

// TODO sync锁
// tox 实例的一些自动化行为整理
type toxAutoAction struct {
	delGroupC chan int // 用于删除被邀群的channel，被主tox循环使用

	theirGroups map[int]bool // accepted group number => true

	initGroupNames map[uint32]string
}

func newToxAutoAction() *toxAutoAction {
	this := &toxAutoAction{}
	this.delGroupC = make(chan int, 16)

	this.theirGroups = make(map[int]bool)

	this.initGroupNames = make(map[uint32]string)

	return this
}

var toxaa = newToxAutoAction()

// 需要与 tox iterate并列执行的函数
// 比如执行删除群的操作。这个操作如果在iterate中执行，会导致程序崩溃。
func (this *toxAutoAction) iterateTasks() {

}

// TODO accept多次会出现什么情况
// called in onGroupInvite, when accepted
func (this *toxAutoAction) onGroupInvited(groupNumber int) {
	this.theirGroups[groupNumber] = true
}

// for self connection status callback
// 在本bot上线时，自动添加几个常期在线bot，能够让本bot的网络稳定些
func autoAddNetHelperBots(t *tox.Tox, status int, d interface{}) {
	log.Println(status, tox.ConnStatusString(status))
	for _, bot := range nethlpbots {
		friendNumber, err := t.FriendByPublicKey(bot)
		if err == nil && status > tox.CONNECTION_NONE {
			if bot == groupbot {
				// t.FriendDelete(friendNumber)
				// err = errors.New("hehe")
			}
		}

		// 查找不到该好友信息，并且当前状态是连接状态
		if err != nil && status > tox.CONNECTION_NONE {
			ret, err := t.FriendAdd(bot, fmt.Sprintf("Hey %d, me here", friendNumber))
			if err != nil {
				log.Println(ret, err)
			}
		}
	}
}

/*
实现自动摘除被别人邀请，但当前只有自己在了的群组。
*/
func autoRemoveInvitedGroups(t *tox.Tox,
	groupNumber int, peerNumber int, change uint8, ud interface{}) {
	// this := toxaa

	// check only me left case
	checkOnlyMeLeftGroup(t, groupNumber, peerNumber, change)
}

// 被邀请的群组被被删除的处理
// 清缓存映射
// 尝试重新加入，因为有可能是我方掉线了。
func removedInvitedGroup(t *tox.Tox, groupNumber int) error {
	groupTitle, err := t.ConferenceGetTitle(uint32(groupNumber))
	gopp.ErrPrint(err)
	if xtox.IsInvitedGroup(t, uint32(groupNumber)) {
		log.Println("Delete invited group: ", groupNumber, groupTitle)
		delete(toxaa.theirGroups, groupNumber)
		_, err = t.ConferenceDelete(uint32(groupNumber))
		gopp.ErrPrint(err)

		// try rejoin
		tryJoinOfficalGroupbotManagedGroups(t)
	} else {
		log.Println("Self created group: don't delete:", groupNumber, groupTitle)
		// 可能也是要删除的，不过删除之后要做其他的工作
	}
	return nil
}

func checkOnlyMeLeftGroup(t *tox.Tox, groupNumber int, peerNumber int, change uint8) {
	this := toxaa

	groupTitle, err := t.GroupGetTitle(groupNumber)
	if err != nil {
		log.Println("wtf", err, groupNumber, peerNumber)
	}
	peerName, err := t.GroupPeerName(groupNumber, peerNumber)
	if err != nil {
		if change != tox.CHAT_CHANGE_PEER_DEL {
			log.Println("wtf", err, peerName)
		}
	}
	// var peerPubkey string

	switch change {
	case tox.CHAT_CHANGE_PEER_DEL:
	case tox.CHAT_CHANGE_PEER_ADD:
	case tox.CHAT_CHANGE_PEER_NAME:
	}

	// check only me left case
	if change == tox.CHAT_CHANGE_PEER_DEL {
		if pn := t.GroupNumberPeers(groupNumber); pn == 1 {
			log.Println("oh, only me left:", groupNumber, groupTitle, xtox.IsInvitedGroup(t, uint32(groupNumber)))
			// check our create group or not
			// 即使不是自己创建的群组，在只剩下自己之后，也可以不删除。因为这个群的所有人就是自己了。
			// 这里找一下为什么程序会崩溃吧
			if _, ok := this.theirGroups[groupNumber]; ok {
				log.Println("invited group matched, clean it", groupNumber, groupTitle)
				delete(this.theirGroups, groupNumber)
				grptype, err := t.GroupGetType(uint32(groupNumber))
				log.Println("before delete group chat", groupNumber, grptype, err)
				switch uint8(grptype) {
				case tox.GROUPCHAT_TYPE_AV:
					// log.Println("dont delete av groupchat for a try", groupNumber, ok, err)
				case tox.GROUPCHAT_TYPE_TEXT:
					// ok, err := this._tox.DelGroupChat(groupNumber)
					// log.Println("after delete group chat", groupNumber, ok, err)
				default:
					log.Fatal("wtf")
				}
				time.AfterFunc(1*time.Second, func() {
					this.delGroupC <- groupNumber
					// why not delete here? deadlock? crash?
				})
				log.Println("Rename....", groupTitle, makeDeletedGroupName(groupTitle))
				t.GroupSetTitle(groupNumber, makeDeletedGroupName(groupTitle))
				log.Println("dont delete invited groupchat for a try", groupNumber, ok, err)
			}
		}
	}

}

// 无用群改名相关功能
func makeDeletedGroupName(groupTitle string) string {
	return fmt.Sprintf("#deleted_invited_groupchat_%s_%s",
		time.Now().Format("20060102_150405"), groupTitle)
}

func isDeletedGroupName(groupTitle string) bool {
	return strings.HasPrefix(groupTitle, "#deleted_invited_groupchat_")
}

func getDeletedGroupName(groupTitle string) string {
	s := groupTitle[len(makeDeletedGroupName("")):]
	return s
}

func isGroupbot(pubkey string) bool { return strings.HasPrefix(groupbot, pubkey) }

// raw group name map
var fixedGroups = map[string]string{
	// "tox-en": "invite 0",
	// "Official Tox groupchat": "invite 0",
	"Tox Public Chat for beautiful ladies": "invite 0",
	// "Chinese 中文":                           "invite 1",
	// "tox-cn": "invite 1",
	// "tox-ru": "invite 3",
	// "Club Cyberia": "invite 3",
	// "Club Cyberia: No Pedos or Pervs": "invite 3",
	"test autobot": "invite 4",
	// "Russian Tox Chat (Use kalina@toxme.io or 12EDB939AA529641CE53830B518D6EB30241868EE0E5023C46A372363CAEC91C2C948AEFE4EB": "invite 5",
}

// 检测是否是固定群组
func isOfficialGroupbotManagedGroups(name string) (rname string, ok bool) {
	for n, _ := range fixedGroups {
		if name == n {
			rname = n
			return
		}
	}
	// 再次尝试采用前缀对比法
	for n, _ := range fixedGroups {
		if len(name) > 5 && strings.HasPrefix(n, name) {
			rname = n
			return
		}
	}
	return
}

// 检查自己是否在固定群中，如果不在，则尝试发送groupbot进群消息
// friend connection callback
// self connection callback
// timer callback
func tryJoinOfficalGroupbotManagedGroups(t *tox.Tox) {
	friendNumber, err := t.FriendByPublicKey(groupbot)
	gopp.ErrPrint(err)
	status, err := t.FriendGetConnectionStatus(friendNumber)
	if status == tox.CONNECTION_NONE {
		return
	}

	curGroups := make(map[string]int32)
	for _, gn := range t.GetChatList() {
		gt, _ := t.GroupGetTitle(int(gn))
		curGroups[gt] = gn
	}

	// 查找群是否是当前的某个群，相似比较
	incurrent := func(name string) bool {
		for groupTitle, gn := range curGroups {
			isInvited := xtox.IsInvitedGroup(t, uint32(gn))
			if !isInvited {
				continue
			}
			if groupTitle == name {
				return true
			}
			if strings.HasPrefix(groupTitle, name) {
				return true
			}
		}
		return false
	}

	// 不在这群，或者在这群，但只自己在了
	for name, handler := range fixedGroups {
		if !incurrent(name) {
			log.Println(name, handler)
			n, err := t.FriendSendMessage(friendNumber, handler)
			if err != nil {
				log.Println(err, n)
			}
		}
	}
}

// 尝试拉好友进固定群组
// 具体进哪些群，目前只提供进 # tox-cn 群组
// 后续如果有更强大的配置功能，可以让用户选择自动进哪些群
func tryPullFriendFixedGroups(t *tox.Tox, friendNumber uint32, status int) {

}

// groupbot's response message
func fixSpecialMessage(t *tox.Tox, friendNumber uint32, msg string) {
	pubkey, err := t.FriendGetPublicKey(friendNumber)
	if err != nil {
		log.Println(err)
	} else {
		if isGroupbot(pubkey) {
			if msg == "Group doesn't exist." {
				t.FriendSendMessage(friendNumber, "group text")
			}
		}
	}
}

///

func toxmeLookup(name string) {

}