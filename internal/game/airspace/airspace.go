package airspace

import (
	"atc-simulator/pkg/types"

	"github.com/hajimehoshi/ebiten/v2"
)

type Sector struct {
	Name        string
	Bounds      []types.Vec2
	MinAltitude float64
	MaxAltitude float64
}

type Airspace struct {
	Waypoints map[string]*types.Waypoint
	Sectors   map[string]*Sector
	Airports  map[string]*Airport

	ExitWaypoints  []string
	EntryWaypoints []string
}

func NewAirspace() *Airspace {
	ap := &Airspace{
		Waypoints: make(map[string]*types.Waypoint),
		Sectors:   make(map[string]*Sector),
		Airports:  make(map[string]*Airport),

		EntryWaypoints: []string{"APIPO", "BISKET", "EMETI", "FILKA"},
		ExitWaypoints:  []string{"APIPO", "BISKET", "EMETI", "FILKA"},
	}

	screenWidth, screenHeight := ebiten.WindowSize()

	ap.Waypoints["APIPO"] = &types.Waypoint{Name: "APIPO", Position: types.NewVec2(float64(screenWidth)*0.1, float64(screenHeight)*0.14)}
	ap.Waypoints["BISKET"] = &types.Waypoint{Name: "BISKET", Position: types.NewVec2(float64(screenWidth)*0.64, float64(screenHeight)*0.23)}
	ap.Waypoints["CIPKA"] = &types.Waypoint{Name: "CIPKA", Position: types.NewVec2(float64(screenWidth)*0.37, float64(screenHeight)*0.54)}
	ap.Waypoints["EMETI"] = &types.Waypoint{Name: "EMETI", Position: types.NewVec2(float64(screenWidth)*0.25, float64(screenHeight)*0.90)}
	ap.Waypoints["FILKA"] = &types.Waypoint{Name: "FILKA", Position: types.NewVec2(float64(screenWidth)*0.67, float64(screenHeight)*0.65)}

	// Example: Define a simple rectangular sector
	ap.Sectors["SECTOR1"] = &Sector{
		Name: "SECTOR1",
		Bounds: []types.Vec2{
			types.NewVec2(0, 0),
			types.NewVec2(1024, 0),
			types.NewVec2(1024, 768),
			types.NewVec2(0, 768),
		},
		MinAltitude: 0,
		MaxAltitude: 40000,
	}
	return ap
}
