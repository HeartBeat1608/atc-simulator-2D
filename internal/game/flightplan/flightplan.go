package flightplan

import "atc-simulator/pkg/types"

type SegmentType int

const (
	SegmentTypeWaypoint SegmentType = iota
	SegmentTypeLanding
)

type FlightPlanSegment struct {
	Type           SegmentType
	WaypointName   string
	AirportID      string
	RunwayName     string
	TargetAltitude float64
	TargetSpeed    float64
}

type FlightPlan struct {
	OriginAirportID      string
	DestinationAirportID string
	Route                []FlightPlanSegment // Sequence of segments
	CurrentSegmentIndex  int
	Callsign             types.AircraftID
}
