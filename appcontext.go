package main

import (
	olog "log"
	"time"

	"github.com/42wim/matterbridge/bridge/config"
	"github.com/42wim/matterbridge/gateway"
	log "github.com/sirupsen/logrus"
	funk "github.com/thoas/go-funk"
)

func init() {
	olog.SetFlags(olog.Flags() | olog.Lshortfile)
	log.StandardLogger().AddHook(&ContextHook{})
}

// usage:
/*
matterbridge.go:57
       appctx = &AppContext{cfg, r}
       appctx.Start()
*/
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

	// add bridge
	{
		cfgstr := `[irc]
[irc.name1]
Server="irc.freenode.net:6667"
Password=""
`
		ucfg := config.NewConfigFromString([]byte(cfgstr))
		log.Println(ucfg.BridgeValues().IRC)
		log.Println(funk.Keys(ucfg.BridgeValues().IRC))
		for name, p := range ucfg.BridgeValues().IRC {
			np := this.Cfg.BridgeValues().IRC
			if np == nil {
				np = map[string]config.Protocol{}
			}
			np[name] = p

			this.Cfg.BridgeValues().IRC = np
		}
	}

	// add gateway

	//
	msg := config.Message{}
	msg.Event = config.EventFailure
	msg.ID = "123"
	msg.Account = "name1"
	this.Router.Message <- msg
}
