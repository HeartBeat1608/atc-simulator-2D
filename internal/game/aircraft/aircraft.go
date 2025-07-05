package aircraft

import (
	"atc-simulator/internal/game/flightplan"
	"atc-simulator/pkg/types"
	"math"
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
)

var StateStringMap = map[AircraftState]string{
	CRUISE:     "CRUISE",
	CLIMB:      "CLIMB",
	DESCEND:    "DESCEND",
	HOLDING:    "HOLDING",
	APPROACH:   "APPROACH",
	LANDED:     "LANDED",
	TAKING_OFF: "TAKING_OFF",
}

type Aircraft struct {
	ID        types.AircraftID
	Position  types.Vec2
	Altitude  float64
	Heading   float64
	Speed     float64
	ClimbRate float64

	TargetAltitude   float64
	TargetSpeed      float64
	TargetHeading    float64
	DirectToWaypoint *types.Waypoint

	State AircraftState

	MaxTurnRateDegPerSec        float64
	MaxClimbRateFPM             float64
	MaxDescentRateFPM           float64
	AccelerationRateKnotsPerSec float64

	IsConflicting bool
	FlightPlan    *flightplan.FlightPlan
}

func NewAircraft(id types.AircraftID, pos types.Vec2, heading, speed, altitude float64, state AircraftState, route []types.Waypoint) *Aircraft {
	return &Aircraft{
		ID:                          id,
		Position:                    pos,
		Altitude:                    altitude,
		Heading:                     heading,
		Speed:                       speed,
		ClimbRate:                   0,
		TargetAltitude:              altitude, // Initial target is current altitude
		TargetSpeed:                 speed,
		TargetHeading:               heading,
		State:                       state,
		MaxTurnRateDegPerSec:        3.0,
		MaxClimbRateFPM:             3000.0,
		MaxDescentRateFPM:           -2500.0,
		AccelerationRateKnotsPerSec: 10.0 / 60.0,
	}
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
		dx := ac.DirectToWaypoint.Position.X - ac.Position.X
		dy := ac.DirectToWaypoint.Position.Y - ac.Position.Y
		targetBearing := math.Atan2(dx, -dy) * 180.0 / math.Pi
		targetBearing = math.Mod(targetBearing+360, 360)

		// log.Printf("DIRECT TO WAYPT: %s | bearing %.0f | current %.0f", ac.DirectToWaypoint.Name, targetBearing, ac.Heading)

		if ac.Position.DistanceTo(ac.DirectToWaypoint.Position) < 20 {
			ac.DirectToWaypoint = nil
			ac.TargetHeading = ac.Heading
		} else {
			ac.TargetHeading = targetBearing
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
