package flightplan

import "atc-simulator/pkg/types"

type FlightPlan struct {
	OriginAirportID      string
	DestinationAirportID string
	Route                []types.Waypoint // Sequence of waypoints
	AssignedAltitude     float64
	AssignedSpeed        float64
	CurrentWaypointIndex int // Index of the next waypoint the aircraft is targeting
}
