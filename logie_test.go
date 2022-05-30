package logie

import (
	"log"
	"os"
	"testing"

	"github.com/i0Ek3/logie"
)

func TestLogie(t *testing.T) {
	logie.Info("std log")
	logie.SetOptions(logie.WithLevel(logie.DebugLevel))
	logie.Debug("change std log to debug level")
	logie.SetOptions(logie.WithFormatter(&logie.JSONFormatter{IgnoreBasicFields: false}))
	logie.Debug("log in json format")
	logie.Info("another log in json format")

	fd, err := os.OpenFile("test.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln("create file test.log failed")
	}
	defer fd.Close()

	l := logie.New(logie.WithLevel(logie.InfoLevel),
		logie.WithPosition(fd),
		logie.WithFormatter(&logie.JsonFormatter{IgnoreBasicFields: false}),
	)
	l.Info("custom log with json formatter")
}
