package gows

import (
	"context"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite drive
)

// GoWS it's Go WebSocket or WhatSapp ;)
type GoWS struct {
	*whatsmeow.Client
	QrChan  chan whatsmeow.QRChannelItem
	Context context.Context

	cancelContext context.CancelFunc
	container     *sqlstore.Container
}

func (gows *GoWS) Start() error {
	// Already logged in, just connect
	if gows.Store.ID != nil {
		return gows.Connect()
	}

	// No ID stored, new login
	qrChan, _ := gows.GetQRChannel(gows.Context)
	// reissue from QrChan to gows.QrChan
	go func() {
		for {
			select {
			case <-gows.Context.Done():
				return
			case qr := <-qrChan:
				gows.QrChan <- qr
			}
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
	gows.cancelContext()
	err := gows.container.Close()
	if err != nil {
		gows.Log.Errorf("Error closing container: %v", err)
	}
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

func BuildSession(ctx context.Context, dialect string, address string) (*GoWS, error) {
	log := waLog.Stdout("Client", "DEBUG", true)
	dbLog := waLog.Stdout("Database", "DEBUG", true)

	// Prepare the database
	container, err := sqlstore.New(dialect, address, dbLog)
	if err != nil {
		return nil, err
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, err
	}

	// Create the client
	ctx, cancel := context.WithCancel(ctx)
	client := whatsmeow.NewClient(deviceStore, log)
	gows := GoWS{
		client,
		make(chan whatsmeow.QRChannelItem, 8),
		ctx,
		cancel,
		container,
	}
	return &gows, nil
}
