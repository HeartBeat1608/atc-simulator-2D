package conflict

import (
	"atc-simulator/internal/game/aircraft"
	"atc-simulator/pkg/types"
	"math"
)

const (
	MIN_HORIZONTAL_SEPARATION = 5.0
	MIN_VERTICAL_SEPARATION   = 1000.0
)

func CheckSeparation(ac1, ac2 *aircraft.Aircraft) bool {
	if math.Abs(ac1.Altitude-ac2.Altitude) < MIN_VERTICAL_SEPARATION {
		distSq := math.Pow(ac1.Position.X-ac2.Position.X, 2) + math.Pow(ac1.Position.Y-ac2.Position.Y, 2)
		minDistPixelSq := math.Pow(MIN_HORIZONTAL_SEPARATION*types.NM_TO_PIXEL, 2)
		if distSq < minDistPixelSq {
			return true
		}
	}
	return false
}

// PredictConflict projects aircraft positions and checks for separation.
// Returns: (isConflict, timeToConflict, collisionPoint1, collisionPoint2)
func PredictConflict(ac1, ac2 *aircraft.Aircraft, futureTimeSeconds float64) (bool, float64, types.Vec2, types.Vec2) {
	// Simple linear projection (ignores turns/climbs mid-projection)
	// For more accuracy, you'd integrate their Update() over small dt steps.

	// Calculate current speed in pixels/second for both aircraft
	speed1PixelsPerSec := ac1.Speed / 3600.0 * types.NM_TO_PIXEL
	speed2PixelsPerSec := ac2.Speed / 3600.0 * types.NM_TO_PIXEL

	// Calculate displacement for the futureTime
	radians1 := ac1.Heading * math.Pi / 180.0
	radians2 := ac2.Heading * math.Pi / 180.0

	deltaX1 := speed1PixelsPerSec * math.Sin(radians1) * futureTimeSeconds
	deltaY1 := -speed1PixelsPerSec * math.Cos(radians1) * futureTimeSeconds // Y-inverted

	deltaX2 := speed2PixelsPerSec * math.Sin(radians2) * futureTimeSeconds
	deltaY2 := -speed2PixelsPerSec * math.Cos(radians2) * futureTimeSeconds // Y-inverted

	projectedPos1 := types.NewVec2(ac1.Position.X+deltaX1, ac1.Position.Y+deltaY1)
	projectedPos2 := types.NewVec2(ac2.Position.X+deltaX2, ac2.Position.Y+deltaY2)

	// Projected altitudes
	altitudeChange1 := ac1.ClimbRate * futureTimeSeconds / 60.0
	altitudeChange2 := ac2.ClimbRate * futureTimeSeconds / 60.0
	projectedAlt1 := ac1.Altitude + altitudeChange1
	projectedAlt2 := ac2.Altitude + altitudeChange2

	// Check vertical separation first
	if math.Abs(projectedAlt1-projectedAlt2) < MIN_VERTICAL_SEPARATION {
		// Check horizontal separation
		distSq := math.Pow(projectedPos1.X-projectedPos2.X, 2) + math.Pow(projectedPos1.Y-projectedPos2.Y, 2)
		minDistPixelsSq := math.Pow(MIN_HORIZONTAL_SEPARATION*types.NM_TO_PIXEL, 2)

		if distSq < minDistPixelsSq {
			// A conflict is predicted within this futureTimeSeconds window
			// For timeToConflict, you'd need more advanced collision geometry (e.g., shortest distance between moving points)
			// For now, return futureTimeSeconds as a proxy
			return true, futureTimeSeconds, projectedPos1, projectedPos2
		}
	}
	return false, 0, types.Vec2{}, types.Vec2{} // No conflict
}
