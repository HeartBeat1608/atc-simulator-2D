package simulation

import (
	"atc-simulator/internal/game/aircraft"
	"atc-simulator/internal/game/airspace"
	"atc-simulator/internal/game/conflict"
	"atc-simulator/pkg/types"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type Simulation struct {
	Aircrafts map[types.AircraftID]*aircraft.Aircraft
	Airspace  *airspace.Airspace
	TickRate  float64
	TimeOfDay time.Time

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
		nextAircraftID:       1,
		maxAircraftsOnScreen: 12,
	}

	s.SpawnRandomAircraft()
	return s
}

func (s *Simulation) Update(dt float64) {
	for _, ac := range s.Aircrafts {
		ac.Update(dt)
		ac.IsConflicting = false
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

func (s *Simulation) randomFloatInRange(minF, maxF float64) float64 {
	fRange := maxF - minF
	return minF + rand.Float64()*fRange
}

func (s *Simulation) SpawnRandomAircraft() {
	// Define spawn points (e.g., edges of your 1024x768 screen)
	minX, maxX := 10.0, 1024.0
	minY, maxY := 10.0, 768.0

	var startPos types.Vec2
	var initialHeading float64

	// Randomly choose an edge to spawn from
	edge := rand.Intn(4) // 0: Top, 1: Right, 2: Bottom, 3: Left
	switch edge {
	case 0: // Top
		startPos = types.NewVec2(s.randomFloatInRange(minX, maxX), minY)
		initialHeading = s.randomFloatInRange(45, 135) // Towards bottom half
	case 1: // Right
		startPos = types.NewVec2(maxX, s.randomFloatInRange(minY, maxY))
		initialHeading = s.randomFloatInRange(135, 225) // Towards left half
	case 2: // Bottom
		startPos = types.NewVec2(s.randomFloatInRange(minX, maxX), maxY)
		initialHeading = s.randomFloatInRange(225, 315) // Towards top half
	case 3: // Left
		startPos = types.NewVec2(minX, s.randomFloatInRange(minY, maxY))
		initialHeading = s.randomFloatInRange(-45, 45) // Towards right half (0-45 or 315-0)
	}

	acID := types.AircraftID(fmt.Sprintf("%s%03d", getRandomAirlinePrefix(), s.nextAircraftID))
	s.nextAircraftID++
	targetAlt := (float64(rand.Intn(20)) + 10) * 1000.0 // 10,000 to 30,000 ft
	startSpeed := 200.0 + rand.Float64()*100.0          // 200-300 knots

	ac := aircraft.NewAircraft(acID, startPos, initialHeading, startSpeed, targetAlt, aircraft.CRUISE, nil)
	s.Aircrafts[acID] = ac
	log.Printf("Spawned aircraft %s at %v, heading %.0f, speed %.0f, altitude %.0f", ac.ID, ac.Position, ac.Heading, ac.Speed, ac.Altitude)
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
			delete(s.Aircrafts, id)
			log.Printf("Aircraft %s left airspace and removed.", id)
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
