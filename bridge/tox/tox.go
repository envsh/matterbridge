package btox

import (
	"fmt"
	"gopp"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/42wim/matterbridge/bridge"
	"github.com/42wim/matterbridge/bridge/config"
	"github.com/42wim/matterbridge/bridge/helper"
	logr "github.com/Sirupsen/logrus"
	// tox "github.com/kitech/go-toxcore"
	tox "github.com/TokTok/go-toxcore-c"
	"github.com/envsh/go-toxcore/xtox"
)

type Btox struct {
	i               *tox.Tox
	Nick            string
	names           map[string][]string
	connected       chan struct{}
	Local           chan config.Message // local queue for flood control
	FirstConnection bool
	disC            chan struct{}

	MessageDelay, MessageQueue, MessageLength int

	*bridge.Config

	store            *Storage
	frndjrman        *FriendJoinedRoomsManager
	groupPeerPubkeys sync.Map // groupNumber => []string
	brgCfgedRooms    map[string]bool
}

var flog *logr.Entry
var protocol = "tox"

func init() {
	flog = logr.WithFields(logr.Fields{"module": protocol})
}

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
}

func New(cfg *bridge.Config) bridge.Bridger {
	b := &Btox{}
	b.Config = cfg
	b.Nick = b.GetString("Nick")
	b.names = make(map[string][]string)
	b.connected = make(chan struct{})
	if b.GetInt("MessageDelay") == 0 {
		b.MessageDelay = 1300
	} else {
		b.MessageDelay = b.GetInt("MessageDelay")
	}
	if b.GetInt("MessageQueue") == 0 {
		b.MessageQueue = 30
	} else {
		b.MessageQueue = b.GetInt("MessageQueue")
	}
	if b.GetInt("MessageLength") == 0 {
		b.MessageLength = 400
	} else {
		b.MessageLength = b.GetInt("MessageLength")
	}
	b.FirstConnection = true
	b.disC = make(chan struct{}, 0)

	b.extraSetup()
	return b
}

func (b *Btox) extraSetup() {
	store := newStorage()
	b.store = store

	statusMessage := "matbrg for toxers. Send me the message 'info', 'help' for a full list of commands. code: https://github.com/envsh/matterbridge ."
	toxctx = xtox.NewToxContext("matbrg.tsbin", b.Nick, statusMessage)
	b.i = xtox.New(toxctx)
	SetAutoBotFeatures(b.i, FOTA_ADD_NET_HELP_BOTS|FOTA_REMOVE_ONLY_ME_INVITED|
		FOTA_ACCEPT_FRIEND_REQUEST|FOTA_ACCEPT_GROUP_INVITE|
		FOTA_KEEP_GROUPCHAT_TITLE)
	gn, err := b.i.ConferenceNew()
	gopp.ErrPrint(err)
	gopp.Assert(gn == 0, "first group number must be 0, but is ", gn)
	_, err = b.i.ConferenceSetTitle(gn, "TrashNoJoin")
	gopp.ErrPrint(err)
	b.initCallbacks()

	b.groupPeerPubkeys = sync.Map{}
	b.frndjrman = newFriendJoinedRoomsManager(b)
	b.frndjrman.loadConfigData()
	b.brgCfgedRooms = make(map[string]bool)

}

func (this *Btox) initConfigData() {
	this.frndjrman.loadConfigData()
}

func (b *Btox) Command(msg *config.Message) string {
	switch msg.Text {
	case "!users":
		t := b.i
		var names []string
		gns, found := xtox.ConferenceFindAll(t, msg.Channel) // TODO improve needed
		if found {
			for _, gn := range gns {
				names = t.ConferenceGetNames(gn)
				break
			}
		}
		go func() {
			b.Remote <- config.Message{Username: b.Nick,
				Text: fmt.Sprintf("There are %d users, %s currently on Tox %s",
					len(names), strings.Join(names, ", "), msg.Channel),
				Channel: msg.Channel, Account: b.Account}
		}()
	case "!ping":
		go func() {
			b.Remote <- config.Message{Username: b.Nick, Text: fmt.Sprintf("pong! on %s", msg.Channel),
				Channel: msg.Channel, Account: b.Account}
		}()
	}
	return ""
}

func (this *Btox) Send(msg config.Message) (string, error) {
	log.Printf("%+v", msg)
	if strings.HasPrefix(msg.Text, "!") {
		this.Command(&msg)
		return "", nil
	}

	if msg.Extra != nil {
		return this._SendFiles(&msg)
	}

	msg.Text = restoreUserName(msg.Username) + msg.Text
	if strings.HasSuffix(msg.Text, "currently on IRC") {
		msg.Text = msg.Username + fmt.Sprintf("There are %d users, %s", strings.Count(msg.Text, ",")+1, msg.Text)
	}

	this._SendImpl(msg.Channel, msg.Text, msg.Event)
	return "", nil
}

func (this *Btox) _SendFiles(msg *config.Message) (string, error) {
	b := this
	// Handle files
	if msg.Extra != nil {
		for _, rmsg := range helper.HandleExtra(msg, b.General) {
			// b.Local <- rmsg
			rmsg.Text = restoreUserName(rmsg.Username) + rmsg.Text
			this._SendImpl(rmsg.Channel, rmsg.Text, rmsg.Event)
		}
		if len(msg.Extra["file"]) > 0 {
			for _, f := range msg.Extra["file"] {
				fi := f.(config.FileInfo)
				if fi.Comment != "" {
					msg.Text += fi.Comment + ": "
				}
				if fi.URL != "" {
					msg.Text = fi.URL
					if fi.Comment != "" {
						msg.Text = fi.Comment + ": " + fi.URL
					}
				}
				// b.Local <- config.Message{Text: msg.Text, Username: msg.Username, Channel: msg.Channel, Event: msg.Event}
				msg.Text = restoreUserName(msg.Username) + msg.Text
				this._SendImpl(msg.Channel, msg.Text, msg.Event)
			}

			return "", nil
		}
	}
	return "", nil
}

func (this *Btox) _SendImpl(channel, msgText string, msgEvent string) {
	t := this.i
	gns, found := xtox.ConferenceFindAll(t, channel) // TODO improve needed
	if found {
		tmsgs := gopp.Splitrn(msgText, tox.MAX_MESSAGE_LENGTH-103)
		for _, gn := range gns {
			groupTitle, _ := t.ConferenceGetTitle(gn)
			for _, tmsg := range tmsgs {
				if msgEvent == config.EVENT_USER_ACTION {
					tmsg = "/me " + tmsg
					_, err := t.ConferenceSendMessage(gn, tox.MESSAGE_TYPE_ACTION, tmsg)
					gopp.ErrPrint(err, gn, groupTitle, len(tmsg))
				} else {
					_, err := t.ConferenceSendMessage(gn, tox.MESSAGE_TYPE_NORMAL, tmsg)
					gopp.ErrPrint(err, gn, groupTitle, len(tmsg))
				}
			}
		}
	} else {
		log.Println("channel not found:", channel)
	}
}

var toxctx *xtox.ToxContext

func (this *Btox) Connect() error {
	b := this
	b.Log.Infof("Connecting %s", b.GetString("Server"))
	log.Println(this.Nick, "===", this.Account, this.Config)
	xtox.Connect(this.i)
	xtox.ConnectFixed(this.i)

	go this.iterate()
	return nil
}

func (this *Btox) JoinChannel(channel config.ChannelInfo) error {
	log.Printf("%+v\n", channel)
	t := this.i

	this.brgCfgedRooms[channel.Name] = true
	// check passive group name
	if rname, ok := isOfficialGroupbotManagedGroups(channel.Name); ok {
		log.Println("It's should be invited group, don't create.", channel.Name, rname)
		return nil
	}

	//
	grptitles := xtox.ConferenceAllTitles(t)
	found := false
	gn := uint32(math.MaxUint32)
	for gn_, title := range grptitles {
		log.Println(gn_, title)
		if title == channel.Name {
			found = true
			gn = gn_
		}
	}
	if found {
		log.Println("Already exist:", gn, channel.Name)
	} else {
		var gn_ uint32
		var err error
		if channel.Options.Key == "audio" {
			gn_ = uint32(t.AddAVGroupChat())
		} else {
			gn_, err = t.ConferenceNew()
		}

		gopp.ErrPrint(err)
		t.ConferenceSetTitle(gn_, channel.Name)
		log.Println("Saving initGroupNames:", gn_, channel.Name, toxaa.initGroupNamesLen())
		toxaa.initGroupNames.LoadOrStore(gn_, channel.Name)
		log.Println("New group created:", gn_, channel.Name)
	}
	return nil
}
func (this *Btox) Disconnect() error {
	log.Println()
	this.disC <- struct{}{}
	return nil
}

//////
func (this *Btox) initCallbacks() {
	t := this.i
	removeLongtimeNoSeeHelperBots(t)
	t.CallbackSelfConnectionStatus(func(_ *tox.Tox, status int, userData interface{}) {
		log.Println(status, tox.ConnStatusString(status))
		// autoAddNetHelperBots(t, status, userData)
	}, nil)

	t.CallbackConferenceAction(func(_ *tox.Tox, groupNumber uint32, peerNumber uint32, action string, userData interface{}) {
		log.Println(groupNumber, peerNumber, action)
	}, nil)

	t.CallbackFriendStatus(func(_ *tox.Tox, friendNumber uint32, status int, userData interface{}) {
		log.Println(friendNumber, status)
	}, nil)
	t.CallbackFriendConnectionStatus(func(_ *tox.Tox, friendNumber uint32, status int, userData interface{}) {
		name, _ := t.FriendGetName(friendNumber)
		log.Println(friendNumber, status, tox.ConnStatusString(status), name)
		// t.ConferenceInvite(friendNumber, uint32(0))
		tryJoinOfficalGroupbotManagedGroups(t)
		this.tryInviteFriendToGroups(friendNumber, status)
		if status > 0 && isGroupbotByNum(t, friendNumber) {
			log.Println("sending cmd: info", friendNumber, status, tox.ConnStatusString(status), name)
			_, err := t.FriendSendMessage(friendNumber, "info") // for tryFixGroupbotGroupInviteCmd
			gopp.ErrPrint(err)
			this.StateMachineEvent("FriendConnectionStatus", friendNumber, status)
		}
	}, nil)

	t.CallbackFriendMessage(func(_ *tox.Tox, friendNumber uint32, msg string, userData interface{}) {
		// log.Println(friendNumber, msg)
		pubkey, err := t.FriendGetPublicKey(friendNumber)
		gopp.ErrPrint(err)
		_ = pubkey
		friendName, err := t.FriendGetName(friendNumber)
		gopp.ErrPrint(err)
		_ = friendName

		this.processFriendCmd(friendNumber, msg)
		if isGroupbotByNum(t, friendNumber) {
			tryFixGroupbotGroupInviteCmd(t, msg)
			this.StateMachineEvent("FriendMessage", friendNumber, msg)
		}
	}, nil)

	t.CallbackConferenceMessage(func(_ *tox.Tox, groupNumber uint32, peerNumber uint32, message string, userData interface{}) {
		log.Println(groupNumber, peerNumber, message)
		if this.processChannelCmd(groupNumber, peerNumber, message) {
			return
		}
		peerPubkey, err := t.ConferencePeerGetPublicKey(groupNumber, peerNumber)
		gopp.ErrPrint(err)
		if strings.HasPrefix(t.SelfGetAddress(), peerPubkey) {
			return
		}
		peerName, err := t.ConferencePeerGetName(groupNumber, peerNumber)
		gopp.ErrPrint(err)
		groupTitle, err := t.ConferenceGetTitle(groupNumber)
		filterTopic := fmt.Sprintf("%s@%s", peerPubkey, groupTitle)
		if err := this.IsFiltered(filterTopic, message); err != nil {
			gopp.ErrPrint(err, peerName, groupTitle)
			return
		}
		rmsg := config.Message{Username: peerName, Channel: groupTitle, Account: this.Account, UserID: peerPubkey}
		rmsg.Protocol = protocol
		rmsg.Text = message
		log.Printf("Sending message(%d) from %s on %s to gateway\n", len(message), groupTitle, this.Account)
		this.Remote <- rmsg
	}, nil)

	t.CallbackFriendRequest(func(_ *tox.Tox, pubkey string, message string, userData interface{}) {
		// _, err := t.FriendAddNorequest(pubkey)
		// gopp.ErrPrint(err)
	}, nil)

	t.CallbackConferenceInvite(func(_ *tox.Tox, friendNumber uint32, itype uint8, cookie string, userData interface{}) {
		log.Println(friendNumber, itype)
		/*
			var gn uint32
			var err error
			switch itype {
			case tox.CONFERENCE_TYPE_TEXT:
				gn, err = t.ConferenceJoin(friendNumber, cookie)
				gopp.ErrPrint(err)
			case tox.CONFERENCE_TYPE_AV:
				gn_, err_ := t.JoinAVGroupChat(friendNumber, cookie)
				gn, err = uint32(gn_), err_
			}
			// 在刚Join的group是无法获得title的
			if false {
				groupTitle, err := t.ConferenceGetTitle(gn)
				gopp.ErrPrint(err)
				log.Println(gn, groupTitle)
			}
			toxaa.onGroupInvited(int(gn))
		*/
	}, nil)
	t.CallbackConferenceInviteAdd(func(_ *tox.Tox, friendNumber uint32, itype uint8, cookie string, userData interface{}) {
		log.Println(friendNumber, itype)
	}, nil)

	t.CallbackConferencePeerNameAdd(func(_ *tox.Tox, groupNumber uint32, peerNumber uint32, name string, userData interface{}) {

	}, nil)
	t.CallbackConferencePeerListChangedAdd(func(_ *tox.Tox, groupNumber uint32, userData interface{}) {
		pubkeysx, found := this.groupPeerPubkeys.Load(groupNumber)
		if !found {
			pubkeysx = []interface{}{}
		}
		newPubkeys := t.ConferenceGetPeerPubkeys(groupNumber)
		added, deleted := DiffSlice(pubkeysx, newPubkeys)

		for _, pubkeyx := range added {
			this.updatePeerState2(groupNumber, pubkeyx.(string), xtox.CHAT_CHANGE_PEER_ADD)
		}
		for _, pubkeyx := range deleted {
			this.updatePeerState2(groupNumber, pubkeyx.(string), xtox.CHAT_CHANGE_PEER_DEL)
		}
		if len(deleted) > 0 {
			checkOnlyMeLeftGroup(t, groupNumber, 0, xtox.CHAT_CHANGE_PEER_DEL)
		}
		this.groupPeerPubkeys.Store(groupNumber, newPubkeys)
		if len(deleted) > 0 {
			this.StateMachineEvent("ConferencePeerListChange", groupNumber)
		}
	}, nil)

	/*
		t.CallbackConferenceNameListChange(func(_ *tox.Tox, groupNumber uint32, peerNumber uint32, change uint8, userData interface{}) {
			this.updatePeerState(groupNumber, peerNumber, change)
			checkOnlyMeLeftGroup(t, int(groupNumber), int(peerNumber), change)
		}, nil)
	*/
	t.CallbackConferenceTitle(func(_ *tox.Tox, groupNumber uint32, peerNumber uint32, title string, userData interface{}) {
		// 防止其他用户修改标题
		// tryKeepGroupTitle(t, groupNumber, peerNumber, title)
	}, nil)
}

// should block
func (this *Btox) iterate() {
	stop := false
	tick := time.NewTicker(1 * time.Second / 5)
	tick2 := time.NewTicker(5 * time.Second)  // for tryJoin
	tick3 := time.NewTicker(15 * time.Second) // for joined room manager

	defer tick.Stop()
	defer tick2.Stop()
	defer tick3.Stop()

	for !stop {
		select {
		case <-tick.C:
			this.i.Iterate2(nil)
		case <-this.disC:
			stop = true
		case <-tick2.C:
			tryJoinOfficalGroupbotManagedGroups(this.i)
		case <-tick3.C:
			this.checkFriendInRoomOrInvite()
		case gn := <-toxaa.delGroupC:
			t := this.i
			isInvited := xtox.IsInvitedGroup(t, uint32(gn))
			if isInvited {
			}
			removedInvitedGroup(t, gn)
			if isInvited {
				// tryJoinOfficalGroupbotManagedGroups(t, friendNumber uint32, status int)
			} else {
				// re create and re pull friend to join in
			}
		}
	}
	log.Println("disconnected")
}
