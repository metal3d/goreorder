// log packate override the default log package from the standard library
// to be activated on verbose mode, use the -v flag
package log

import (
	l "log"
	"os"
)

var verbose bool
var (
	logger *l.Logger
)

func init() {
	// set the log output to stderr
	logger = l.New(os.Stderr, "", l.LstdFlags)
}

func GetLogger() *l.Logger {
	return logger
}

// SetVerbose sets the verbose mode.
func SetVerbose(v bool) {
	if v {
		logger.SetOutput(os.Stderr)
	} else {
		// make a writer to os.DevNull
		writer, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
		logger.SetOutput(writer)
	}
}
