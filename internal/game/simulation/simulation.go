package simulation

import (
	"atc-simulator/internal/game/aircraft"
	"atc-simulator/internal/game/airspace"
	"atc-simulator/internal/game/conflict"
	"atc-simulator/internal/game/flightplan"
	"atc-simulator/pkg/types"
	"fmt"
	"log"
	"math/rand"
	"slices"
	"time"
)

type Simulation struct {
	Aircrafts       map[types.AircraftID]*aircraft.Aircraft
	Airspace        *airspace.Airspace
	TickRate        float64
	TimeOfDay       time.Time
	GameTimeSeconds float64

	HandOffs        int
	MissedHandoffs  int
	Conflicts       int
	RadioLog        []RadioMessage
	maxRadioLogSize int

	lastSpawnTime        time.Time
	spawnInterval        time.Duration
	nextAircraftID       int
	maxAircraftsOnScreen int
}

func NewSimulation(tickRate float64) *Simulation {
	simpleAirspace := airspace.NewAirspace()

	s := &Simulation{
		Aircrafts: make(map[types.AircraftID]*aircraft.Aircraft),
		Airspace:  simpleAirspace,
		TickRate:  tickRate,
		TimeOfDay: time.Now(),

		lastSpawnTime:        time.Now(),
		spawnInterval:        20 * time.Second,
		nextAircraftID:       100,
		maxAircraftsOnScreen: 5,
		maxRadioLogSize:      50,

		HandOffs:       0,
		MissedHandoffs: 0,
		Conflicts:      0,
	}

	s.SpawnRandomAircraft()
	return s
}

func (s *Simulation) Update(dt float64) {
	s.GameTimeSeconds += dt
	for _, ac := range s.Aircrafts {
		ac.Update(dt)
		ac.IsConflicting = false

		if ac.FlightPlan != nil && ac.FlightPlan.CurrentSegmentIndex >= len(ac.FlightPlan.Route) {
			isAtExit := false
			for _, exitWpName := range s.Airspace.ExitWaypoints {
				if exitWp, ok := s.Airspace.Waypoints[exitWpName]; ok {
					if ac.Position.DistanceTo(exitWp.Position) < 50 {
						isAtExit = true
						break
					}
				}
			}

			if isAtExit {
				s.HandOffAircraft(ac.ID)
			} else {
				// Aircraft completed plan but not at exit. Could be a holding pattern or error condition.
				// For now, it will just keep flying straight. Later, make it hold or get penalized.
				continue
			}
		}
	}
	s.CheckForConflicts()
	s.TimeOfDay = s.TimeOfDay.Add(time.Duration(dt*float64(time.Second)) * time.Second)

	if len(s.Aircrafts) < s.maxAircraftsOnScreen {
		s.lastSpawnTime = s.lastSpawnTime.Add(time.Duration(dt) * time.Second)
		if time.Since(s.lastSpawnTime) > s.spawnInterval {
			s.SpawnRandomAircraft()
			s.lastSpawnTime = time.Now()
		}
	}

	s.CleanupAircraft()
}

func (s *Simulation) HandOffAircraft(aircraftID types.AircraftID) {
	if ac, ok := s.Aircrafts[aircraftID]; ok {
		s.AddRadioMessage(ac.ID, "Good day, contact next controller.", false)
		log.Printf("HANDOFF: Aircraft % sucessfully handed off.", ac.ID)
		delete(s.Aircrafts, aircraftID)
		s.HandOffs++
	}
}

func (s *Simulation) randomFloatInRange(minF, maxF float64) float64 {
	fRange := maxF - minF
	return minF + rand.Float64()*fRange
}

func (s *Simulation) getWaypoint(waypoint string) (*types.Waypoint, bool) {
	wp, ok := s.Airspace.Waypoints[waypoint]
	return wp, ok
}

func (s *Simulation) SpawnRandomAircraft() {
	// Define spawn points (e.g., edges of your 1024x768 screen)
	minX, maxX := 10.0, 1024.0
	minY, maxY := 10.0, 768.0

	var startPos types.Vec2
	acID := types.AircraftID(fmt.Sprintf("%s%03d", getRandomAirlinePrefix(), s.nextAircraftID))
	s.nextAircraftID++
	targetAlt := (float64(rand.Intn(20)) + 10) * 1000.0 // 10,000 to 30,000 ft
	startSpeed := 200.0 + rand.Float64()*100.0          // 200-300 knots

	// Randomly choose an edge to spawn from
	edge := rand.Intn(4) // 0: Top, 1: Right, 2: Bottom, 3: Left
	switch edge {
	case 0: // Top
		startPos = types.NewVec2(s.randomFloatInRange(minX, maxX), minY)
	case 1: // Right
		startPos = types.NewVec2(maxX, s.randomFloatInRange(minY, maxY))
	case 2: // Bottom
		startPos = types.NewVec2(s.randomFloatInRange(minX, maxX), maxY)
	case 3: // Left
		startPos = types.NewVec2(minX, s.randomFloatInRange(minY, maxY))
	}

	var initialHeading float64
	var entryWpName, exitWpName string

	if len(s.Airspace.EntryWaypoints) == 0 {
		log.Println("WARNING: No entry waypoints defined, spawining at generic location")
	} else {
		entryWpName = s.Airspace.EntryWaypoints[rand.Intn(len(s.Airspace.EntryWaypoints))]
		entryWp, ok := s.Airspace.Waypoints[entryWpName]
		if !ok {
			log.Printf("WARNING: No entry waypoints defined, spawining at generic location")
			initialHeading = s.randomFloatInRange(0.0, 360.0)
		} else {
			initialHeading = startPos.HeadingTo(entryWp.Position)
		}
	}

	if len(s.Airspace.ExitWaypoints) > 0 {
		for {
			exitWpName = s.Airspace.ExitWaypoints[rand.Intn(len(s.Airspace.ExitWaypoints))]
			if exitWpName != entryWpName || len(s.Airspace.ExitWaypoints) == 1 {
				break
			}
		}
	} else {
		log.Println("WARNING: no exit waypoints defined, aircraft will have no clear destination.")
	}

	flightPlanSegments := []flightplan.FlightPlanSegment{
		{
			WaypointName:   entryWpName,
			TargetAltitude: targetAlt,
			TargetSpeed:    startSpeed,
		},
	}

	waypointNames := make([]string, 0, len(s.Airspace.Waypoints))
	addedWaypoints := make([]string, 0)
	for k := range s.Airspace.Waypoints {
		waypointNames = append(waypointNames, k)
	}
	retries := 8
	for len(flightPlanSegments) < 3 {
		wpName := waypointNames[rand.Intn(len(waypointNames))]
		if wpName == entryWpName || wpName == exitWpName || slices.Contains(addedWaypoints, wpName) {
			retries--
			if retries <= 0 {
				break
			}
			continue
		}

		flightPlanSegments = append(flightPlanSegments, flightplan.FlightPlanSegment{
			WaypointName:   wpName,
			TargetAltitude: targetAlt * (rand.Float64()/2 + 0.75),
			TargetSpeed:    startSpeed * (rand.Float64()/2 + 0.75),
		})
		addedWaypoints = append(addedWaypoints, wpName)
	}

	flightPlanSegments = append(flightPlanSegments, flightplan.FlightPlanSegment{
		WaypointName:   exitWpName,
		TargetAltitude: targetAlt * (rand.Float64()/2 + 0.75),
		TargetSpeed:    startSpeed * (rand.Float64()/2 + 0.75),
	})

	flightPlan := &flightplan.FlightPlan{
		OriginAirportID:      "RANDOM",
		DestinationAirportID: exitWpName,
		Callsign:             acID,
		Route:                flightPlanSegments,
		CurrentSegmentIndex:  0,
	}

	ac := aircraft.NewAircraft(
		acID,
		startPos,
		initialHeading,
		startSpeed,
		targetAlt,
		aircraft.CRUISE,
		flightPlan,
		s.getWaypoint,
		s.AddRadioMessage,
	)
	s.Aircrafts[acID] = ac
	log.Printf("Spawned aircraft %s (Filed for %s) at %v, heading %.0f, speed %.0f, altitude %.0f", ac.ID, exitWpName, ac.Position, ac.Heading, ac.Speed, ac.Altitude)
}

func getRandomAirlinePrefix() string {
	prefixes := []string{"AAL", "SWA", "DAL", "UAL", "JBU", "ASA", "FFT", "AI", "JAL"}
	return prefixes[rand.Intn(len(prefixes))]
}

func (s *Simulation) CheckForConflicts() {
	aircraftSlice := []*aircraft.Aircraft{}
	for _, ac := range s.Aircrafts {
		aircraftSlice = append(aircraftSlice, ac)
	}

	for i := 0; i < len(aircraftSlice); i++ {
		for j := i + 1; j < len(aircraftSlice); j++ {
			ac1 := aircraftSlice[i]
			ac2 := aircraftSlice[j]
			if conflict.CheckSeparation(ac1, ac2) {
				log.Printf("CONFLICT: %s and %s", ac1.ID, ac2.ID)
				ac1.IsConflicting = true // Mark for visual warning
				ac2.IsConflicting = true
			}
		}
	}
}

func (s *Simulation) CleanupAircraft() {
	minX, maxX := -50.0, 1074.0 // Slightly outside screen
	minY, maxY := -50.0, 818.0

	for id, ac := range s.Aircrafts {
		if ac.Position.X < minX || ac.Position.X > maxX || ac.Position.Y < minY || ac.Position.Y > maxY {
			// Only count as missed handoff if it wasn't already handed off
			// You'll need a mechanism to check if it was 'expected' to be handed off.
			// For simplicity, for now, any exit without HandOffAircraft call is a "missed".
			if ac.FlightPlan != nil && ac.FlightPlan.CurrentSegmentIndex < len(ac.FlightPlan.Route) {
				// Aircraft didn't complete its plan or wasn't at exit
				log.Printf("MISSED HANDOFF: Aircraft %s left airspace without proper handoff!", id)
				s.MissedHandoffs++ // Add MissedHandoffs to Simulation struct
			} else {
				log.Printf("Aircraft %s left airspace and removed.", id)
			}

			delete(s.Aircrafts, id)
		}
	}
}

func (s *Simulation) IssueHeading(aircraftID types.AircraftID, heading float64) error {
	if ac, ok := s.Aircrafts[aircraftID]; ok {
		ac.SetHeading(heading)
		return nil
	}
	return fmt.Errorf("aircraft %s not found", aircraftID)
}

func (s *Simulation) IssueAltitude(aircraftID types.AircraftID, altitude float64) error {
	if ac, ok := s.Aircrafts[aircraftID]; ok {
		ac.SetAltitude(altitude)
		return nil
	}
	return fmt.Errorf("aircraft %s not found", aircraftID)
}

func (s *Simulation) IssueSpeed(aircraftID types.AircraftID, speed float64) error {
	if ac, ok := s.Aircrafts[aircraftID]; ok {
		ac.SetSpeed(speed)
		return nil
	}
	return fmt.Errorf("aircraft %s not found", aircraftID)
}

func (s *Simulation) IssueDirectTo(aircraftID types.AircraftID, wp *types.Waypoint) error {
	if ac, ok := s.Aircrafts[aircraftID]; ok {
		ac.SetDirectTo(wp)
		return nil
	}
	return fmt.Errorf("aircraft %s not found", aircraftID)
}
