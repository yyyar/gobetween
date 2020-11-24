package logging

/**
 * log.go - logging wrapper
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
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
		logger := &lumberjack.Logger{
			Filename:   output,
			MaxSize:    10, // megabytes
			MaxBackups: 3,
			MaxAge:     30,   //days
			Compress:   true, // disabled by default
		}
		logrus.SetOutput(logger)
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
