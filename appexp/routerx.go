package appexp

import (
	"fmt"
	"gopp"
	"log"
	"reflect"
	"strings"

	"github.com/42wim/matterbridge/bridge/config"
	"github.com/42wim/matterbridge/gateway"
)

func HasBridge(r *gateway.Router, pt, name string) bool {
	brvs := r.BridgeValues()
	has := false
	EachBridge(r, brvs, func(proto string, vals map[string]config.Protocol) {
		if _, ok := vals[name]; ok && strings.ToLower(proto) == pt {
			has = true
		}
	})
	return has
}
func FindBridge(r *gateway.Router, pt, name string) (config.Protocol, bool) {
	brvs := r.BridgeValues()
	retpto := config.Protocol{}
	has := false
	EachBridge(r, brvs, func(proto string, vals map[string]config.Protocol) {
		if pto, ok := vals[name]; ok && strings.ToLower(proto) == pt {
			has = true
			retpto = pto
		}
	})
	return retpto, has
}

// 简化版本，直接传字符串参数
// [irc]
// [irc.somenode1]
// server =
// ...
func AddBridge1(r *gateway.Router, cfg string) {
	oldvals := ViperValues(r.Config).AllSettings()
	cfgo := config.NewConfigFromString([]byte(cfg))
	newvpo := ViperValues(r.Config)
	for k, v := range oldvals { // very important, or lost old config values
		newvpo.Set(k, v)
	}

	EachBridge(r, cfgo.BridgeValues(), func(proto string, vals map[string]config.Protocol) {
		brvs := r.BridgeValues()
		brvx := reflect.ValueOf(brvs).Elem()
		oldvalsx := brvx.FieldByName(proto).Interface()
		oldvals := oldvalsx.(map[string]config.Protocol)
		if oldvals == nil {
			oldvals = map[string]config.Protocol{}
			brvx.FieldByName(proto).Set(reflect.ValueOf(oldvals))
		}
		for name, val := range vals {
			// log.Println(name, val)
			if _, ok := oldvals[name]; ok {
				log.Printf("Bridge already exist %s.%s\n", proto, name)
			} else {
				oldvals[name] = val
				log.Printf("Bridge added %s.%s\n", proto, name)
			}
		}
	})
}

func EachBridge(r *gateway.Router, brvs *config.BridgeValues, f func(proto string, vals map[string]config.Protocol)) {
	brvx := reflect.ValueOf(brvs).Elem()
	brvty := brvx.Type()
	for idx := 0; idx < brvx.NumField(); idx++ {
		brve := brvx.Field(idx)
		if brve.Type().Kind() == reflect.Map {
			brv := brve.Interface().(map[string]config.Protocol)
			// proto := strings.ToLower(brvty.Field(idx).Name)
			proto := brvty.Field(idx).Name
			f(proto, brv)
		}
	}
}

func HasGateway(r *gateway.Router, name string) bool {
	_, ok := r.Gateways[name]
	return ok
}

// 简化版本，直接传字符串参数
// [[gateway]]
// name = "gateway1"
// enable=true
// [[gateway.in]]
// [[gateway.out]]
// ...
func AddGateway1(r *gateway.Router, cfg string) error {
	oldvals := ViperValues(r.Config).AllSettings()
	cfgo := config.NewConfigFromString([]byte(cfg))
	newvpo := ViperValues(r.Config)
	for k, v := range oldvals { // very important, or lost old config values
		newvpo.Set(k, v)
	}

	for _, entry := range cfgo.BridgeValues().Gateway {
		if !entry.Enable {
			continue
		}
		if entry.Name == "" {
			return fmt.Errorf("%s", "Gateway without name found")
		}
		if _, ok := r.Gateways[entry.Name]; ok {
			return fmt.Errorf("Gateway with name %s already exists", entry.Name)
		}

		r.Gateways[entry.Name] = gateway.New(entry, r)
		for bracc, br := range r.Gateways[entry.Name].Bridges {
			// br.Config = r.Config
			// fakecfg := ((*fakeconfig)(br.Config))
			// log.Println(fakecfg == nil)
			if false {
				log.Println(bracc, bracc == br.Protocol+"."+br.Name)
			}
		}
		log.Printf("Gateway added %s\n", entry.Name)

		StartGateway(r, entry.Name)
	}
	return nil
}

func StartGateway(r *gateway.Router, account string) error {
	for acc, gw := range r.Gateways {
		if acc != account {
			continue
		}

		for brname, br := range gw.Bridges {
			log.Printf("Start gateway's bridge %s.%s.%s\n", acc, brname, br.Protocol)
			log.Println("Server", br.GetString("Server"))
			if true {
				err := br.Connect()
				gopp.ErrPrint(err)
				if err != nil {
					return fmt.Errorf("Bridge %s failed to start: %v", br.Account, err)
				}
				err = br.JoinChannels()
				gopp.ErrPrint(err)
				if err != nil {
					return fmt.Errorf("Bridge %s failed to join channel: %v", br.Account, err)
				}
			}
		}

		return nil
	}
	return fmt.Errorf("Bridge not found %s", account)
}

func SetEnabledGateway(r *gateway.Router, account string, enabled bool) {
}

func SendFailmsgAsreconn(r *gateway.Router, account string) {
	msgo := config.Message{}
	msgo.Account = account
	msgo.Event = config.EventFailure
	r.Message <- msgo
}
