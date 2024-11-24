package gows

import (
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/util/log"
)

// BuildSingleDeviceStore creates a store that can be used to interact with a single device.
func BuildSingleDeviceStore(dialect string, address string) (*store.Device, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New(dialect, address, dbLog)
	if err != nil {
		return nil, err
	}

	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, err
	}
	return deviceStore, err
}
