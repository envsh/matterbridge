package btox

import (
	"errors"
	"fmt"
	"mkuse/hlpbot"
	"strings"

	funk "github.com/thoas/go-funk"
)

func (this *Btox) IsFiltered(topic, msg string) error {
	if strings.HasPrefix(msg, "@@") { // 不转发的消息格式
		return errors.New("explict no forward @@ hint:")
	}

	/*
		if !grc.TryPut(topic) {
			return errors.New("rate limit exceed:" + topic)
		}
	*/

	// TODO configure this?
	if strings.HasSuffix(topic, "@Tox Public Chat") ||
		strings.HasSuffix(topic, "@test autobot") {
		// сука Блять: Language: srp Script: Cyrillic
		// test До́брое у́тро!: Language: rus Script: Cyrillic
		lang := helper.DetectLang(msg)
		if funk.Contains([]string{"rus", "srp", "mkd"}, lang) {
			return fmt.Errorf("#tox can not say ru(%s) words: %s", lang, msg)
		}
	}
	return nil
}
