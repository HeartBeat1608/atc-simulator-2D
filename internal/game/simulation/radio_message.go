package simulation

import (
	"atc-simulator/pkg/types"
	"time"
)

type RadioMessage struct {
	Timestamp time.Time
	Callsign  types.AircraftID
	Message   string
	IsUrgent  bool
}

func (s *Simulation) AddRadioMessage(callsign types.AircraftID, message string, isUrgent bool) {
	msg := RadioMessage{
		Timestamp: time.Now(),
		Callsign:  callsign,
		Message:   message,
		IsUrgent:  isUrgent,
	}
	s.RadioLog = append(s.RadioLog, msg)

	if len(s.RadioLog) > s.maxRadioLogSize {
		s.RadioLog = s.RadioLog[len(s.RadioLog)-s.maxRadioLogSize:]
	}
}
