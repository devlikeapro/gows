package gows

import (
	"context"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	waLog "go.mau.fi/whatsmeow/util/log"
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

	//for evt := range qrChan {
	//	if evt.Event == "code" {
	//		// Render the QR code here
	//		// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
	//		// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
	//		qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
	//	} else {
	//		fmt.Println("Login event:", evt.Event)
	//	}
	//}
	return nil
}

func (gows *GoWS) Stop() {
	gows.Disconnect()
}
