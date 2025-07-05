package airspace

import "atc-simulator/pkg/types"

type Sector struct {
	Name        string
	Bounds      []types.Vec2
	MinAltitude float64
	MaxAltitude float64
}

type Airspace struct {
	Waypoints map[string]*types.Waypoint
	Sectors   map[string]*Sector
	Runways   map[string]*types.Vec2
}

func NewAirspace() *Airspace {
	ap := &Airspace{
		Waypoints: make(map[string]*types.Waypoint),
		Sectors:   make(map[string]*Sector),
		Runways:   make(map[string]*types.Vec2),
	}

	ap.Waypoints["WAYPT1"] = &types.Waypoint{Name: "WAYPT1", Position: types.NewVec2(200, 200)}
	ap.Waypoints["WAYPT2"] = &types.Waypoint{Name: "WAYPT2", Position: types.NewVec2(800, 200)}
	ap.Waypoints["WAYPT3"] = &types.Waypoint{Name: "WAYPT3", Position: types.NewVec2(800, 600)}
	ap.Waypoints["WAYPT4"] = &types.Waypoint{Name: "WAYPT4", Position: types.NewVec2(200, 600)}

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
