package btox

import (
	"fmt"
	"gopp"

	tox "github.com/TokTok/go-toxcore-c"
)

func (this *Btox) isChannelCmd(groupNumber, peerNumber uint32, msg string) bool {
	return false
}

// return true if is processed channel command
func (this *Btox) processChannelCmd(groupNumber, peerNumber uint32, msg string) bool {
	if msg == "!help" {
		this.processChanHelpCmd(groupNumber, peerNumber, msg)
		return true
	}
	return false
}

func (this *Btox) processChanHelpCmd(groupNumber, peerNumber uint32, msg string) {
	peerName, err := this.i.ConferencePeerGetName(groupNumber, peerNumber)
	gopp.ErrPrint(err)
	helpText := fmt.Sprintf("%s: Valid commands: !help !users !ping", peerName)
	this.i.ConferenceSendMessage(groupNumber, tox.MESSAGE_TYPE_NORMAL, helpText)
}
