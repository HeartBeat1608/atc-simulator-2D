package aircraft

import (
	"atc-simulator/internal/game/airspace"
	"atc-simulator/internal/game/flightplan"
	"atc-simulator/pkg/types"
	"fmt"
	"log"
	"math"
	"time"
)

type AircraftState int

const (
	CRUISE AircraftState = iota
	CLIMB
	DESCEND
	HOLDING
	APPROACH
	LANDED
	TAKING_OFF
	READY_FOR_HANDOFF
)

var StateStringMap = map[AircraftState]string{
	CRUISE:            "CRUISE",
	CLIMB:             "CLIMB",
	DESCEND:           "DESCEND",
	HOLDING:           "HOLDING",
	APPROACH:          "APPROACH",
	LANDED:            "LANDED",
	TAKING_OFF:        "TAKING_OFF",
	READY_FOR_HANDOFF: "READY_FOR_HANDOFF",
}

type Aircraft struct {
	ID        types.AircraftID
	Position  types.Vec2
	Altitude  float64
	Heading   float64
	Speed     float64
	ClimbRate float64

	TargetAltitude    float64
	TargetSpeed       float64
	TargetHeading     float64
	DirectToWaypoint  *types.Waypoint
	ClearedForHandoff bool
	ClearedForLanding bool

	State AircraftState

	MaxTurnRateDegPerSec        float64
	MaxClimbRateFPM             float64
	MaxDescentRateFPM           float64
	AccelerationRateKnotsPerSec float64

	IsConflicting bool
	FlightPlan    *flightplan.FlightPlan
	LandingRunway *airspace.Runway
	Airspace      *airspace.Airspace

	AddRadioMessageFunc func(callsign types.AircraftID, message string, isUrgent bool)

	SpawnTime               time.Time
	LastRadioTime           time.Time
	MessageDebounceTime     time.Duration
	PreviousAltitudeRequest bool
	PreviousSpeedRequest    bool
	PreviousWaypointReached string
}

func NewAircraft(id types.AircraftID, pos types.Vec2, heading, speed, altitude float64, state AircraftState, flightPlan *flightplan.FlightPlan, asp *airspace.Airspace, addRadioMessageFunc func(types.AircraftID, string, bool)) *Aircraft {
	ac := &Aircraft{
		ID:                          id,
		Position:                    pos,
		Altitude:                    altitude,
		Heading:                     heading,
		Speed:                       speed,
		ClimbRate:                   0,
		TargetAltitude:              altitude,
		TargetSpeed:                 speed,
		TargetHeading:               heading,
		ClearedForHandoff:           false,
		ClearedForLanding:           false,
		State:                       state,
		MaxTurnRateDegPerSec:        3.0,
		MaxClimbRateFPM:             3000.0,
		MaxDescentRateFPM:           -2500.0,
		AccelerationRateKnotsPerSec: 10.0 / 60.0,
		Airspace:                    asp,
		FlightPlan:                  flightPlan,
		SpawnTime:                   time.Now(),
		LastRadioTime:               time.Now(),
		MessageDebounceTime:         5 * time.Second,
		AddRadioMessageFunc:         addRadioMessageFunc,
	}

	if ac.AddRadioMessageFunc != nil {
		ac.AddRadioMessageFunc(ac.ID, fmt.Sprintf("Requesting clearance to %s", ac.FlightPlan.DestinationAirportID), false)
	}

	return ac
}

func (ac *Aircraft) Update(dt float64) {
	rateScale := dt / 60.0
	if ac.Altitude < ac.TargetAltitude {
		rate := math.Min(ac.MaxClimbRateFPM, (ac.TargetAltitude-ac.Altitude)/(rateScale))
		ac.ClimbRate = rate
		ac.Altitude += ac.ClimbRate * rateScale
		if ac.Altitude >= ac.TargetAltitude {
			ac.Altitude = ac.TargetAltitude
			ac.ClimbRate = 0
			if ac.State == CLIMB {
				ac.State = CRUISE
			}
		}
	} else if ac.Altitude > ac.TargetAltitude {
		rate := math.Max(ac.MaxDescentRateFPM, (ac.TargetAltitude-ac.Altitude)/(rateScale))
		ac.ClimbRate = rate
		ac.Altitude += ac.ClimbRate * rateScale
		if ac.Altitude <= ac.TargetAltitude {
			ac.Altitude = ac.TargetAltitude
			ac.ClimbRate = 0
			if ac.State == DESCEND {
				ac.State = CRUISE
			}
		}
	} else {
		ac.ClimbRate = 0
	}

	if ac.DirectToWaypoint != nil {
		if ac.Position.DistanceTo(ac.DirectToWaypoint.Position) < 30 {
			ac.DirectToWaypoint = nil

			ac.FlightPlan.CurrentSegmentIndex++
			if ac.FlightPlan.CurrentSegmentIndex >= len(ac.FlightPlan.Route) {
				log.Printf("%s completed its flight plan in this sector.", ac.ID)
				ac.State = READY_FOR_HANDOFF
			}
			ac.TargetHeading = ac.Heading
		} else {
			ac.TargetHeading = ac.Position.HeadingTo(ac.DirectToWaypoint.Position)
		}
	}

	if ac.DirectToWaypoint == nil && ac.FlightPlan != nil && ac.FlightPlan.CurrentSegmentIndex < len(ac.FlightPlan.Route) {
		nextSegment := ac.FlightPlan.Route[ac.FlightPlan.CurrentSegmentIndex]

		switch nextSegment.Type {
		case flightplan.SegmentTypeWaypoint:
			{
				nextWaypoint, ok := ac.GetWaypoint(nextSegment.WaypointName)
				if ok {
					ac.SetDirectTo(nextWaypoint)

					if ac.TargetAltitude == ac.Altitude {
						ac.SetAltitude(nextSegment.TargetAltitude)
					}
					if ac.TargetSpeed == ac.Speed {
						ac.SetSpeed(nextSegment.TargetSpeed)
					}

					log.Printf("%s now directing to %s (Segment %d)", ac.ID, nextSegment.WaypointName, ac.FlightPlan.CurrentSegmentIndex)
				} else {
					log.Printf("ERROR: Waypoint %s not found for %s's flight plan segment %d", nextSegment.WaypointName, ac.ID, ac.FlightPlan.CurrentSegmentIndex)
				}
			}
		case flightplan.SegmentTypeLanding:
			{
				if airport, ok := ac.Airspace.Airports[nextSegment.AirportID]; ok {
					if runway, ok := airport.Runways[nextSegment.RunwayName]; ok {
						ac.LandingRunway = runway
						ac.SetDirectToRunway(runway.Name, runway.Threshold)
						if ac.TargetAltitude == ac.Altitude {
							ac.SetAltitude(nextSegment.TargetAltitude)
						}
						if ac.TargetSpeed == ac.Speed {
							ac.SetSpeed(nextSegment.TargetSpeed)
						}
						ac.State = APPROACH
						log.Printf("%s cleared for approach to %s at %s. Directing to threshold %v", ac.ID, runway.Name, airport.ID, runway.Threshold)
					} else {
						log.Printf("ERROR: Runway %s not found for airport %s in %s's flight plan", nextSegment.RunwayName, nextSegment.AirportID, ac.ID)
					}
				} else {
					log.Printf("ERROR: Airport %s not found for %s's landing segment", nextSegment.AirportID, ac.ID)
				}
				break
			}
		}
	}

	if ac.State == APPROACH && ac.LandingRunway != nil {
		ac.TargetHeading = ac.Position.HeadingTo(ac.LandingRunway.Threshold)
		distanceToThreshold := ac.Position.DistanceTo(ac.LandingRunway.Threshold)
		if distanceToThreshold < 150 {
			ac.TargetSpeed = 150
			if ac.Altitude > 1000 && distanceToThreshold < 100 {
				ac.SetAltitude(0)
				ac.ClimbRate = -1500
			} else if ac.Altitude > 500 {
				ac.SetAltitude(0)
				ac.ClimbRate = -500
			} else {
				ac.SetAltitude(0)
				ac.ClimbRate = -200
			}
		} else {
			ac.TargetSpeed = 200
		}

		if distanceToThreshold < 20 && ac.Altitude < 100 {
			ac.State = LANDED
			ac.Speed = 0
			ac.ClimbRate = 0
			ac.Position = ac.LandingRunway.Threshold
			ac.DirectToWaypoint = nil
			ac.FlightPlan.CurrentSegmentIndex = len(ac.FlightPlan.Route)
			log.Printf("%s HAS LANDED at %s!", ac.ID, ac.LandingRunway.Name)
			ac.AddRadioMessageFunc(ac.ID, fmt.Sprintf("Touch down, %s", ac.LandingRunway.Name), false)
		}
	}

	if ac.Heading != ac.TargetHeading {
		diff := math.Mod(ac.TargetHeading-ac.Heading+360, 360)
		var turnAmount float64
		if diff > 180 {
			turnAmount = -ac.MaxTurnRateDegPerSec * dt
		} else {
			turnAmount = ac.MaxTurnRateDegPerSec * dt
		}

		if math.Abs(diff) < math.Abs(turnAmount) {
			ac.Heading = ac.TargetHeading
		} else {
			ac.Heading = math.Mod(ac.Heading+turnAmount+360, 360)
		}
	}

	if ac.Speed < ac.TargetSpeed {
		ac.Speed += ac.AccelerationRateKnotsPerSec * dt
		if ac.Speed > ac.TargetSpeed {
			ac.Speed = ac.TargetSpeed
		}
	} else if ac.Speed > ac.TargetSpeed {
		ac.Speed -= ac.AccelerationRateKnotsPerSec * dt
		if ac.Speed < ac.TargetSpeed {
			ac.Speed = ac.TargetSpeed
		}
	}

	radians := ac.Heading * math.Pi / 180.0
	pixelsPerSec := ac.Speed / 3600.0 * types.NM_TO_PIXEL

	ac.Position.X += pixelsPerSec * math.Sin(radians) * dt
	ac.Position.Y -= pixelsPerSec * math.Cos(radians) * dt

	// Radio communication logic
	if time.Since(ac.LastRadioTime) > ac.MessageDebounceTime {
		if !ac.ClearedForLanding {
			if ac.TargetAltitude > ac.Altitude+100 && !ac.PreviousAltitudeRequest {
				ac.AddRadioMessageFunc(ac.ID, fmt.Sprintf("Requesting higher to FL%.0f", ac.TargetAltitude/100), false)
				ac.PreviousAltitudeRequest = true
				ac.LastRadioTime = time.Now()
			} else if ac.TargetAltitude < ac.Altitude-100 && !ac.PreviousAltitudeRequest {
				ac.AddRadioMessageFunc(ac.ID, fmt.Sprintf("Requesting lower to FL%.0f", ac.TargetAltitude/100), false)
				ac.PreviousAltitudeRequest = true
				ac.LastRadioTime = time.Now()
			} else if math.Abs(ac.TargetAltitude-ac.Altitude) < 100 {
				ac.PreviousAltitudeRequest = false
			}
		} else {
			ac.PreviousAltitudeRequest = false
		}

		// Altitude requests
		if ac.TargetAltitude > ac.Altitude+100 && !ac.PreviousAltitudeRequest { // Target higher
			ac.AddRadioMessageFunc(ac.ID, fmt.Sprintf("Requesting higher to FL%.0f", ac.TargetAltitude/100), false)
			ac.PreviousAltitudeRequest = true
			ac.LastRadioTime = time.Now()
		} else if ac.TargetAltitude < ac.Altitude-100 && !ac.PreviousAltitudeRequest { // Target lower
			ac.AddRadioMessageFunc(ac.ID, fmt.Sprintf("Requesting lower to FL%.0f", ac.TargetAltitude/100), false)
			ac.PreviousAltitudeRequest = true
			ac.LastRadioTime = time.Now()
		} else if math.Abs(ac.TargetAltitude-ac.Altitude) < 100 { // Reached target altitude
			ac.PreviousAltitudeRequest = false // Reset request state
		}

		// Speed requests (similar logic)
		if math.Abs(ac.TargetSpeed-ac.Speed) > 50 && !ac.PreviousSpeedRequest {
			// ac.AddRadioMessageFunc(ac.ID, fmt.Sprintf("Requesting speed %.0f knots", ac.TargetSpeed), false)
			// ac.PreviousSpeedRequest = true
			// ac.LastRadioTime = time.Now()
		} else if math.Abs(ac.TargetSpeed-ac.Speed) < 10 {
			ac.PreviousSpeedRequest = false
		}
	}

	if ac.State == APPROACH && !ac.ClearedForLanding {
		distanceToThreshold := ac.Position.DistanceTo(ac.LandingRunway.Threshold)
		headingDiff := math.Abs(ac.Heading - ac.LandingRunway.Heading)
		if headingDiff > 180 {
			headingDiff = 360 - headingDiff
		}

		if distanceToThreshold > 500 && ac.Altitude < 5000 && headingDiff < 30 {
			ac.AddRadioMessageFunc(ac.ID, fmt.Sprintf("Requesting clearance to land runway %s.", ac.LandingRunway.Name), false)
			ac.LastRadioTime = time.Now()
		}
	}

	// Waypoint Reached Report (should be less frequent, maybe not debounced by general message time)
	// This typically happens when DirectToWaypoint is reset.
	// So this logic will move into the DirectToWaypoint reached block.
	if ac.DirectToWaypoint == nil && ac.FlightPlan != nil && ac.FlightPlan.CurrentSegmentIndex > 0 &&
		ac.FlightPlan.CurrentSegmentIndex-1 < len(ac.FlightPlan.Route) { // Ensure index is valid for prev segment
		prevWpName := ac.FlightPlan.Route[ac.FlightPlan.CurrentSegmentIndex-1].WaypointName
		if ac.PreviousWaypointReached != prevWpName {
			ac.AddRadioMessageFunc(ac.ID, fmt.Sprintf("Approaching %s", prevWpName), false) // Report reaching, or approaching next
			ac.PreviousWaypointReached = prevWpName
			ac.LastRadioTime = time.Now() // Debounce here too
		}
	}
}

func (ac *Aircraft) SetHeading(h float64) {
	ac.TargetHeading = math.Mod(h+360, 360)
	ac.DirectToWaypoint = nil
}

func (ac *Aircraft) SetAltitude(alt float64) {
	ac.TargetAltitude = alt
	if alt > ac.Altitude {
		ac.State = CLIMB
	} else if alt < ac.Altitude {
		ac.State = DESCEND
	} else {
		ac.State = CRUISE
	}
}

func (ac *Aircraft) SetSpeed(s float64) {
	ac.TargetSpeed = s
}

func (ac *Aircraft) SetDirectTo(wp *types.Waypoint) {
	ac.DirectToWaypoint = wp
}

func (ac *Aircraft) SetDirectToRunway(name string, pos types.Vec2) {
	ac.DirectToWaypoint = &types.Waypoint{
		Name:     name,
		Position: pos,
	}
}

func (ac *Aircraft) GetWaypoint(wpName string) (wp *types.Waypoint, ok bool) {
	wp, ok = ac.Airspace.Waypoints[wpName]
	return
}
