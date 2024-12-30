package gows

import (
	"context"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite drive
)

// GoWS it's Go WebSocket or WhatSapp ;)
type GoWS struct {
	*whatsmeow.Client
	Context context.Context
	Events  chan interface{}

	cancelContext context.CancelFunc
	container     *sqlstore.Container
}

func (gows *GoWS) handleEvent(event interface{}) {
	var data interface{}
	switch event.(type) {
	case *events.Connected:
		// Populate the ConnectedEventData with the ID and PushName
		data = &ConnectedEventData{
			ID:       gows.Store.ID,
			PushName: gows.Store.PushName,
		}

	default:
		data = event
	}

	// reissue from Events to client
	gows.Events <- data
}

func (gows *GoWS) Start() error {
	// Already logged in, just connect
	if gows.Store.ID != nil {
		gows.AddEventHandler(gows.handleEvent)
		return gows.Connect()
	}

	// No ID stored, new login
	qrChan, _ := gows.GetQRChannel(gows.Context)

	// reissue from QrChan to Events
	go func() {
		for {
			select {
			case <-gows.Context.Done():
				return
			case qr := <-qrChan:
				gows.Events <- qr
			}
		}
	}()

	gows.AddEventHandler(gows.handleEvent)
	return gows.Connect()
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
		ctx,
		make(chan interface{}, 10),
		cancel,
		container,
	}
	return &gows, nil
}
