//Package logging Functions to set unique configuration for use with the logrus logger
package logging

import (
	"fmt"
	"os"
	"runtime"
	"regexp"

	log "github.com/sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
)

func SetParameters() LogValues {
	var logval LogValues

	logval.version = msgs.PGO_VERSION

	return logval
}

//LogValues holds the standard log value types
type LogValues struct {
	version  string
}

// formatter adds default fields to each log entry.
type formatter struct {
	fields log.Fields
	lf     log.Formatter
}

// Format satisfies the logrus.Formatter interface.
func (f *formatter) Format(e *log.Entry) ([]byte, error) {
	for k, v := range f.fields {
			e.Data[k] = v
	}
	return f.lf.Format(e)
}

//CrunchyLogger adds the customized logging fields to the logrus instance context
func CrunchyLogger(logDetails LogValues) {
	//Sets calling method as a field
	log.SetReportCaller(true)

	crunchyTextFormatter := &log.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := f.File
			function := f.Function
			re1 := regexp.MustCompile(`postgres-operator/(.*go)`)
			result1 := re1.FindStringSubmatch(f.File)
			if len(result1) > 1 {
				filename = result1[1]
			}
	
			re2 := regexp.MustCompile(`postgres-operator/(.*)`)
			result2 := re2.FindStringSubmatch(f.Function)
			if len(result2) > 1 {
				function = result2[1]
			}
			return fmt.Sprintf("%s()", function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
		//ForceColors: true,
		FullTimestamp: true,
		//DisableTruncation: true,
	}

	crunchyTextFormatter.TimestampFormat = "2006-01-02 15:04:05 -0700"

	log.SetFormatter(&formatter{
		fields: log.Fields{
			"version":  logDetails.version,
		},
		lf: crunchyTextFormatter,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the debug severity or above.
	log.SetLevel(log.DebugLevel)
}