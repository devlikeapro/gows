package gows

import (
	"context"
	_ "github.com/mattn/go-sqlite3" // Import the SQLite drive
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// GoWS it's Go WebSocket or WhatSapp ;)
type GoWS struct {
	*whatsmeow.Client
	Context context.Context
	events  chan interface{}

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

	// reissue from events to client
	select {
	case <-gows.Context.Done():
		return
	case gows.events <- data:
	}
}

func (gows *GoWS) Start() error {
	gows.AddEventHandler(gows.handleEvent)

	// Not connected, listen for QR code events
	if gows.Store.ID == nil {
		gows.listenQRCodeEvents()
	}

	return gows.Connect()
}

func (gows *GoWS) listenQRCodeEvents() {
	// No ID stored, new login
	qrChan, _ := gows.GetQRChannel(gows.Context)

	// reissue from QrChan to events
	go func() {
		for {
			select {
			case <-gows.Context.Done():
				return
			case qr := <-qrChan:
				// If the event is empty, we should stop the goroutine
				if qr.Event == "" {
					return
				}
				gows.events <- qr
			}
		}
	}()
}

func (gows *GoWS) Stop() {
	gows.Disconnect()
	gows.cancelContext()
	err := gows.container.Close()
	if err != nil {
		gows.Log.Errorf("Error closing container: %v", err)
	}
	close(gows.events)
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

func BuildSession(ctx context.Context, log waLog.Logger, dialect string, address string) (*GoWS, error) {
	// Prepare the database
	container, err := sqlstore.New(dialect, address, log.Sub("Database"))
	if err != nil {
		return nil, err
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, err
	}

	// Configure the client
	client := whatsmeow.NewClient(deviceStore, log.Sub("Client"))
	client.AutomaticMessageRerequestFromPhone = true
	client.EmitAppStateEventsOnFullSync = true

	ctx, cancel := context.WithCancel(ctx)
	gows := GoWS{
		client,
		ctx,
		make(chan interface{}, 10),
		cancel,
		container,
	}
	return &gows, nil
}

func (gows *GoWS) GetEventChannel() <-chan interface{} {
	return gows.events
}

func (gows *GoWS) SendMessage(ctx context.Context, to types.JID, msg *waE2E.Message, extra ...whatsmeow.SendRequestExtra) (resp whatsmeow.SendResponse, err error) {
	resp, err = gows.Client.SendMessage(ctx, to, msg, extra...)
	if err != nil {
		return
	}
	info := &types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     to,
			Sender:   gows.GetOwnId(),
			IsFromMe: true,
			IsGroup:  to.Server == types.GroupServer,
		},
		ID:        resp.ID,
		Timestamp: resp.Timestamp,
		ServerID:  resp.ServerID,
	}
	evt := &events.Message{Info: *info, Message: msg, RawMessage: msg}
	go gows.handleEvent(evt)
	return
}
