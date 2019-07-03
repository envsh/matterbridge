package birc

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	irc "github.com/42wim/go-ircevent"
	"github.com/42wim/matterbridge/bridge"
	"github.com/42wim/matterbridge/bridge/config"
	log "github.com/Sirupsen/logrus"
	"github.com/paulrosania/go-charset/charset"
	_ "github.com/paulrosania/go-charset/data"
	"github.com/saintfish/chardet"
	ircm "github.com/sorcix/irc"
)

type Birc struct {
	i     *irc.Connection
	Nick  string
	names map[string][]string

	*bridge.Config

	connected       chan struct{}
	Local           chan config.Message // local queue for flood control
	FirstConnection bool

	MessageDelay, MessageQueue, MessageLength int
}

var protocol = "irc"

func New(cfg *bridge.Config) bridge.Bridger {
	b := &Birc{}
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
	return b
}

func (b *Birc) Command(msg *config.Message) string {
	switch msg.Text {
	case "!users":
		b.i.AddCallback(ircm.RPL_NAMREPLY, b.storeNames)
		b.i.AddCallback(ircm.RPL_ENDOFNAMES, b.endNames)
		b.i.SendRaw("NAMES " + msg.Channel)
	case "!ping":
		go func() {
			b.Remote <- config.Message{Username: b.Nick, Text: fmt.Sprintf("pong! on %s", msg.Channel),
				Channel: msg.Channel, Account: b.Account}
		}()
	}
	return ""
}

func (b *Birc) Connect() error {
	b.Local = make(chan config.Message, b.MessageQueue+10)
	b.Log.Infof("Connecting %s %s %s", b.GetString("Server"), "nick:", b.Nick)
	server, portstr, err := net.SplitHostPort(b.GetString("Server"))
	if err != nil {
		return err
	}
	i := irc.IRC(b.Nick, b.Nick)
	if log.GetLevel() == log.DebugLevel {
		i.Debug = true
	}
	b.Log.Infoln(b == nil, i == nil)
	if i == nil {
		err := fmt.Errorf("Why irc.IRC is nil? %s %s", b.GetString("Server"), b.Nick)
		b.Log.Errorln(err.Error())
		return err
	}
	i.UseTLS = b.GetBool("UseTLS")
	i.UseSASL = b.GetBool("UseSASL")
	i.SASLLogin = b.GetString("NickServNick")
	i.SASLPassword = b.GetString("NickServPassword")
	i.TLSConfig = &tls.Config{InsecureSkipVerify: b.GetBool("SkipTLSVerify"), ServerName: server}
	i.KeepAlive = time.Minute
	i.PingFreq = time.Minute
	i.Password = b.GetString("Password")

	i.AddCallback(ircm.RPL_WELCOME, b.handleNewConnection)
	i.AddCallback(ircm.RPL_ENDOFMOTD, b.handleOtherAuth)
	err = i.Connect(server + ":" + portstr)
	if err != nil {
		return err
	}
	b.i = i
	select {
	case <-b.connected:
		b.Log.Info("Connection succeeded")
	case <-time.After(time.Second * 30):
		return fmt.Errorf("connection timed out")
	}
	i.Debug = false
	// clear on reconnects
	i.ClearCallback(ircm.RPL_WELCOME)
	i.AddCallback(ircm.RPL_WELCOME, func(event *irc.Event) {
		b.Remote <- config.Message{Username: "system", Text: "rejoin", Channel: "", Account: b.Account, Event: config.EventRejoinChannels}
		// set our correct nick on reconnect if necessary
		if event.Nick == "" {
			b.Log.Infof("Invalid or empty nick ", b.Nick)
		} else {
			b.Nick = event.Nick
		}
	})
	go i.Loop()
	go b.doSend()
	return nil
}

func (b *Birc) Disconnect() error {
	b.Log.Infof("Disconnecting server: %s nick: %s", b.GetString("Server"), b.Nick)
	defer func() {
		if err := recover(); err != nil {
			var cok bool
			if b.i != nil {
				cok = b.i.Connected()
			}
			b.Log.Errorln("somepanic", err, b.i == nil, cok)
		}
	}()
	b.clearCallbacks()
	b.i.Disconnect()
	if b.i.Connected() {
		b.Log.Warnf("Must not connected server: %s nick: %s", b.GetString("Server"), b.Nick)
	}
	close(b.Local)
	return nil
}
func (b *Birc) clearCallbacks() {
	i := b.i
	if i == nil {
		return
	}
	i.ClearCallback(ircm.RPL_WELCOME)
	i.ClearCallback(ircm.RPL_ENDOFMOTD)
	i.ClearCallback(ircm.RPL_NAMREPLY)
	i.ClearCallback(ircm.RPL_ENDOFNAMES)

	i.ClearCallback("PRIVMSG")
	i.ClearCallback("CTCP_ACTION")
	i.ClearCallback(ircm.RPL_TOPICWHOTIME)
	i.ClearCallback(ircm.NOTICE)
	i.ClearCallback(ircm.RPL_MYINFO)
	i.ClearCallback("PING")
	i.ClearCallback("JOIN")
	i.ClearCallback("PART")
	i.ClearCallback("QUIT")
	i.ClearCallback("KICK")
	i.ClearCallback("*")
}

func (b *Birc) JoinChannel(channel config.ChannelInfo) error {
	if channel.Options.Key != "" {
		b.Log.Debugf("using key %s for channel %s", channel.Options.Key, channel.Name)
		b.i.Join(channel.Name + " " + channel.Options.Key)
	} else {
		b.i.Join(channel.Name)
	}
	return nil
}

func (b *Birc) Send(msg config.Message) (string, error) {
	// ignore delete messages
	if msg.Event == config.EventMsgDelete {
		return "", nil
	}
	b.Log.Debugf("Receiving %#v", msg)
	if strings.HasPrefix(msg.Text, "!") {
		b.Command(&msg)
		return "", nil
	}

	if b.GetString("Charset") != "" {
		buf := new(bytes.Buffer)
		w, err := charset.NewWriter(b.GetString("Charset"), buf)
		if err != nil {
			b.Log.Errorf("charset from utf-8 conversion failed: %s", err)
			return "", err
		}
		fmt.Fprintf(w, msg.Text)
		w.Close()
		msg.Text = buf.String()
	}

	for _, text := range strings.Split(msg.Text, "\n") {
		if len(text) > b.MessageLength {
			text = text[:b.MessageLength] + " <message clipped>"
		}
		if len(b.Local) < b.MessageQueue {
			if len(b.Local) == b.MessageQueue-1 {
				text = text + " <message clipped>"
			}
			b.Local <- config.Message{Text: text, Username: msg.Username, Channel: msg.Channel, Event: msg.Event}
		} else {
			b.Log.Debugf("flooding, dropping message (queue at %d)", len(b.Local))
		}
	}
	return "", nil
}

func (b *Birc) doSend() {
	rate := time.Millisecond * time.Duration(b.MessageDelay)
	throttle := time.NewTicker(rate)
	for msg := range b.Local {
		<-throttle.C
		if msg.Event == config.EventUserAction {
			b.i.Action(msg.Channel, msg.Username+msg.Text)
		} else {
			b.i.Privmsg(msg.Channel, msg.Username+msg.Text)
		}
	}
}

func (b *Birc) endNames(event *irc.Event) {
	channel := event.Arguments[1]
	sort.Strings(b.names[channel])
	maxNamesPerPost := (300 / b.nicksPerRow()) * b.nicksPerRow()
	continued := false
	for len(b.names[channel]) > maxNamesPerPost {
		b.Remote <- config.Message{Username: b.Nick, Text: b.formatnicks(b.names[channel][0:maxNamesPerPost], continued),
			Channel: channel, Account: b.Account}
		b.names[channel] = b.names[channel][maxNamesPerPost:]
		continued = true
	}
	b.Remote <- config.Message{Username: b.Nick, Text: b.formatnicks(b.names[channel], continued),
		Channel: channel, Account: b.Account}
	b.names[channel] = nil
	b.i.ClearCallback(ircm.RPL_NAMREPLY)
	b.i.ClearCallback(ircm.RPL_ENDOFNAMES)
}

func (b *Birc) handleNewConnection(event *irc.Event) {
	b.Log.Debug("Registering callbacks")
	i := b.i
	b.Nick = event.Arguments[0]
	i.AddCallback("PRIVMSG", b.handlePrivMsg)
	i.AddCallback("CTCP_ACTION", b.handlePrivMsg)
	i.AddCallback(ircm.RPL_TOPICWHOTIME, b.handleTopicWhoTime)
	i.AddCallback(ircm.NOTICE, b.handleNotice)
	//i.AddCallback(ircm.RPL_MYINFO, func(e *irc.Event) { b.Log.Infof("%s: %s", e.Code, strings.Join(e.Arguments[1:], " ")) })
	i.AddCallback("PING", func(e *irc.Event) {
		i.SendRaw("PONG :" + e.Message())
		b.Log.Debugf("PING/PONG")
	})
	i.AddCallback("JOIN", b.handleJoinPart)
	i.AddCallback("PART", b.handleJoinPart)
	i.AddCallback("QUIT", b.handleJoinPart)
	i.AddCallback("KICK", b.handleJoinPart)
	i.AddCallback("*", b.handleOther)
	// we are now fully connected
	b.connected <- struct{}{}
}

func (b *Birc) handleJoinPart(event *irc.Event) {
	channel := event.Arguments[0]
	if event.Code == "KICK" {
		b.Log.Infof("Got kicked from %s by %s", channel, event.Nick)
		b.Remote <- config.Message{Username: "system", Text: "rejoin", Channel: channel, Account: b.Account, Event: config.EventRejoinChannels}
		return
	}
	if event.Code == "QUIT" {
		if event.Nick == b.Nick && strings.Contains(event.Raw, "Ping timeout") {
			b.Log.Infof("%s reconnecting ..", b.Account)
			b.Remote <- config.Message{Username: "system", Text: "reconnect", Channel: channel, Account: b.Account, Event: config.EventFailure}
			return
		}
	}
	if event.Nick != b.Nick {
		b.Log.Debugf("Sending JOIN_LEAVE event from %s to gateway", b.Account)
		b.Remote <- config.Message{Username: "system", Text: event.Nick + " " + strings.ToLower(event.Code) + "s", Channel: channel, Account: b.Account, Event: config.EventJoinLeave}
		return
	}
	b.Log.Debugf("handle %#v", event)
}

func (b *Birc) handleNotice(event *irc.Event) {
	if strings.Contains(event.Message(), "This nickname is registered") && event.Nick == b.GetString("NickServNick") {
		b.i.Privmsg(b.GetString("NickServNick"), "IDENTIFY "+b.GetString("NickServPassword"))
	} else {
		b.handlePrivMsg(event)
	}
}

func (b *Birc) handleOther(event *irc.Event) {
	switch event.Code {
	case "372", "375", "376", "250", "251", "252", "253", "254", "255", "265", "266", "002", "003", "004", "005":
		return
	}
	b.Log.Debugf("%#v", event.Raw)
}

func (b *Birc) handleOtherAuth(event *irc.Event) {
	if strings.EqualFold(b.GetString("NickServNick"), "Q@CServe.quakenet.org") {
		b.Log.Debugf("Authenticating %s against %s", b.GetString("NickServUsername"), b.GetString("NickServNick"))
		b.i.Privmsg(b.GetString("NickServNick"), "AUTH "+b.GetString("NickServUsername")+" "+b.GetString("NickServPassword"))
	}
}

func (b *Birc) handlePrivMsg(event *irc.Event) {
	b.Nick = b.i.GetNick()
	// freenode doesn't send 001 as first reply
	if event.Code == "NOTICE" {
		return
	}
	// don't forward queries to the bot
	if event.Arguments[0] == b.Nick {
		return
	}
	// don't forward message from ourself
	if event.Nick == b.Nick {
		return
	}
	rmsg := config.Message{Username: event.Nick, Channel: event.Arguments[0], Account: b.Account, UserID: event.User + "@" + event.Host}
	b.Log.Debugf("handlePrivMsg() %s %s %#v", event.Nick, event.Message(), event)
	msg := ""
	if event.Code == "CTCP_ACTION" {
		//	msg = event.Nick + " "
		rmsg.Event = config.EventUserAction
	}
	msg += event.Message()
	// strip IRC colors
	re := regexp.MustCompile(`[[:cntrl:]](?:\d{1,2}(?:,\d{1,2})?)?`)
	msg = re.ReplaceAllString(msg, "")
	var r io.Reader
	var err error
	mycharset := b.GetString("Charset")
	if mycharset == "" {
		// detect what were sending so that we convert it to utf-8
		detector := chardet.NewTextDetector()
		result, err := detector.DetectBest([]byte(msg))
		if err != nil {
			b.Log.Infof("detection failed for msg: %#v", msg)
			return
		}
		b.Log.Debugf("detected %s confidence %#v", result.Charset, result.Confidence)
		mycharset = result.Charset
		// if we're not sure, just pick ISO-8859-1
		if result.Confidence < 80 {
			mycharset = "ISO-8859-1"
		}
	}
	r, err = charset.NewReader(mycharset, strings.NewReader(msg))
	if err != nil {
		b.Log.Errorf("charset to utf-8 conversion failed: %s", err)
		return
	}
	output, _ := ioutil.ReadAll(r)
	msg = string(output)
	b.Log.Debugf("Sending message from %s on %s to gateway", event.Arguments[0], b.Account)
	rmsg.Text = msg
	b.Remote <- rmsg
}

func (b *Birc) handleTopicWhoTime(event *irc.Event) {
	parts := strings.Split(event.Arguments[2], "!")
	t, err := strconv.ParseInt(event.Arguments[3], 10, 64)
	if err != nil {
		b.Log.Errorf("Invalid time stamp: %s", event.Arguments[3])
	}
	user := parts[0]
	if len(parts) > 1 {
		user += " [" + parts[1] + "]"
	}
	b.Log.Debugf("%s: Topic set by %s [%s]", event.Code, user, time.Unix(t, 0))
}

func (b *Birc) nicksPerRow() int {
	return 4
	/*
		if b.Config.Mattermost.NicksPerRow < 1 {
			return 4
		}
		return b.Config.Mattermost.NicksPerRow
	*/
}

func (b *Birc) storeNames(event *irc.Event) {
	channel := event.Arguments[2]
	b.names[channel] = append(
		b.names[channel],
		strings.Split(strings.TrimSpace(event.Message()), " ")...)
}

func (b *Birc) formatnicks(nicks []string, continued bool) string {
	return plainformatter(nicks, b.nicksPerRow())
	/*
		switch b.Config.Mattermost.NickFormatter {
		case "table":
			return tableformatter(nicks, b.nicksPerRow(), continued)
		default:
			return plainformatter(nicks, b.nicksPerRow())
		}
	*/
}
