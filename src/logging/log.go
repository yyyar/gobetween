/**
 * log.go - logging wrapper
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package logging

import (
	"bytes"
	"fmt"
	"github.com/Sirupsen/logrus"
	"os"
	"strings"
)

/**
 * Logging initialize
 */
func init() {
	logrus.SetFormatter(new(MyFormatter))
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetOutput(os.Stdout)
}

/**
 * Configure logging
 */
func Configure(output string, l string) {

	if output == "" || output == "stdout" {
		logrus.SetOutput(os.Stdout)
	} else if output == "stderr" {
		logrus.SetOutput(os.Stderr)
	} else {
		f, err := os.OpenFile(output, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.SetOutput(f)
	}

	if l == "" {
		return
	}

	if level, err := logrus.ParseLevel(l); err != nil {
		logrus.Fatal("Unknown loglevel ", l)
	} else {
		logrus.SetLevel(level)
	}
}

/**
 * Our custom formatter
 */
type MyFormatter struct{}

/**
 * Format entry
 */
func (f *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	name, ok := entry.Data["name"]
	if !ok {
		name = "default"
	}
	fmt.Fprintf(b, "%s [%-5.5s] (%s): %s\n", entry.Time.Format("2006-01-02 15:04:05"), strings.ToUpper(entry.Level.String()), name, entry.Message)
	return b.Bytes(), nil
}

/**
 * Add logger name as field var
 */
func For(name string) *logrus.Entry {
	return logrus.WithField("name", name)
}

/* ----- Wrap logrus ------ */

func Debug(args ...interface{}) {
	logrus.Debug(args...)
}

func Info(args ...interface{}) {
	logrus.Info(args...)
}

func Warn(args ...interface{}) {
	logrus.Warn(args...)
}

func Error(args ...interface{}) {
	logrus.Error(args...)
}

func Fatal(args ...interface{}) {
	logrus.Fatal(args...)
}
