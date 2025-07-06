package flightplan

import "atc-simulator/pkg/types"

type FlightPlanSegment struct {
	WaypointName   string
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
