package gows

import (
	"context"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite drive
)

// GoWS it's Go WebSocket or WhatSapp ;)
type GoWS struct {
	*whatsmeow.Client
	QrChan chan whatsmeow.QRChannelItem
}

func Build(store *store.Device) *GoWS {
	log := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(store, log)
	return &GoWS{
		client,
		make(chan whatsmeow.QRChannelItem, 8),
	}
}

func (gows *GoWS) Start() error {
	// Already logged in, just connect
	if gows.Store.ID != nil {
		return gows.Connect()
	}

	// No ID stored, new login
	qrChan, _ := gows.GetQRChannel(context.TODO())
	// reissue from QrChan to gows.QrChan
	go func() {
		for evt := range qrChan {
			gows.QrChan <- evt
		}
	}()

	err := gows.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (gows *GoWS) Stop() {
	gows.Disconnect()
	close(gows.QrChan)
}

func (gows *GoWS) GetOwnId() types.JID {
	if gows == nil {
		return types.EmptyJID
	}
	id := gows.Store.ID
	if id == nil {
		return types.EmptyJID
	}
	return *id
}

type ConnectedEventData struct {
	ID       *types.JID
	PushName string
}

func BuildSession(dialect string, address string) *GoWS {
	store, err := BuildSingleDeviceStore(dialect, address)
	if err != nil {
		panic(err)
	}
	gws := Build(store)
	return gws
}
