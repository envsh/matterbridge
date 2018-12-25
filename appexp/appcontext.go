package appexp

import (
	"gopp"
	olog "log"
	"time"

	"github.com/42wim/matterbridge/bridge/config"
	"github.com/42wim/matterbridge/gateway"
	log "github.com/sirupsen/logrus"
)

func init() {
	olog.SetFlags(olog.Flags() | olog.Lshortfile)
	log.StandardLogger().AddHook(&ContextHook{})
}

// usage:
/*
matterbridge.go:57
       AppcontextMain(cfg, r)
*/
func StartAppContext(c config.Config, r *gateway.Router) {
	appctx = &AppContext{c, r}
	appctx.Start()
}

var appctx *AppContext

type AppContext struct {
	Cfg    config.Config
	Router *gateway.Router
}

func (this *AppContext) Start() {
	go this.run()
}
func (this *AppContext) run() {
	time.Sleep(1 * time.Second)

	return
	this.test_dyncfg()
	return
	//
	msg := config.Message{}
	msg.Event = config.EventFailure
	msg.ID = "123"
	msg.Account = "name1"
	this.Router.Message <- msg
}

func (this *AppContext) test_dyncfg() {
	// add bridge
	{
		cfgstr := `[irc]
[irc.name1]
Server="irc.freenode.net:6697"
UseTLS=true
Nick="matterbot1"
Password=""
`
		AddBridge1(this.Router, cfgstr)
	}
	{
		cfgstr := `[irc]
[irc.name2]
Server="irc.freenode.net:6697"
UseTLS=true
Nick="matterbot2"
Password=""
`
		AddBridge1(this.Router, cfgstr)
	}

	// add gateway
	{
		cfgstr := `[[gateway]]
name="gateway1"
enable=true
[[gateway.inout]]
account="irc.name1"
channel="#testing1"
[[gateway.inout]]
account="irc.name2"
channel="#testing2"
`
		err := AddGateway1(this.Router, cfgstr)
		gopp.ErrPrint(err)
	}
}
