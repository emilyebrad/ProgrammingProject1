package main

import (
	"fmt"
	"image"
	_ "image/png"
	"io"
	"log"
	"math/rand"
	_ "math/rand"
	"os"
	_ "time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	_ "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	_ "github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/solarlune/resolv"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

func main() {
	ebiten.SetWindowSize(1000, 1000)
	ebiten.SetWindowTitle("Barking Ball Game")
	backgroundPict, _, err := ebitenutil.NewImageFromFile("PixelGrassBackground.png")
	playerImage, _, err := ebitenutil.NewImageFromFile("Dog1.png")
	ballImage, _, err := ebitenutil.NewImageFromFile("tennis.png")
	barkImage, _, err := ebitenutil.NewImageFromFile("BarkWave.png")

	if err != nil {
		fmt.Println("Unable to load background image:", err)
	}
	if err != nil {
		log.Fatal(err)
	}
	audioContext := audio.NewContext(44100)
	barkFile, err := os.Open("dog-bark-effect.wav")
	if err != nil {
		fmt.Println("Cannot open dog-bark-effect.wav:", err)
	}
	defer barkFile.Close()
	stream, err := wav.DecodeWithSampleRate(44100, barkFile)
	if err != nil {
		fmt.Println("Failed to decode dog-bark-effect.wav:?, err")
	}
	barkSound, err := audioContext.NewPlayer(stream)
	if err != nil {
		fmt.Println("Failed to run game", err)
	}
	space := resolv.NewSpace(1000, 1000, 64, 64)
	ourGame := firstGame{
		player:      playerImage,
		background:  backgroundPict,
		ball:        ballImage,
		xloc:        465,
		yloc:        700,
		speed:       3,
		playerSpeed: 8,
		font:        LoadFont("Square-Black.ttf", 24),
		frameWidth:  65,
		frameHeight: 65,
		space:       space,
		barkImage:   barkImage,
		barkAudio:   barkSound,
	}
	playerShape := resolv.NewRectangle(float64(ourGame.xloc), float64(ourGame.yloc), float64(ourGame.frameWidth), float64(ourGame.frameHeight))
	space.Add(playerShape)
	ourGame.playerShape = playerShape
}

type firstGame struct {
	player          *ebiten.Image
	ball            *ebiten.Image
	balls           []*ball
	barks           []*bark
	background      *ebiten.Image
	backgroundXView int
	backgroundYView int
	state           gameState
	font            font.Face
	xloc            int
	yloc            int
	speed           int
	score           int
	playerSpeed     int
	playerFrame     int
	playerCounter   int
	playerShape     *resolv.ConvexPolygon
	frameWidth      int
	frameHeight     int
	space           *resolv.Space
	barkImage       *ebiten.Image
	barkAudio       *audio.Player
	audioContext    *audio.Context
}

type gameState int

type ball struct {
	x           float64
	y           float64
	speed       float64
	playerShape *resolv.ConvexPolygon
}

type bark struct {
	x           float64
	y           float64
	speed       float64
	playerShape *resolv.ConvexPolygon
}

const (
	gameStateStart gameState = iota
	gameStatePlay
	ballWidth  = 32
	ballHeight = 32
	barkWidth  = 32
	barkHeight = 32
)

func (f *firstGame) Update() error {
	if f.state == gameStateStart {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			f.state = gameStatePlay
		}
		return nil
	} else {
		f.backgroundYView -= 4
		backgroundHeight := f.background.Bounds().Dy() * 2
		if f.backgroundYView >= -backgroundHeight {
			f.backgroundYView -= backgroundHeight
		}
		//Makes sure the player can't move off the screen
		if f.xloc < 0 {
			f.xloc = 0
		}
		if f.xloc > 1000-32 {
			f.xloc = 1000 - 32
		}
		//Movement of player
		moving := false
		if ebiten.IsKeyPressed(ebiten.KeyA) {
			f.xloc -= f.speed
			moving = true
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) {
			f.xloc += f.speed
			moving = true
		}
		if moving {
			f.playerCounter++
			if f.playerCounter >= 6 {
				f.playerFrame = (f.playerFrame + 1) % 4
				f.playerCounter = 0
			}
		} else {
			f.playerCounter = 0
			f.playerFrame = 0
		}
		backgroundHeight = f.background.Bounds().Dy()
		maxY := backgroundHeight * 2
		f.backgroundYView += 1
		f.backgroundYView %= maxY

		//Spawns in balls in random positions
		if rand.Intn(60) == 0 {
			ball := &ball{
				x:     float64(rand.Intn(1000 - ballWidth)),
				y:     -float64(rand.Intn(ballHeight)),
				speed: 2,
			}
			ball.playerShape = resolv.NewRectangle(ball.x, ball.y, ballWidth, ballHeight)
			f.balls = append(f.balls, ball)
			f.space.Add(ball.playerShape)
		}

		//When player presses space the dog will send out a bark
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			bark := &bark{
				x:     float64(f.xloc + f.frameWidth/2),
				y:     float64(f.yloc),
				speed: 5,
			}
			bark.playerShape = resolv.NewRectangle(bark.x, bark.y, barkWidth, barkHeight)
			f.barks = append(f.barks, bark)
			f.space.Add(bark.playerShape)
			f.barkAudio.Rewind()
			f.barkAudio.Play()
		}

		//Updates the ball and bark positions each frame
		for _, ball := range f.balls {
			ball.y += ball.speed
			ball.playerShape.SetPosition(ball.x, ball.y)
		}
		for _, bark := range f.barks {
			bark.y -= bark.speed
			bark.playerShape.SetPosition(bark.x, bark.y)
		}

		var newBalls []*ball
		var newBarks []*bark

		/* Supposed to check for collisions and if a collision is detected it removes the balla and bark shape, increments the score, marks that the ball has hit something, and breaks to stop checking other balls
		(I sadly didn't give myself enough time to figure out how to correctly detect collisions, so I got stuck here)
		*/
		for _, bark := range f.barks {
			hit := false
			for _, ball := range f.balls {
				if bark.playerShape.Bounds().Intersects(ball.playerShape.Bounds()) {
					f.space.Remove(ball.playerShape)
					f.space.Remove(bark.playerShape)
					f.score++
					hit = true
					break
				}
			}
			if !hit {
				newBarks = append(newBarks, bark)
			}
		}
		for _, ball := range f.balls {
			newBalls = append(newBalls, ball)
		}
		f.balls = newBalls
		f.barks = newBarks
		return nil
	}
}

func (f firstGame) Draw(screen *ebiten.Image) {
	if f.state == gameStateStart {
		drawFace := text.NewGoXFace(f.font)
		textOpts := &text.DrawOptions{
			DrawImageOptions: ebiten.DrawImageOptions{},
			LayoutOptions:    text.LayoutOptions{},
		}
		textOpts.GeoM.Reset()
		textOpts.GeoM.Translate(50, 50)
		textOpts.ColorScale.ScaleWithColor(colornames.Lightblue)
		text.Draw(screen, "Controls: Use the arrow keys A and D to move", drawFace, textOpts)
		textOpts.GeoM.Reset()
		textOpts.GeoM.Translate(140, 100)
		text.Draw(screen, "Space to bark", drawFace, textOpts)
		textOpts.GeoM.Reset()
		textOpts.GeoM.Translate(140, 150)
		text.Draw(screen, "Balls that get past you will cost points", drawFace, textOpts)
		textOpts.GeoM.Reset()
		textOpts.GeoM.Translate(400, 450)
		textOpts.ColorScale.ScaleWithColor(colornames.Cadetblue)
		text.Draw(screen, "Press Space to Start", drawFace, textOpts)
	} else {
		backgroundWidth := f.background.Bounds().Dx()
		backgroundHeight := f.background.Bounds().Dy()
		scaleX := float64(1000) / float64(backgroundWidth)
		scaleY := float64(1000) / float64(backgroundHeight)
		const repeat = 3
		for i := 0; i < repeat; i++ {
			y := float64(f.backgroundYView + i*backgroundHeight)
			// Handle wrapping (if background scrolls completely offscreen)
			if y > float64(backgroundHeight) {
				y -= float64(backgroundHeight * i)
			}
			drawOps := &ebiten.DrawImageOptions{}
			drawOps.GeoM.Scale(scaleX, scaleY)
			drawOps.GeoM.Translate(0, y)
			screen.DrawImage(f.background, drawOps)
		}
		imageX := f.playerFrame * f.frameWidth
		imageY := 0
		frame := f.player.SubImage(image.Rect(imageX, imageY, imageX+f.frameWidth, imageY+f.frameHeight)).(*ebiten.Image)
		playerOps := ebiten.DrawImageOptions{}
		playerOps.GeoM.Translate(float64(f.xloc), float64(f.yloc))
		screen.DrawImage(frame, &playerOps)

		for _, ball := range f.balls {
			ops := &ebiten.DrawImageOptions{}
			ops.GeoM.Translate(ball.x, ball.y)
			screen.DrawImage(f.ball, ops)
		}

		for _, bark := range f.barks {
			ops := &ebiten.DrawImageOptions{}
			ops.GeoM.Translate(bark.x, bark.y)
			screen.DrawImage(f.barkImage, ops)
		}
	}
}

func (f firstGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func LoadFont(fontFile string, size float64) font.Face {
	fileHandle, err := os.Open(fontFile)
	if err != nil {
		log.Fatal(err)
	}
	fontData, err := io.ReadAll(fileHandle)
	if err != nil {
		log.Fatal(err)
	}
	ttFont, err := opentype.Parse(fontData)
	if err != nil {
		log.Fatal(err)
	}
	fontFace, err := opentype.NewFace(ttFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	return fontFace
}
