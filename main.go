package main

import (
	"fmt"
	"os"
	"point-set/base"
	"point-set/core"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	if base.InDebug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	addr := "127.0.0.1:10000"
	mgr, err := core.NewRoomManager(addr)
	if err != nil {
		panic(err)
	}
	if base.InDebug {
		mgr.CreateTestRoom()
	}

	go startHttp(mgr)

	base.LogPrint(base.LevelInfo, nil, fmt.Sprintf("start KCP %s", addr))
	err = mgr.Listen()
	if err != nil {
		panic(err)
	}
}
