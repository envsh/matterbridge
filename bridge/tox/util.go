package btox

import (
	"fmt"
	"regexp"

	funk "github.com/thoas/go-funk"
)

func DiffSlice(old, new_ interface{}) (added []interface{}, deleted []interface{}) {
	funk.ForEach(old, func(e interface{}) {
		if !funk.Contains(new_, e) {
			deleted = append(deleted, e)
		}
	})
	funk.ForEach(new_, func(e interface{}) {
		if !funk.Contains(old, e) {
			added = append(added, e)
		}
	})
	return
}

func restoreUserName(Username string) string {
	reg := `\[freenode_(.+)@matrix\]`
	exp := regexp.MustCompile(reg)
	mats := exp.FindAllStringSubmatch(Username, -1)
	if len(mats) > 0 {
		return fmt.Sprintf("[%s@irc] ", mats[0][1])
	}
	return Username
}
