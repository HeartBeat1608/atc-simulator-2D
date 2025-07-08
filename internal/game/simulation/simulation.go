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

	"github.com/hajimehoshi/ebiten/v2"
)

type Simulation struct {
	Aircrafts       map[types.AircraftID]*aircraft.Aircraft
	Airspace        *airspace.Airspace
	TickRate        float64
	TimeOfDay       time.Time
	GameTimeSeconds float64

	HandOffs       int
	MissedHandoffs int
	Conflicts      int
	Landings       int

	RadioLog        []RadioMessage
	maxRadioLogSize int

	lastSpawnTime        time.Time
	spawnInterval        time.Duration
	nextAircraftID       int
	maxAircraftsOnScreen int
	landingProbability   float64

	WorldToScreen func(wx float64, wy float64) (sx float64, sy float64)
	ScreenToWorld func(wx float64, wy float64) (sx float64, sy float64)
}

func NewSimulation(tickRate float64) *Simulation {
	simpleAirspace := airspace.NewAirspace()

	kiaAirport := airspace.Airport{
		ID:       "KBLR",
		Name:     "Kempegowda International Airport",
		Position: types.NewVec2(512, 384),
		Runways: map[string]*airspace.Runway{
			"RWY09": {Name: "RWY09", Threshold: types.NewVec2(200, 384), Heading: 90},
			"RWY27": {Name: "RWY09", Threshold: types.NewVec2(824, 384), Heading: 270},
		},
	}

	simpleAirspace.AddAirport(kiaAirport.ID, kiaAirport.Name, kiaAirport.Position, []airspace.Runway{
		*kiaAirport.Runways["RWY09"],
		*kiaAirport.Runways["RWY27"],
	})

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
		landingProbability:   0.8,

		HandOffs:       0,
		MissedHandoffs: 0,
		Conflicts:      0,
	}

	s.SpawnRandomAircraft()
	return s
}

func (s *Simulation) Update(dt float64) {
	s.GameTimeSeconds += dt
	for id, ac := range s.Aircrafts {
		ac.Update(dt)
		ac.IsConflicting = false

		if ac.FlightPlan != nil && ac.FlightPlan.CurrentSegmentIndex >= len(ac.FlightPlan.Route) {
			if ac.State == aircraft.LANDED {
				if _, exists := s.Aircrafts[id]; exists {
					s.LandAircraft(id)
					delete(s.Aircrafts, id)
				}
			} else if ac.ClearedForHandoff {
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
					continue
				}
			} else if ac.ClearedForLanding {
				if ac.State == aircraft.LANDED {
					s.LandAircraft(ac.ID)
					continue
				}
			}
		} else {
			// Aircraft completed plan but not cleared for handoff/landing.
			// This is where a penalty could occur later if it leaves the zone.
			// For now, it will just keep flying straight until CleanupAircraft gets it.
			// Or, the aircraft might start requesting guidance here.
			continue
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

func (s *Simulation) ClearLanding(aircraftID types.AircraftID, runwayName string) bool {
	ac, ok := s.Aircrafts[aircraftID]
	if !ok {
		log.Printf("ClearLanding: Aircraft %s not found.", aircraftID)
		return false
	}

	var targetRunway *airspace.Runway
	for _, airport := range s.Airspace.Airports {
		if rwy, found := airport.Runways[runwayName]; found {
			targetRunway = rwy
			break
		}
	}

	if targetRunway == nil {
		s.AddRadioMessage("ATC", fmt.Sprintf("Negative, %s, runway %s is not valid.", ac.ID, runwayName), true)
		return false // Runway not found
	}

	if ac.ClearedForLanding {
		// Already cleared, confirm it
		s.AddRadioMessage("ATC", fmt.Sprintf("Confirming landing clearance for %s on %s.", ac.ID, runwayName), false)
		return true
	}

	landingSegment := flightplan.FlightPlanSegment{
		Type:           flightplan.SegmentTypeLanding,
		AirportID:      targetRunway.AirportID, // Assuming AirportID for runway is its name for simplicity,
		RunwayName:     targetRunway.Name,      // Or you might use Airport.ID here
		TargetAltitude: 2000,                   // Standard approach altitude
		TargetSpeed:    200,                    // Standard approach speed
	}

	ac.FlightPlan.Route = []flightplan.FlightPlanSegment{landingSegment}
	ac.FlightPlan.CurrentSegmentIndex = 0 // Reset to start new plan

	ac.LandingRunway = targetRunway
	ac.ClearedForLanding = true
	ac.PreviousAltitudeRequest = false
	ac.PreviousSpeedRequest = false

	s.AddRadioMessage("ATC", fmt.Sprintf("%s, cleared for ILS approach runway %s.", ac.ID, runwayName), false)
	return true
}

func (s *Simulation) LandAircraft(aircraftID types.AircraftID) {
	if ac, ok := s.Aircrafts[aircraftID]; ok {
		log.Printf("SCORE: Aircraft %s successfully landed.", ac.ID)
		s.Landings++
	}
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

func (s *Simulation) SpawnRandomAircraft() {
	// Define spawn points (e.g., edges of your 1024x768 screen)
	screenWidth, screenHeight := ebiten.WindowSize()
	minX, maxX := 100.0, float64(screenWidth)-100.0
	minY, maxY := 100.0, float64(screenHeight)-100.0

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

	isLandingAircraft := rand.Float64() < s.landingProbability
	if isLandingAircraft {
		waypointNames := make([]string, 0, len(s.Airspace.Waypoints))
		addedWaypoints := make([]string, 0)
		for k := range s.Airspace.Waypoints {
			waypointNames = append(waypointNames, k)
		}
		retries := 8
		for len(flightPlanSegments) < 2 {
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
	}

	fpLastSegment := flightplan.FlightPlanSegment{
		WaypointName:   exitWpName,
		TargetAltitude: targetAlt * (rand.Float64()/2 + 0.75),
		TargetSpeed:    startSpeed * (rand.Float64()/2 + 0.75),
	}

	if isLandingAircraft && len(s.Airspace.Airports) > 0 {
		airportIDs := make([]string, 0, len(s.Airspace.Airports))
		for id := range s.Airspace.Airports {
			airportIDs = append(airportIDs, id)
		}

		targetAirportID := airportIDs[rand.Intn(len(airportIDs))]
		targetAirport := s.Airspace.Airports[targetAirportID]

		runwaysNames := make([]string, 0, len(targetAirport.Runways))
		for name := range targetAirport.Runways {
			runwaysNames = append(runwaysNames, name)
		}
		targetRunwayName := runwaysNames[rand.Intn(len(runwaysNames))]

		fpLastSegment.RunwayName = targetRunwayName
		fpLastSegment.AirportID = targetAirportID
		fpLastSegment.Type = flightplan.SegmentTypeLanding
		fpLastSegment.TargetAltitude = 2000
		fpLastSegment.TargetSpeed = 225
	}

	flightPlanSegments = append(flightPlanSegments, fpLastSegment)

	flightPlan := &flightplan.FlightPlan{
		OriginAirportID:      "RANDOM",
		DestinationAirportID: fpLastSegment.WaypointName,
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
		s.Airspace,
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
	screenWidth, screenHeight := ebiten.WindowSize()
	buffer := 100.0

	worldLeft, worldTop := s.ScreenToWorld(0, 0)
	worldRight, worldBottom := s.ScreenToWorld(float64(screenWidth), float64(screenHeight))

	worldMinX := worldLeft - buffer
	worldMinY := worldRight + buffer
	worldMaxX := worldTop - buffer
	worldMaxY := worldBottom + buffer

	for id, ac := range s.Aircrafts {
		if time.Since(ac.SpawnTime) < time.Minute || ac.DirectToWaypoint != nil {
			// skip cleanup for first 1 minute of ops (avoids unnecessary checks)
			continue
		}

		if ac.Position.X < worldMinX || ac.Position.X > worldMaxX || ac.Position.Y < worldMinY || ac.Position.Y > worldMaxY || ac.State != aircraft.LANDED {
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
		if ac.DirectToWaypoint != nil {
			ac.DirectToWaypoint = nil
		}
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
		if ac.FlightPlan != nil && ac.FlightPlan.CurrentSegmentIndex < len(ac.FlightPlan.Route) {
			wpIdx := -1
			for i, r := range ac.FlightPlan.Route {
				if r.WaypointName == wp.Name {
					wpIdx = i
					break
				}
			}

			if wpIdx != -1 {
				ac.FlightPlan.CurrentSegmentIndex = wpIdx
			}
		}

		ac.SetDirectTo(wp)
		return nil
	}
	return fmt.Errorf("aircraft %s not found", aircraftID)
}

func (s *Simulation) ClearHandoff(aircraftID types.AircraftID) bool {
	ac, ok := s.Aircrafts[aircraftID]
	if !ok {
		log.Printf("ClearHandoff: Aircraft %s not found.", aircraftID)
		return false
	}

	if ac.FlightPlan == nil || ac.FlightPlan.CurrentSegmentIndex < len(ac.FlightPlan.Route) {
		s.AddRadioMessage("ATC", fmt.Sprintf("Negative, %s, you are not ready for handoff.", ac.ID), true)
		return false // Not at end of flight plan yet
	}

	if ac.ClearedForHandoff {
		s.AddRadioMessage("ATC", fmt.Sprintf("Confirming handoff clearance for %s, you are already cleared.", ac.ID), false)
		return true // Already cleared, no change
	}

	ac.ClearedForHandoff = true

	s.AddRadioMessage("ATC", fmt.Sprintf("%s, contact departure, good day.", ac.ID), false)
	return true
}
