package airspace

import "atc-simulator/pkg/types"

type Runway struct {
	Name      string
	Threshold types.Vec2
	Heading   float64
	Length    float64
	AirportID string
}

type Airport struct {
	ID       string
	Name     string
	Position types.Vec2
	Runways  map[string]*Runway
}

func (ap *Airspace) AddAirport(airportID, name string, pos types.Vec2, runways []Runway) {
	airport := &Airport{
		ID:       airportID,
		Name:     name,
		Position: pos,
		Runways:  make(map[string]*Runway),
	}

	for _, rwy := range runways {
		rwy.AirportID = airportID
		airport.Runways[rwy.Name] = &rwy
	}
	ap.Airports[airportID] = airport
}
