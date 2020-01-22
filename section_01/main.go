package main

import (
	"C"
	"log"
	"runtime"
	"time"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/golang-ui/nuklear/nk"
	"github.com/xlab/closer"
)

const (
	winWidth  = 400
	winHeight = 500

	maxVertexBuffer  = 512 * 1024
	maxElementBuffer = 128 * 1024
)

func init() {
	log.Printf("application init")
	//确保当前线程绑定在操作系统线程上，这是一些图形库的要求
	runtime.LockOSThread()
}

func main() {
	//进行GLFW初始化，如果初始化失败，使用closer在程序退出前执行清理和日志打印
	if err := glfw.Init(); err != nil {
		closer.Fatalln(err)
	}
	//设置GLFW的版本
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	//使用OpenGL核心模式
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	//向前兼容，3.0以后支持，Mac OS上使用3.2版本以上必须设置
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	//使用GLFW创建一个窗口，如果创建失败退出程序
	win, err := glfw.CreateWindow(winWidth, winHeight, "Nuklear Demo", nil, nil)
	if err != nil {
		closer.Fatalln(err)
	}
	//设置当前OpenGL上下文为当前窗口线程，每个线程只能有一个上下文
	win.MakeContextCurrent()

	width, height := win.GetSize()
	log.Printf("glfw: created window %dx%d", width, height)

	//初始化OpenGL
	if err := gl.Init(); err != nil {
		closer.Fatalln("opengl: init failed:", err)
	}
	//视口设置
	gl.Viewport(0, 0, int32(width), int32(height))

	//初始化Nuklear，返回值ctx是Nuklear的上下文
	ctx := nk.NkPlatformInit(win, nk.PlatformInstallCallbacks)

	//字体设置
	atlas := nk.NewFontAtlas()
	nk.NkFontStashBegin(&atlas)
	nk.NkFontStashEnd()

	//退出事件
	exitC := make(chan struct{}, 1)
	doneC := make(chan struct{}, 1)
	closer.Bind(func() {
		close(exitC)
		<-doneC
	})

	//背景颜色
	state := &State{
		bgColor: nk.NkRgba(28, 48, 62, 255),
	}
	//初始化输入框
	nk.NkTexteditInitDefault(&state.text)

	//fps
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
			//glfw的事件响应
			glfw.PollEvents()
			//nuklear布局
			gfxMain(win, ctx, state)
		}
	}
}

func gfxMain(win *glfw.Window, ctx *nk.Context, state *State) {
	nk.NkPlatformNewFrame()

	log.Printf("current refresh")
	// Layout

	bounds := nk.NkRect(30, 50, 230, 300)
	log.Printf("bounds get x=%f", bounds.X())
	update := nk.NkBegin(ctx, "Demo", bounds,
		nk.WindowBorder|nk.WindowMovable|nk.WindowScalable|nk.WindowMinimizable|nk.WindowTitle|nk.WindowClosable)

	if update > 0 {
		nk.NkLayoutRowStatic(ctx, 30, 80, 1)
		{
			if nk.NkButtonLabel(ctx, "button") > 0 {
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
	width, height := win.GetSize()
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
