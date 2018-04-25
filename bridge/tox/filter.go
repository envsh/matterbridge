package btox

import (
	"errors"
	"mkuse/hlpbot"
	"strings"
)

func (this *Btox) IsFiltered(topic, msg string) error {
	if strings.HasPrefix(msg, "@@") { // 不转发的消息格式
		return errors.New("explict no forward @@ hint:")
	}

	if !grc.TryPut(topic) {
		return errors.New("rate limit exceed:" + topic)
	}

	lang := helper.DetectLang(msg)
	if lang == "ru" && strings.HasSuffix(topic, "Tox Public Chat") {
		return errors.New("#tox can not say ru words: " + msg)
	}
	return nil
}
