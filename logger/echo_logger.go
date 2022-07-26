package logger

import (
	"io"

	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
)

/*
	Wrapper around logrus.Logger for the methods missing from the echo.Logger
	interface. All of the "rest" of the methods on the echoLogger interface are
	called directly on the logrus.Logger

	1. We "may" use the *j methods eventually, so just in case they just log the
	   map[string]interface{} as it comes in
	2. The Output method is easy, so it is also implemented
	3. The setlevel/level  methods panic because we only care about the single
	   logger level
	4. The Prefix/Header related methods are not implemented because they don't
	   matter to the logrus logger.
*/

type EchoLogger struct {
	*logrus.Entry
}

/// Wrapping _level_j methods
func (el EchoLogger) Printj(j log.JSON) { el.Logger.Printf("%+v", j) }
func (el EchoLogger) Debugj(j log.JSON) { el.Logger.Debugf("%+v", j) }
func (el EchoLogger) Infoj(j log.JSON)  { el.Logger.Infof("%+v", j) }
func (el EchoLogger) Errorj(j log.JSON) { el.Logger.Errorf("%+v", j) }
func (el EchoLogger) Warnj(j log.JSON)  { el.Logger.Warnf("%+v", j) }
func (el EchoLogger) Fatalj(j log.JSON) { el.Logger.Fatalf("%+v", j) }
func (el EchoLogger) Panicj(j log.JSON) { el.Logger.Panicf("%+v", j) }

/// output is easy
func (el EchoLogger) SetOutput(out io.Writer) { el.Logger.SetOutput(out) }
func (el EchoLogger) Output() io.Writer       { return el.Logger.Out }

/// we don't use the "set level" on the echo logger since we're using a single unified logger
func (el EchoLogger) SetLevel(log.Lvl) {
	panic("DON'T USE THIS METHOD - SET IT ON THE LOGRUS LOGGER")
}
func (el EchoLogger) Level() log.Lvl {
	panic("DON'T USE THIS METHOD - SET IT ON THE LOGRUS LOGGER")
}

/// we don't set a prefix ever on the logger
func (el EchoLogger) SetPrefix(string) { panic("DON'T USE THIS METHOD") }
func (el EchoLogger) Prefix() string   { panic("DON'T USE THIS METHOD") }

/// we don't set a header on the logger either
func (el EchoLogger) SetHeader(_ string) { panic("DON'T USE THIS METHOD") }
