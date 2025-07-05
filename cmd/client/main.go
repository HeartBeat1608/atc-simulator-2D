package main

import (
	"atc-simulator/internal/game/aircraft"
	"atc-simulator/internal/game/simulation"
	"atc-simulator/internal/ui"
	"atc-simulator/pkg/types"
	"fmt"
	"image/color"
	_ "image/png"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/labstack/gommon/log"
)

type Camera struct {
	X, Y                 float64
	PanStartX, PanStartY int
	Scale                float64
}

type Game struct {
	width, height int
	camera        *Camera
	sim           *simulation.Simulation
	aircraftImage *ebiten.Image

	selectedAircraftID types.AircraftID
	commandInput       *ui.TextInput
}

func NewGame(screenWidth, screenHeight int) *Game {
	game := &Game{
		sim:    simulation.NewSimulation(60.0),
		camera: &Camera{0, 0, 0, 0, 1.0},
		width:  screenWidth,
		height: screenHeight,
	}

	var err error
	game.aircraftImage, _, err = ebitenutil.NewImageFromFile("internal/assets/images/aircraft.png")
	if err != nil {
		log.Fatal(err)
	}

	game.commandInput = ui.NewTextInput(10, screenHeight-48, screenWidth/2, 30, func(cmd string) {
		game.parseAndExecuteCommand(cmd)
	})

	return game
}

func (g *Game) Update() error {
	dt := 1.0 / g.sim.TickRate
	g.sim.Update(dt)

	g.handleInput()
	g.commandInput.Update()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	g.drawAirspace(screen)

	for _, ac := range g.sim.Aircrafts {
		g.drawAircraft(screen, ac)
	}

	g.drawUI(screen)
	ebitenutil.DebugPrint(screen, "FPS: "+strconv.FormatFloat(ebiten.ActualFPS(), 'f', 2, 64))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.width, g.height
}

func (g *Game) handleInput() {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		if g.commandInput.IsClicked(x, y) {
			g.commandInput.IsActive = true
			return
		} else {
			g.commandInput.IsActive = false
		}

		clickedPos := types.NewVec2(float64(x), float64(y))
		g.selectedAircraftID = "" // Clear current selection

		// Check for aircraft hit
		for _, ac := range g.sim.Aircrafts {
			// A simple bounding box check (adjust size based on your aircraft sprite)
			acWidth, acHeight := float64(g.aircraftImage.Bounds().Dx()), float64(g.aircraftImage.Bounds().Dy())
			if clickedPos.X >= ac.Position.X-acWidth/2 && clickedPos.X <= ac.Position.X+acWidth/2 &&
				clickedPos.Y >= ac.Position.Y-acHeight/2 && clickedPos.Y <= ac.Position.Y+acHeight/2 {
				g.selectedAircraftID = ac.ID
				log.Printf("Selected aircraft: %s", g.selectedAircraftID)
				break
			}
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyH) && g.selectedAircraftID != "" {
		if ac, ok := g.sim.Aircrafts[g.selectedAircraftID]; ok {
			newHeading := math.Mod(ac.Heading+45, 360)
			g.sim.IssueHeading(g.selectedAircraftID, newHeading)
			log.Printf("%s: Turning to %.0f", g.selectedAircraftID, newHeading)
		}
	}

	_, wy := ebiten.Wheel()
	if wy != 0 {
		cursorX, cursorY := ebiten.CursorPosition()
		worldX, worldY := g.screenToWorld(float64(cursorX), float64(cursorY))

		oldScale := g.camera.Scale
		if wy > 0 {
			oldScale *= 1.1
		} else {
			oldScale /= 1.1
		}
		g.camera.Scale = math.Max(0.5, math.Min(3.0, oldScale))

		newWorldX, newWorldY := g.screenToWorld(float64(cursorX), float64(cursorY))
		g.camera.X -= (newWorldX - worldX)
		g.camera.Y -= (newWorldY - worldY)
	}

	// Right mouse button for pan
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		dx, dy := ebiten.CursorPosition()
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			// Store initial click position for panning
			g.camera.PanStartX, g.camera.PanStartY = dx, dy
		} else {
			g.camera.X -= float64(dx-g.camera.PanStartX) / g.camera.Scale
			g.camera.Y -= float64(dy-g.camera.PanStartY) / g.camera.Scale
			g.camera.PanStartX, g.camera.PanStartY = dx, dy // Update for next frame
		}
	}
}

// Helper: Convert screen coordinates to world coordinates
func (g *Game) screenToWorld(sx, sy float64) (wx, wy float64) {
	wx = sx/g.camera.Scale + g.camera.X
	wy = sy/g.camera.Scale + g.camera.Y
	return
}

// Helper: Convert world coordinates to screen coordinates
func (g *Game) worldToScreen(wx, wy float64) (sx, sy float64) {
	sx = (wx - g.camera.X) * g.camera.Scale
	sy = (wy - g.camera.Y) * g.camera.Scale
	return
}

func (g *Game) drawAircraft(screen *ebiten.Image, ac *aircraft.Aircraft) {
	screenX, screenY := g.worldToScreen(ac.Position.X, ac.Position.Y)

	rotation := (ac.Heading - 0) * math.Pi / 100.0

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(g.aircraftImage.Bounds().Dx()/2), -float64(g.aircraftImage.Bounds().Dy()/2))
	op.GeoM.Rotate(rotation)
	op.GeoM.Scale(g.camera.Scale, g.camera.Scale)
	op.GeoM.Translate(screenX, screenY)

	if g.selectedAircraftID == ac.ID {
		op.ColorScale.SetA(0.8)
		vector.StrokeRect(screen, float32(ac.Position.X-10), float32(ac.Position.Y-10), 20, 20, 1, color.RGBA{255, 255, 255, 255}, false)
	}

	screen.DrawImage(g.aircraftImage, op)

	lineLength := 30.0 * g.camera.Scale // Line length scales with zoom
	radians := ac.Heading * math.Pi / 180.0
	endWorldX := ac.Position.X + lineLength/g.camera.Scale*math.Sin(radians) // Calculate end point in world coords
	endWorldY := ac.Position.Y - lineLength/g.camera.Scale*math.Cos(radians)
	endScreenX, endScreenY := g.worldToScreen(endWorldX, endWorldY)
	vector.StrokeLine(screen, float32(screenX), float32(screenY), float32(endScreenX), float32(endScreenY), 1, color.RGBA{100, 100, 255, 255}, false)

	tagText := fmt.Sprintf("%s\nALT:%.0f (%.0f)\nSPD:%.0f (%.0f)\nHDG:%.0f (%.0f)\nSTS: %s",
		ac.ID, ac.Altitude, ac.TargetAltitude,
		ac.Speed, ac.TargetSpeed,
		ac.Heading, ac.TargetHeading, aircraft.StateStringMap[ac.State])

	ebitenutil.DebugPrintAt(screen, tagText, int(ac.Position.X)+10, int(ac.Position.Y)-20)

	// Conflict highlight (also relative to screenX, screenY)
	if ac.IsConflicting {
		vector.DrawFilledCircle(screen, float32(screenX), float32(screenY), float32(10*g.camera.Scale), color.RGBA{255, 0, 0, 100}, false)
	}
}

func (g *Game) drawAirspace(screen *ebiten.Image) {
	// Convert Waypoint positions
	for _, wp := range g.sim.Airspace.Waypoints {
		screenX, screenY := g.worldToScreen(wp.Position.X, wp.Position.Y)
		vector.DrawFilledCircle(screen, float32(screenX), float32(screenY), float32(3*g.camera.Scale), color.RGBA{0, 255, 255, 255}, false)
		ebitenutil.DebugPrintAt(screen, wp.Name, int(screenX)+5, int(screenY)+5)
	}

	// Convert Sector bounds
	sector := g.sim.Airspace.Sectors["SECTOR1"]
	if sector != nil && len(sector.Bounds) >= 2 {
		for i := 0; i < len(sector.Bounds); i++ {
			p1World := sector.Bounds[i]
			p2World := sector.Bounds[(i+1)%len(sector.Bounds)]
			p1ScreenX, p1ScreenY := g.worldToScreen(p1World.X, p1World.Y)
			p2ScreenX, p2ScreenY := g.worldToScreen(p2World.X, p2World.Y)
			vector.StrokeLine(screen, float32(p1ScreenX), float32(p1ScreenY), float32(p2ScreenX), float32(p2ScreenY), float32(1*g.camera.Scale), color.RGBA{0, 100, 0, 255}, false)
		}
	}
}

func (g *Game) drawUI(screen *ebiten.Image) {
	g.commandInput.Draw(screen)

	selectedAcText := "Selected: None"
	if g.selectedAircraftID != "" {
		selectedAcText = "Selected: " + string(g.selectedAircraftID)
	}

	ebitenutil.DebugPrintAt(screen, selectedAcText, 10, 700)
}

func (g *Game) parseAndExecuteCommand(cmd string) {
	parts := strings.Fields(cmd) // Split by whitespace
	if len(parts) < 2 {
		log.Printf("Invalid command format: %s. Expected: [<Callsign>] <Command> <Value>", cmd)
		return
	}

	var aircraftID types.AircraftID
	var commandType, valueStr string

	if len(parts) == 2 && g.selectedAircraftID == "" {
		log.Printf("No Aircraft selected")
		return
	} else if len(parts) == 2 {
		aircraftID = g.selectedAircraftID
		commandType = strings.ToUpper(parts[0])
		valueStr = parts[1]
	} else {
		aircraftID = types.AircraftID(strings.ToUpper(parts[0]))
		commandType = strings.ToUpper(parts[1])
		valueStr = parts[2]
	}

	_, exists := g.sim.Aircrafts[aircraftID]
	if !exists {
		log.Printf("Aircraft %s not found.", aircraftID)
		return
	}

	switch commandType {
	case "H", "HEADING":
		heading, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || heading < 0 || heading >= 360 {
			log.Printf("Invalid heading value: %s. Must be 0-359.", valueStr)
			return
		}
		g.sim.IssueHeading(aircraftID, heading)
		log.Printf("Issued H %.0f to %s", heading, aircraftID)
	case "A", "ALT", "ALTITUDE":
		altitude, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || altitude < 0 { // Add more realistic altitude bounds
			log.Printf("Invalid altitude value: %s. Must be positive.", valueStr)
			return
		}
		g.sim.IssueAltitude(aircraftID, altitude)
		log.Printf("Issued A %.0f to %s", altitude, aircraftID)
	case "S", "SPD", "SPEED":
		speed, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || speed < 0 { // Add realistic speed bounds
			log.Printf("Invalid speed value: %s. Must be positive.", valueStr)
			return
		}
		g.sim.IssueSpeed(aircraftID, speed)
		log.Printf("Issued S %.0f to %s", speed, aircraftID)
	case "D", "DIRECT":
		waypointName := strings.ToUpper(valueStr)
		wp, ok := g.sim.Airspace.Waypoints[waypointName]
		if !ok {
			log.Printf("Waypoint %s not found.", waypointName)
			return
		}
		g.sim.IssueDirectTo(aircraftID, wp)
		log.Printf("Issued D %s to %s", waypointName, aircraftID)
	default:
		log.Printf("Unknown command type: %s", commandType)
	}
}

func main() {
	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowTitle("ATC Simulator")
	ebiten.SetVsyncEnabled(true)

	game := NewGame(1024, 768)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
