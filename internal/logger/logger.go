package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// L is the shared application logger.
var L = logrus.New()

func Init() {
	L.SetOutput(os.Stdout)
	L.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_FORMAT")), "json") {
		L.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z07:00",
		})
	}
	level, err := logrus.ParseLevel(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if err != nil {
		level = logrus.InfoLevel
	}
	L.SetLevel(level)
}

func WithComponent(component string) *logrus.Entry {
	return L.WithField("component", component)
}
