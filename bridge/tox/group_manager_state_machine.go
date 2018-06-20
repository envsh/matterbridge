package btox

import (
	"fmt"
	"gopp"
	"log"
	"strings"

	funk "github.com/thoas/go-funk"
)

const (
	STM_INIT = iota
	STM_WAIT_RESP_INFO
	STM_INVITE_X_DONE_1
	STM_WAIT_RESP_GROUP_CREATED
	STM_RESP_GROUP_CREATED
	STM_INVITE_X2 // tmp group #xxx
	STM_DONE
)

type StmState struct {
	state        int
	need_creates []string
	creating     string
}

var stms = &StmState{STM_INIT, make([]string, 0), ""}
var stm_state = STM_INIT
var stm_need_creates = make([]string, 0)

// 2个无法解决的问题，无法一次性拿到groupbot的群组列表
// 无法确定新创建的群组的改名时间
func (this *Btox) StateMachineEvent(evt string, args ...interface{}) {
	if true {
		return // TODO
	}
	switch evt {
	case "FriendConnectionStatus": // groupbot online
		switch stm_state {
		case STM_INIT:
			log.Println("semd cmd info...", args[0])
			msg := fmt.Sprintf("info")
			_, err := this.i.FriendSendMessage(args[0].(uint32), msg)
			gopp.ErrPrint(err)
			stm_state = STM_WAIT_RESP_INFO
		default:
			log.Println("no care state pair:", evt, stm_state)
		}
	case "FriendMessage":
		friendNumber := args[0].(uint32)
		msg := args[1].(string)
		log.Println(evt, args)
		switch stm_state {
		case STM_WAIT_RESP_INFO:
			if messageIsInfoResp(msg) {
				this.sendCommandInvitexs(friendNumber, msg)
				stm_state = STM_INVITE_X_DONE_1
				this.prepareCreateGroups(friendNumber, msg)
				this.sendCommandCreateGroup(friendNumber)
			}
		case STM_WAIT_RESP_GROUP_CREATED:
			if strings.HasPrefix(args[1].(string), "Group chat ") {
				log.Println(stm_state, creating_group_name, msg)
				stm_state = STM_RESP_GROUP_CREATED
			}
		default:
			log.Println("no care state pair:", evt, stm_state)
		}
	case "ConferencePeerListChange":
		groupNumber := args[0].(uint32)
		title, err := this.i.ConferenceGetTitle(groupNumber)
		gopp.ErrPrint(err, groupNumber)
		if strings.HasPrefix(title, "Groupchat #") {
			curnum := title[11:]
			if curnum == creating_group_number {
				this.i.ConferenceSetTitle(groupNumber, creating_group_name)
				friendNumber, err := this.i.FriendByPublicKey(groupbot)
				gopp.ErrPrint(err, groupbot)
				this.sendCommandCreateGroup(friendNumber)
			}
		}
	default:
		log.Println("no impl evt:", evt)
	}
}

func (this *Btox) sendCommandInvitexs(friendNumber uint32, msg string) {
	egrps := parseInfoResp(msg)

	existsGroupsx := funk.Values(egrps)
	for title, _ := range this.brgCfgedRooms {
		if !funk.Contains(existsGroupsx, title) {
			// log.Println("matbrg configed but not exist group, creating:", title)
		} else {
			if _, ok := isOfficialGroupbotManagedGroups(title); ok {
			}
			gnum := egrps[title]
			log.Println("reverse invite ", gnum, title)
			this.i.FriendSendMessage(friendNumber, fmt.Sprintf("invite %s", gnum))
		}
	}
}

func parseInfoResp(msg string) map[string]string {
	egrps := make(map[string]string)
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Group ") {
			fields := strings.Split(line, "|")
			gnum := strings.Split(fields[0], " ")[1]
			gtitle := fields[3][8:]
			egrps[gtitle] = gnum
		}
	}
	log.Println(egrps)
	return egrps
}

func (this *Btox) prepareCreateGroups(friendNumber uint32, msg string) {
	egrps := parseInfoResp(msg)

	stm_need_creates = make([]string, 0)
	existsGroupsx := funk.Values(egrps)
	for title, _ := range this.brgCfgedRooms {
		if !funk.Contains(existsGroupsx, title) {
			log.Println("matbrg configed but not exist group, creating:", title)
			stm_need_creates = append(stm_need_creates, title)
		}
	}
	log.Println("prepared need create groups:", len(stm_need_creates))
}

var creating_group_name string
var creating_group_number string

func (this *Btox) sendCommandCreateGroup(friendNumber uint32) {
	if len(stm_need_creates) == 0 {
		log.Println("Create group in groupbot done.")
		stm_state = STM_DONE
		return
	}
	title := stm_need_creates[0]
	stm_need_creates = stm_need_creates[1:]
	creating_group_name = title
	stm_state = STM_WAIT_RESP_GROUP_CREATED
	log.Println("Creating group in groupbot:", title)
	_, err := this.i.FriendSendMessage(friendNumber, fmt.Sprintf("group text"))
	gopp.ErrPrint(err, title)
}

func (this *Btox) sendCommandInvitx2(friendNumber uint32, msg string) {

	fields := strings.Split(msg, " ")
	gnum_s := fields[2]

	creating_group_number = gnum_s
	stm_state = STM_INVITE_X2
	log.Println("reverse invite tmp group:", gnum_s, creating_group_name)
	_, err := this.i.FriendSendMessage(friendNumber, fmt.Sprintf("invite %s", gnum_s))
	gopp.ErrPrint(err, msg)
}

func messageIsInfoResp(msg string) bool {
	lines := strings.Split(msg, "\n")
	if len(lines) >= 3 {
		if strings.HasPrefix(lines[0], "Uptime: ") &&
			strings.HasPrefix(lines[1], "Friends: ") &&
			strings.HasPrefix(lines[2], "Inactive friends are purged after 365 days") {
			return true
		}
	}
	return false
}
