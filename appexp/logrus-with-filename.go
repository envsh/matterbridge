package appexp

import (
	"fmt"
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
)

// usage: logrus.StandardLogger().AddHook(&ContextHook{})
// ContextHook ...
type ContextHook struct{}

// Levels ...
func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire ...
func (hook ContextHook) Fire(entry *logrus.Entry) error {
	const depth = 10 // can change that
	if pc, file, line, ok := runtime.Caller(depth); ok {
		funcName := runtime.FuncForPC(pc).Name()

		entry.Data["src"] = fmt.Sprintf("%s:%v:%s", path.Base(file), line, path.Base(funcName))
	}

	return nil
}
