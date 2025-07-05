package ui

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type TextInput struct {
	Text     string
	IsActive bool
	X, Y     int
	Width    int
	Height   int
	OnSubmit func(string)
}

func NewTextInput(x, y, width, height int, onSubmit func(string)) *TextInput {
	return &TextInput{
		Text:     "",
		IsActive: false,
		X:        x,
		Y:        y,
		Width:    width,
		Height:   height,
		OnSubmit: onSubmit,
	}
}

func (ti *TextInput) Update() {
	if !ti.IsActive {
		return
	}

	ti.Text += string(ebiten.AppendInputChars(nil))

	if ebiten.IsKeyPressed(ebiten.KeyBackspace) {
		if len(ti.Text) > 0 {
			ti.Text = ti.Text[:len(ti.Text)-1]
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		if ti.OnSubmit != nil {
			ti.OnSubmit(strings.TrimSpace(ti.Text))
		}
		ti.Text = ""
		ti.IsActive = false
	}
}

func (ti *TextInput) Draw(screen *ebiten.Image) {
	bgColor := color.RGBA{50, 50, 50, 255}
	if ti.IsActive {
		bgColor = color.RGBA{80, 80, 80, 255} // Darker when active
	}
	vector.DrawFilledRect(screen, float32(ti.X), float32(ti.Y), float32(ti.Width), float32(ti.Height), bgColor, false)

	// Draw border
	vector.DrawFilledRect(screen, float32(ti.X), float32(ti.Y), float32(ti.Width), 1, color.White, false)
	vector.DrawFilledRect(screen, float32(ti.X), float32(ti.Y+ti.Height-1), float32(ti.Width), 1, color.White, false)
	vector.DrawFilledRect(screen, float32(ti.X), float32(ti.Y), 1, float32(ti.Height), color.White, false)
	vector.DrawFilledRect(screen, float32(ti.X+ti.Width-1), float32(ti.Y), 1, float32(ti.Height), color.White, false)

	// Draw text
	displayTxt := ti.Text
	if ti.IsActive {
		displayTxt += "_" // Cursor
	}

	ebitenutil.DebugPrintAt(screen, displayTxt, ti.X+5, ti.Y+(ti.Height-16)/2) // Adjust for font height
}

// IsClicked checks if the mouse click is within the text input bounds
func (ti *TextInput) IsClicked(mouseX, mouseY int) bool {
	return mouseX >= ti.X && mouseX <= ti.X+ti.Width &&
		mouseY >= ti.Y && mouseY <= ti.Y+ti.Height
}
