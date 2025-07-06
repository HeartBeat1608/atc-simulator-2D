package ui

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(ti.Text) > 0 {
			ti.Text = ti.Text[:len(ti.Text)-1]
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if ti.OnSubmit != nil {
			ti.OnSubmit(strings.TrimSpace(ti.Text))
		}
		ti.Text = ""
		ti.IsActive = false
	}
}

func (ti *TextInput) Draw(screen *ebiten.Image, x, y, width, height int) {
	bgColor := color.RGBA{50, 50, 50, 255}
	if ti.IsActive {
		bgColor = color.RGBA{80, 80, 80, 255} // Darker when active
	}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(height), bgColor, false)

	// Draw border
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), 1, color.White, false)
	vector.DrawFilledRect(screen, float32(x), float32(y+height-1), float32(width), 1, color.White, false)
	vector.DrawFilledRect(screen, float32(x), float32(y), 1, float32(height), color.White, false)
	vector.DrawFilledRect(screen, float32(x+width-1), float32(y), 1, float32(height), color.White, false)

	// Draw text
	displayTxt := ti.Text
	if ti.IsActive {
		displayTxt += "_" // Cursor
	}

	ebitenutil.DebugPrintAt(screen, displayTxt, x+5, y+(height-16)/2)
}

// IsClicked checks if the mouse click is within the text input bounds
func (ti *TextInput) IsClicked(mouseX, mouseY, x, y, width, height int) bool {
	return mouseX >= x && mouseX <= x+width &&
		mouseY >= y && mouseY <= y+height
}
