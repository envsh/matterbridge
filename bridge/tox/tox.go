package btox

import (
	"gopp"
	"log"
	"math"
	"strings"
	"time"

	"github.com/42wim/matterbridge/bridge/config"
	// log "github.com/Sirupsen/logrus"
	tox "github.com/kitech/go-toxcore"
	"github.com/kitech/go-toxcore/xtox"
)

type Btox struct {
	i               *tox.Tox
	Nick            string
	names           map[string][]string
	Config          *config.Protocol
	Remote          chan config.Message
	connected       chan struct{}
	Local           chan config.Message // local queue for flood control
	Account         string
	FirstConnection bool
	disC            chan struct{}

	store *Storage
}

// var flog *log.Entry
var protocol = "tox"

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	// flog = log.WithFields(log.Fields{"module": protocol})
}

func New(cfg config.Protocol, account string, c chan config.Message) *Btox {
	b := &Btox{}
	b.Config = &cfg
	b.Nick = b.Config.Nick
	b.Remote = c
	b.names = make(map[string][]string)
	b.Account = account
	b.connected = make(chan struct{})
	if b.Config.MessageDelay == 0 {
		b.Config.MessageDelay = 1300
	}
	if b.Config.MessageQueue == 0 {
		b.Config.MessageQueue = 30
	}
	if b.Config.MessageLength == 0 {
		b.Config.MessageLength = 400
	}
	b.FirstConnection = true
	b.disC = make(chan struct{}, 0)

	store := newStorage()
	b.store = store

	toxctx = xtox.NewToxContext("matbrg.tsbin", b.Nick, "matbrg for tox")
	b.i = xtox.New(toxctx)
	b.initCallbacks()
	return b
}

func (this *Btox) Send(msg config.Message) (string, error) {
	log.Printf("%+v", msg)
	t := this.i
	gns, found := xtox.ConferenceFindAll(t, msg.Channel)
	if found {
		tmsg := msg.Username + msg.Text
		for _, gn := range gns {
			groupTitle, _ := t.ConferenceGetTitle(gn)
			if msg.Event == config.EVENT_USER_ACTION {
				tmsg = "/me " + tmsg
				_, err := t.ConferenceSendMessage(gn, tox.MESSAGE_TYPE_ACTION, tmsg)
				gopp.ErrPrint(err, gn, groupTitle)
			} else {
				_, err := t.ConferenceSendMessage(gn, tox.MESSAGE_TYPE_NORMAL, tmsg)
				gopp.ErrPrint(err, gn, groupTitle)
			}
		}
	} else {
		log.Println("not found:", msg.Channel)
	}
	return "", nil
}

var toxctx *xtox.ToxContext

func (this *Btox) Connect() error {
	log.Println(this.Nick, "===", this.Account, this.Config)
	xtox.Connect(this.i)

	go this.iterate()
	return nil
}
func (this *Btox) JoinChannel(channel config.ChannelInfo) error {
	log.Printf("%+v\n", channel)
	t := this.i

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
		gn_, err := t.ConferenceNew()
		gopp.ErrPrint(err)
		t.ConferenceSetTitle(gn_, channel.Name)
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
	t.CallbackSelfConnectionStatus(func(_ *tox.Tox, status int, userData interface{}) {
		log.Println(status, tox.ConnStatusString(status))
		autoAddNetHelperBots(t, status, userData)
	}, nil)

	t.CallbackConferenceAction(func(_ *tox.Tox, groupNumber uint32, peerNumber uint32, action string, userData interface{}) {
		log.Println(groupNumber, peerNumber, action)
	}, nil)

	t.CallbackFriendStatus(func(_ *tox.Tox, friendNumber uint32, status int, userData interface{}) {
		log.Println(friendNumber, status)
	}, nil)
	t.CallbackFriendConnectionStatus(func(_ *tox.Tox, friendNumber uint32, status int, userData interface{}) {
		log.Println(friendNumber, status, tox.ConnStatusString(status))
		// t.ConferenceInvite(friendNumber, uint32(0))
		tryJoinOfficalGroupbotManagedGroups(t)
	}, nil)

	t.CallbackFriendMessage(func(_ *tox.Tox, friendNumber uint32, msg string, userData interface{}) {
		log.Println(friendNumber, msg)
		pubkey, err := t.FriendGetPublicKey(friendNumber)
		gopp.ErrPrint(err)
		_ = pubkey
		friendName, err := t.FriendGetName(friendNumber)
		gopp.ErrPrint(err)
		_ = friendName

		this.processFriendCmd(friendNumber, msg)
	}, nil)

	t.CallbackConferenceMessage(func(_ *tox.Tox, groupNumber uint32, peerNumber uint32, message string, userData interface{}) {
		log.Println(groupNumber, peerNumber, message)
		peerPubkey, err := t.ConferencePeerGetPublicKey(groupNumber, peerNumber)
		gopp.ErrPrint(err)
		if strings.HasPrefix(t.SelfGetAddress(), peerPubkey) {
			return
		}
		peerName, err := t.ConferencePeerGetName(groupNumber, peerNumber)
		gopp.ErrPrint(err)
		groupTitle, err := t.ConferenceGetTitle(groupNumber)
		rmsg := config.Message{Username: peerName, Channel: groupTitle, Account: this.Account, UserID: peerPubkey}
		rmsg.Protocol = protocol
		rmsg.Text = message
		log.Printf("Sending message from %s on %s to gateway\n", groupTitle, this.Account)
		this.Remote <- rmsg
	}, nil)

	t.CallbackFriendRequest(func(_ *tox.Tox, pubkey string, message string, userData interface{}) {
		_, err := t.FriendAddNorequest(pubkey)
		gopp.ErrPrint(err)
	}, nil)

	t.CallbackConferenceInvite(func(_ *tox.Tox, friendNumber uint32, itype uint8, data []byte, userData interface{}) {
		log.Println(friendNumber, itype)
		var gn uint32
		var err error
		switch int(itype) {
		case tox.CONFERENCE_TYPE_TEXT:
			gn, err = t.ConferenceJoin(friendNumber, data)
			gopp.ErrPrint(err)
		case tox.CONFERENCE_TYPE_AV:
			gn_, err_ := t.JoinAVGroupChat(friendNumber, data)
			gn, err = uint32(gn_), err_
		}
		toxaa.onGroupInvited(int(gn))
	}, nil)

	t.CallbackConferenceNameListChange(func(_ *tox.Tox, groupNumber uint32, peerNumber uint32, change uint8, userData interface{}) {
		checkOnlyMeLeftGroup(t, int(groupNumber), int(peerNumber), change)
	}, nil)

	t.CallbackConferenceTitle(func(_ *tox.Tox, groupNumber uint32, peerNumber uint32, title string, userData interface{}) {
		// TODO 防止其他用户修改标题
		peerPubkey, err := t.ConferencePeerGetPublicKey(groupNumber, peerNumber)
		gopp.ErrPrint(err)
		if peerPubkey != t.SelfGetPublicKey() {
			// restore initilized group title
		}
	}, nil)
}

// should block
func (this *Btox) iterate() {
	stop := false
	tick := time.NewTicker(1 * time.Second / 5)
	tick2 := time.NewTicker(5 * time.Second) // for tryJoin
	for !stop {
		select {
		case <-tick.C:
			this.i.Iterate2(nil)
		case <-this.disC:
			stop = true
		case <-tick2.C:
			tryJoinOfficalGroupbotManagedGroups(this.i)
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