package nsqd

import (
	"github.com/nsqio/nsq/internal/lg"
)

type Logger lg.Logger

const (
	LOG_DEBUG = lg.DEBUG
	LOG_INFO  = lg.INFO
	LOG_WARN  = lg.WARN
	LOG_ERROR = lg.ERROR
	LOG_FATAL = lg.FATAL
)

func (n *NSQD) logf(level lg.LogLevel, f string, args ...interface{}) {
	opts := n.getOpts()
	lg.Logf(opts.Logger, opts.logLevel, level, f, args...)
}
