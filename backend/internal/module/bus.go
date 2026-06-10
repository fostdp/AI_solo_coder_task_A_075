package module

import (
	"time"

	"gas-leak-monitor/internal/model"
)

type ValidatedDetectorData struct {
	DetectorID    string
	Concentration float64
	Timestamp     time.Time
	Valid         bool
	Reason        string
}

type ValidatedSensorData struct {
	SensorID  string
	Type      string
	Value     float64
	Timestamp time.Time
	Valid     bool
	Reason    string
}

type AlarmEvent struct {
	DetectorID    string
	Level         int
	Concentration float64
	Message       string
	PartitionID   string
}

type ControlCommand struct {
	DeviceType  string
	PartitionID string
	Action      string
	Source      string
}

type LocateRequestMsg struct {
	Readings      []model.DetectorReading
	WindSpeed     float64
	WindDirection float64
	WindTimestamp  time.Time
	Method        string
	ResultChan    chan *model.LeakSourceResult
}

type WSBroadcast struct {
	Type    string
	Payload interface{}
}

type MessageBus struct {
	ValidatedDetectorData chan *ValidatedDetectorData
	ValidatedSensorData   chan *ValidatedSensorData
	AlarmEvent            chan *AlarmEvent
	ControlCommand        chan *ControlCommand
	LocateRequest         chan *LocateRequestMsg
	WSBroadcast           chan *WSBroadcast
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		ValidatedDetectorData: make(chan *ValidatedDetectorData, 200),
		ValidatedSensorData:   make(chan *ValidatedSensorData, 100),
		AlarmEvent:            make(chan *AlarmEvent, 100),
		ControlCommand:        make(chan *ControlCommand, 100),
		LocateRequest:         make(chan *LocateRequest, 10),
		WSBroadcast:           make(chan *WSBroadcast, 256),
	}
}
