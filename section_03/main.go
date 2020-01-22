package main

import (
	"C"
	"image"
	_ "image/png"
	"log"
	"runtime"
	"time"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/golang-ui/nuklear/nk"
	"github.com/xlab/closer"
)
import (
	"fmt"
	"os"
	"path"
)

const (
	winWidth  = 800
	winHeight = 600

	maxVertexBuffer  = 512 * 1024
	maxElementBuffer = 128 * 1024
)

var home nk.Image

func init() {
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		closer.Fatalln(err)
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	win, err := glfw.CreateWindow(winWidth, winHeight, "Nuklear Demo", nil, nil)
	if err != nil {
		closer.Fatalln(err)
	}
	win.MakeContextCurrent()

	width, height := win.GetSize()
	log.Printf("glfw: created window %dx%d", width, height)

	if err := gl.Init(); err != nil {
		closer.Fatalln("opengl: init failed:", err)
	}
	gl.Viewport(0, 0, int32(width), int32(height))

	ctx := nk.NkPlatformInit(win, nk.PlatformInstallCallbacks)

	atlas := nk.NewFontAtlas()
	nk.NkFontStashBegin(&atlas)

	config := nk.NkFontConfig(17)
	// config.SetOversample(2, 2)
	config.SetRange(nk.NkFontChineseGlyphRanges())
	//sansFont := nk.NkFontAtlasAddFromFile(atlas, "assets/SourceHanSansK-Normal.ttf", 14, &config)
	sansFont := nk.NkFontAtlasAddFromFile(atlas, "../assets/DroidSansFallback.ttf", 17, &config)
	// simsunFont := nk.NkFontAtlasAddFromFile(atlas, "/Library/Fonts/Microsoft/SimHei.ttf", 14, &config)
	nk.NkFontStashEnd()
	if sansFont != nil {
		nk.NkStyleSetFont(ctx, sansFont.Handle())
	}

	exitC := make(chan struct{}, 1)
	doneC := make(chan struct{}, 1)
	closer.Bind(func() {
		close(exitC)
		<-doneC
	})

	state := &State{
		bgColor: nk.NkRgba(28, 48, 62, 255),
	}
	// nk.NkTexteditInitDefault(&state.text)
	initImages()

	fpsTicker := time.NewTicker(time.Second / 30)
	for {
		select {
		case <-exitC:
			nk.NkPlatformShutdown()
			glfw.Terminate()
			fpsTicker.Stop()
			close(doneC)
			return
		case <-fpsTicker.C:
			if win.ShouldClose() {
				close(exitC)
				continue
			}
			glfw.PollEvents()
			gfxMain(win, ctx, state)
		}
	}
}
func initImages() {
	rgbaImg := loadImage("../assets/001-home.png")
	if rgbaImg == nil {
		log.Printf("load image fail !")
		return
	}
	var i uint32 = 0
	home = NkImageFromRgba(&i, rgbaImg)
}

func loadImage(filePath string) *image.NRGBA {
	_, filename, _, _ := runtime.Caller(1)
	datapath := path.Join(path.Dir(filename), filePath)
	fmt.Println(datapath)
	file, err := os.Open(datapath)
	if err != nil {
		log.Printf("read %s error", filePath, err)
		return nil
	}
	img, _, decodeErr := image.Decode(file)
	if decodeErr != nil {
		log.Printf("decode image error %s", filePath, decodeErr)
		return nil
	}

	switch img.(type) {
	case *image.RGBA:
		// i in an *image.RGBA
		log.Printf("RGBA")
	case *image.NRGBA:
		// i in an *image.NRBGA
		log.Printf("NRGBA")
	}
	if p, ok := img.(*image.NRGBA); ok {
		return p
	}
	return nil
}

// NkImageFromRgba converts RGBA image to NkImage (texture)
// Call with tex=0 for first time use, then keep tex for later use.
func NkImageFromRgba(tex *uint32, rgba *image.NRGBA) nk.Image {
	gl.Enable(gl.TEXTURE_2D)
	if *tex == 0 {
		gl.GenTextures(1, tex)
	}
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, *tex)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_NEAREST)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR_MIPMAP_NEAREST)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,                         // Level of detail, 0 is base image level
		gl.RGBA8,                  // Format. COuld ge RGB8 or RGB16UI
		int32(rgba.Bounds().Dx()), // Width
		int32(rgba.Bounds().Dy()), // Height
		0,                         // Must be 0
		gl.RGBA,                   // Pixel data format of last parameter rgba,Pix, could be RGB
		gl.UNSIGNED_BYTE,          // Data type for of last parameter rgba,Pix, could be UNSIGNED_SHORT
		gl.Ptr(rgba.Pix))          // Pixel data
	gl.GenerateMipmap(gl.TEXTURE_2D)
	return nk.NkImageId(int32(*tex))
}

func gfxMain(win *glfw.Window, ctx *nk.Context, state *State) {
	nk.NkPlatformNewFrame()

	width, height := win.GetSize()

	// Menu
	menuSlot := float32(64)
	menuSize := int32(5)
	menuWidth := (menuSlot+5)*float32(menuSize) + 5
	menuBounds := nk.NkRect(float32(float32(width)/2-menuWidth/2), 5, menuWidth, menuSlot+10)
	menuUpdate := nk.NkBegin(ctx, "menu", menuBounds, nk.WindowNoScrollbar|nk.WindowBackground)
	if menuUpdate > 0 {
		nk.NkLayoutRowBegin(ctx, nk.Static, menuSlot, menuSize)
		{
			nk.NkLayoutRowPush(ctx, menuSlot)

			if nk.NkButtonImage(ctx, home) > 0 {
				log.Println("[INFO] button pressed!")
			}
			nk.NkLayoutRowPush(ctx, menuSlot)
			if nk.NkButtonLabel(ctx, "测试button") > 0 {
				log.Println("[INFO] button pressed!")
			}
			nk.NkLayoutRowPush(ctx, menuSlot)
			if nk.NkButtonLabel(ctx, "测试button") > 0 {
				log.Println("[INFO] button pressed!")
			}
			nk.NkLayoutRowPush(ctx, menuSlot)
			if nk.NkButtonLabel(ctx, "测试button") > 0 {
				log.Println("[INFO] button pressed!")
			}
			nk.NkLayoutRowPush(ctx, menuSlot)
			if nk.NkButtonLabel(ctx, "测试button") > 0 {
				log.Println("[INFO] button pressed!")
			}
		}

		nk.NkLayoutRowEnd(ctx)
	}
	nk.NkEnd(ctx)

	// Layout
	bounds := nk.NkRect(float32(width-230-5), float32(height-250-5), 230, 250)
	update := nk.NkBegin(ctx, "Demo", bounds,
		nk.WindowBorder)

	if update > 0 {
		nk.NkLayoutRowStatic(ctx, 30, 80, 1)
		{
			if nk.NkButtonLabel(ctx, "测试button") > 0 {
				log.Println("[INFO] button pressed!")
			}
		}
		nk.NkLayoutRowDynamic(ctx, 30, 2)
		{
			if nk.NkOptionLabel(ctx, "easy", flag(state.opt == Easy)) > 0 {
				state.opt = Easy
			}
			if nk.NkOptionLabel(ctx, "hard", flag(state.opt == Hard)) > 0 {
				state.opt = Hard
			}
		}
		nk.NkLayoutRowDynamic(ctx, 30, 1)
		{
			nk.NkEditBuffer(ctx, nk.EditField, &state.text, nk.NkFilterDefault)
			if nk.NkButtonLabel(ctx, "Print Entered Text") > 0 {
				log.Println(state.text.GetGoString())
			}
		}
		nk.NkLayoutRowDynamic(ctx, 25, 1)
		{
			nk.NkPropertyInt(ctx, "Compression:", 0, &state.prop, 100, 10, 1)
		}
		nk.NkLayoutRowDynamic(ctx, 20, 1)
		{
			nk.NkLabel(ctx, "background:", nk.TextLeft)
		}
		nk.NkLayoutRowDynamic(ctx, 25, 1)
		{
			size := nk.NkVec2(nk.NkWidgetWidth(ctx), 400)
			if nk.NkComboBeginColor(ctx, state.bgColor, size) > 0 {
				nk.NkLayoutRowDynamic(ctx, 120, 1)
				cf := nk.NkColorCf(state.bgColor)
				cf = nk.NkColorPicker(ctx, cf, nk.ColorFormatRGBA)
				state.bgColor = nk.NkRgbCf(cf)
				nk.NkLayoutRowDynamic(ctx, 25, 1)
				r, g, b, a := state.bgColor.RGBAi()
				r = nk.NkPropertyi(ctx, "#R:", 0, r, 255, 1, 1)
				g = nk.NkPropertyi(ctx, "#G:", 0, g, 255, 1, 1)
				b = nk.NkPropertyi(ctx, "#B:", 0, b, 255, 1, 1)
				a = nk.NkPropertyi(ctx, "#A:", 0, a, 255, 1, 1)
				state.bgColor.SetRGBAi(r, g, b, a)
				nk.NkComboEnd(ctx)
			}
		}
	}
	nk.NkEnd(ctx)

	// Render
	bg := make([]float32, 4)
	nk.NkColorFv(bg, state.bgColor)
	gl.Viewport(0, 0, int32(width), int32(height))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.ClearColor(bg[0], bg[1], bg[2], bg[3])
	nk.NkPlatformRender(nk.AntiAliasingOn, maxVertexBuffer, maxElementBuffer)
	win.SwapBuffers()
}

type Option uint8

const (
	Easy Option = 0
	Hard Option = 1
)

type State struct {
	bgColor nk.Color
	prop    int32
	opt     Option
	text    nk.TextEdit
}

func onError(code int32, msg string) {
	log.Printf("[glfw ERR]: error %d: %s", code, msg)
}
