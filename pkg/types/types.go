package types

import "math"

type AircraftID string

type Vec2 struct {
	X float64
	Y float64
}

func NewVec2(x, y float64) Vec2 {
	return Vec2{x, y}
}

func (v1 Vec2) DistanceTo(v2 Vec2) float64 {
	dx := v1.X - v2.X
	dy := v1.Y - v2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (v Vec2) HeadingTo(target Vec2) float64 {
	dx := target.X - v.X
	dy := target.Y - v.Y

	angleFromPositiveX := math.Atan2(dy, dx)

	angleDegrees := angleFromPositiveX * 180.0 / math.Pi

	aviationHeading := angleDegrees + 90

	normalizedHeading := math.Mod(aviationHeading+360.0, 360.0)
	return normalizedHeading
}

type Waypoint struct {
	Name     string
	Position Vec2
}
