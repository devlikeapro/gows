package service

import (
	"github.com/devlikeapro/noweb2/gows"
)

func BuildSession() *gows.GoWS {
	store, err := gows.BuildSingleDeviceStore("sqlite3", "file:.sessions/gows.db?_foreign_keys=on")
	if err != nil {
		panic(err)
	}
	gws := gows.Build(store)
	return gws
}
