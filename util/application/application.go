package application

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/sansebasko/engine/audio/al"
	"github.com/sansebasko/engine/audio/vorbis"
	"github.com/sansebasko/engine/camera"
	"github.com/sansebasko/engine/camera/control"
	"github.com/sansebasko/engine/core"
	"github.com/sansebasko/engine/gls"
	"github.com/sansebasko/engine/gui"
	"github.com/sansebasko/engine/math32"
	"github.com/sansebasko/engine/renderer"
	"github.com/sansebasko/engine/util/logger"
	"github.com/sansebasko/engine/window"
)

// Application is a standard application object which can be used as a base for G3N applications.
// It creates a Window, OpenGL state, default cameras, default scene and Gui and has a method to run the render loop.
type Application struct {
	core.Dispatcher                         // Embedded event dispatcher
	core.TimerManager                       // Embedded timer manager
	wmgr              window.IWindowManager // Window manager
	win               window.IWindow        // Application window
	gl                *gls.GLS              // OpenGL state
	log               *logger.Logger        // Default application logger
	renderer          *renderer.Renderer    // Renderer object
	camPersp          *camera.Perspective   // Perspective camera
	camOrtho          *camera.Orthographic  // Orthographic camera
	camera            camera.ICamera        // Current camera
	orbit             *control.OrbitControl // Camera orbit controller
	frameRater        *FrameRater           // Render loop frame rater
	keyState          *KeyState             // State of keys
	mouseState        *MouseState           // State of pressed mouse buttons
	audioDev          *al.Device            // Default audio device
	scene             *core.Node            // Node container for 3D tests
	guiroot           *gui.Root             // Gui root panel
	frameCount        uint64                // Frame counter
	frameTime         time.Time             // Time at the start of the frame
	frameDelta        time.Duration         // Time delta from previous frame
	startTime         time.Time             // Time at the start of the render loop
	fullScreen        *bool                 // Full screen option
	swapInterval      *int                  // Swap interval option
	targetFPS         *uint                 // Target FPS option
	noglErrors        *bool                 // No OpenGL check errors options
	cpuProfile        *string               // File to write cpu profile to
	execTrace         *string               // File to write execution trace data to
}

// Options defines initial options passed to the application creation function
type Options struct {
	Title       string // Initial window title
	Height      int    // Initial window height (default is screen width)
	Width       int    // Initial window width (default is screen height)
	Fullscreen  bool   // Window full screen flag (default = false)
	LogPrefix   string // Log prefix (default = "")
	LogLevel    int    // Initial log level (default = DEBUG)
	EnableFlags bool   // Enable command line flags (default = false)
	TargetFPS   uint   // Desired frames per second rate (default = 60)
}

// appInstance contains the pointer to the single Application instance
var appInstance *Application

// Create creates and returns the application object using the specified options.
// This function must be called only once.
func Create(ops Options) (*Application, error) {

	if appInstance != nil {
		return nil, fmt.Errorf("Application already created")
	}
	app := new(Application)
	appInstance = app
	app.Dispatcher.Initialize()
	app.TimerManager.Initialize()

	// Initialize options defaults
	app.fullScreen = new(bool)
	app.swapInterval = new(int)
	app.targetFPS = new(uint)
	app.noglErrors = new(bool)
	app.cpuProfile = new(string)
	app.execTrace = new(string)
	*app.swapInterval = -1
	*app.targetFPS = 60

	// Options parameter overrides some options
	if ops.TargetFPS != 0 {
		*app.fullScreen = ops.Fullscreen
		*app.targetFPS = ops.TargetFPS
	}

	// Creates flags if requested (override options defaults)
	if ops.EnableFlags {
		app.fullScreen = flag.Bool("fullscreen", *app.fullScreen, "Starts application with full screen")
		app.swapInterval = flag.Int("swapinterval", *app.swapInterval, "Sets the swap buffers interval to this value")
		app.targetFPS = flag.Uint("targetfps", *app.targetFPS, "Sets the frame rate in frames per second")
		app.noglErrors = flag.Bool("noglerrors", *app.noglErrors, "Do not check OpenGL errors at each call (may increase FPS)")
		app.cpuProfile = flag.String("cpuprofile", *app.cpuProfile, "Activate cpu profiling writing profile to the specified file")
		app.execTrace = flag.String("exectrace", *app.execTrace, "Activate execution tracer writing data to the specified file")
	}
	flag.Parse()

	// Creates application logger
	app.log = logger.New(ops.LogPrefix, nil)
	app.log.AddWriter(logger.NewConsole(false))
	app.log.SetFormat(logger.FTIME | logger.FMICROS)
	app.log.SetLevel(ops.LogLevel)

	// Window event handling must run on the main OS thread
	runtime.LockOSThread()

	// Get the window manager
	wmgr, err := window.Manager("glfw")
	if err != nil {
		return nil, err
	}
	app.wmgr = wmgr

	// Get the screen resolution
	swidth, sheight := app.wmgr.ScreenResolution(nil)
	var posx, posy int
	// If not full screen, sets the window size
	if !*app.fullScreen {
		if ops.Width != 0 {
			posx = (swidth - ops.Width) / 2
			if posx < 0 {
				posx = 0
			}
			swidth = ops.Width
		}
		if ops.Height != 0 {
			posy = (sheight - ops.Height) / 2
			if posy < 0 {
				posy = 0
			}
			sheight = ops.Height
		}
	}

	// Creates window
	win, err := app.wmgr.CreateWindow(swidth, sheight, ops.Title, *app.fullScreen)
	if err != nil {
		return nil, err
	}
	win.SetPos(posx, posy)
	app.win = win

	// Create OpenGL state
	gl, err := gls.New()
	if err != nil {
		return nil, err
	}
	app.gl = gl
	// Checks OpenGL errors
	app.gl.SetCheckErrors(!*app.noglErrors)

	// Logs OpenGL version
	glVersion := app.Gl().GetString(gls.VERSION)
	app.log.Info("OpenGL version: %s", glVersion)

	// Clears the screen
	cc := math32.NewColor("gray")
	app.gl.ClearColor(cc.R, cc.G, cc.B, 1)
	app.gl.Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)

	// Creates KeyState
	app.keyState = NewKeyState(win)

	// Creates MouseState
	app.mouseState = NewMouseState(win)

	// Creates perspective camera
	width, height := app.win.Size()
	aspect := float32(width) / float32(height)
	app.camPersp = camera.NewPerspective(65, aspect, 0.01, 1000)

	// Creates orthographic camera
	app.camOrtho = camera.NewOrthographic(-2, 2, 2, -2, 0.01, 100)
	app.camOrtho.SetPosition(0, 0, 3)
	app.camOrtho.LookAt(&math32.Vector3{0, 0, 0})
	app.camOrtho.SetZoom(1.0)

	// Default camera is perspective
	app.camera = app.camPersp

	// Creates orbit camera control
	// It is important to do this after the root panel subscription
	// to avoid GUI events being propagated to the orbit control.
	app.orbit = control.NewOrbitControl(app.camera, app.win)

	// Creates scene for 3D objects
	app.scene = core.NewNode()

	// Creates gui root panel
	app.guiroot = gui.NewRoot(app.gl, app.win)
	app.guiroot.SetColor(math32.NewColor("silver"))

	// Creates renderer
	app.renderer = renderer.NewRenderer(gl)
	err = app.renderer.AddDefaultShaders()
	if err != nil {
		return nil, fmt.Errorf("Error from AddDefaulShaders:%v", err)
	}
	app.renderer.SetScene(app.scene)
	app.renderer.SetGui(app.guiroot)

	// Create frame rater
	app.frameRater = NewFrameRater(*app.targetFPS)

	// Sets the default window resize event handler
	app.win.SubscribeID(window.OnWindowSize, app, func(evname string, ev interface{}) {
		app.OnWindowResize()
	})
	app.OnWindowResize()

	return app, nil
}

// Get returns the application single instance or nil
// if the application was not created yet
func Get() *Application {

	return appInstance
}

// Log returns the application logger
func (app *Application) Log() *logger.Logger {

	return app.log
}

// Window returns the application window
func (app *Application) Window() window.IWindow {

	return app.win
}

// KeyState returns the application KeyState
func (app *Application) KeyState() *KeyState {

	return app.keyState
}

// MouseState returns the application MouseState
func (app *Application) MouseState() *MouseState {

	return app.mouseState
}

// Gl returns the OpenGL state
func (app *Application) Gl() *gls.GLS {

	return app.gl
}

// Gui returns the current application Gui root panel
func (app *Application) Gui() *gui.Root {

	return app.guiroot
}

// Scene returns the current application 3D scene
func (app *Application) Scene() *core.Node {

	return app.scene
}

// SetScene sets the 3D scene to be rendered
func (app *Application) SetScene(scene *core.Node) {

	app.renderer.SetScene(scene)
}

// SetGui sets the root panel of the gui to be rendered
func (app *Application) SetGui(root *gui.Root) {

	app.guiroot = root
	app.renderer.SetGui(app.guiroot)
}

// SetPanel3D sets the gui panel inside which the 3D scene is shown.
func (app *Application) SetPanel3D(panel3D gui.IPanel) {

	app.renderer.SetGuiPanel3D(panel3D)
}

// Panel3D returns the current gui panel where the 3D scene is shown.
func (app *Application) Panel3D() gui.IPanel {

	return app.renderer.Panel3D()
}

// CameraPersp returns the application perspective camera
func (app *Application) CameraPersp() *camera.Perspective {

	return app.camPersp
}

// CameraOrtho returns the application orthographic camera
func (app *Application) CameraOrtho() *camera.Orthographic {

	return app.camOrtho
}

// Camera returns the current application camera
func (app *Application) Camera() camera.ICamera {

	return app.camera
}

// SetCamera sets the current application camera
func (app *Application) SetCamera(cam camera.ICamera) {

	app.camera = cam
}

// Orbit returns the current camera orbit control
func (app *Application) Orbit() *control.OrbitControl {

	return app.orbit
}

// SetOrbit sets the camera orbit control
func (app *Application) SetOrbit(oc *control.OrbitControl) {

	app.orbit = oc
}

// FrameRater returns the FrameRater object
func (app *Application) FrameRater() *FrameRater {

	return app.frameRater
}

// FrameCount returns the total number of frames since the call to Run()
func (app *Application) FrameCount() uint64 {

	return app.frameCount
}

// FrameDelta returns the duration of the previous frame
func (app *Application) FrameDelta() time.Duration {

	return app.frameDelta
}

// FrameDeltaSeconds returns the duration of the previous frame
// in float32 seconds
func (app *Application) FrameDeltaSeconds() float32 {

	return float32(app.frameDelta.Seconds())
}

// RunTime returns the duration since the call to Run()
func (app *Application) RunTime() time.Duration {

	return time.Now().Sub(app.startTime)
}

// RunSeconds returns the elapsed time in seconds since the call to Run()
func (app *Application) RunSeconds() float32 {

	return float32(time.Now().Sub(app.startTime).Seconds())
}

// Renderer returns the application renderer
func (app *Application) Renderer() *renderer.Renderer {

	return app.renderer
}

// SetCPUProfile must be called before Run() and sets the file name for cpu profiling.
// If set the cpu profiling starts before running the render loop and continues
// till the end of the application.
func (app *Application) SetCPUProfile(fname string) {

	*app.cpuProfile = fname
}

// SetOnWindowResize replaces the default window resize handler with the specified one
func (app *Application) SetOnWindowResize(f func(evname string, ev interface{})) {

	app.win.UnsubscribeID(window.OnWindowSize, app)
	app.win.SubscribeID(window.OnWindowSize, app, f)
}

// Run runs the application render loop
func (app *Application) Run() error {

	// Set swap interval
	if *app.swapInterval >= 0 {
		app.wmgr.SetSwapInterval(*app.swapInterval)
		app.log.Debug("Swap interval set to: %v", *app.swapInterval)
	}

	// Start profiling if requested
	if *app.cpuProfile != "" {
		f, err := os.Create(*app.cpuProfile)
		if err != nil {
			return err
		}
		defer f.Close()
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
		app.log.Info("Started writing CPU profile to: %s", *app.cpuProfile)
	}

	// Start execution trace if requested
	if *app.execTrace != "" {
		f, err := os.Create(*app.execTrace)
		if err != nil {
			return err
		}
		defer f.Close()
		err = trace.Start(f)
		if err != nil {
			return err
		}
		defer trace.Stop()
		app.log.Info("Started writing execution trace to: %s", *app.execTrace)
	}

	app.startTime = time.Now()
	app.frameTime = time.Now()

	// Render loop
	for true {
		// If was requested to terminate the application by trying to close the window
		// or by calling Quit(), dispatch OnQuit event for subscribers.
		// If no subscriber cancelled the event, terminates the application.
		if app.win.ShouldClose() {
			canceled := app.Dispatch(gui.OnQuit, nil)
			if !canceled {
				canceled = dispatchRecursive(gui.OnQuit, nil, app.scene.Children())
			}
			if !canceled {
				canceled = dispatchRecursive(gui.OnQuit, nil, app.guiroot.Children())
			}
			if canceled {
				app.win.SetShouldClose(false)
			} else {
				break
			}
		}

		// Starts measuring this frame
		app.frameRater.Start()

		// Updates frame start and time delta in context
		now := time.Now()
		app.frameDelta = now.Sub(app.frameTime)
		app.frameTime = now

		// Process root panel timers
		if app.Gui() != nil {
			app.Gui().TimerManager.ProcessTimers()
		}

		// Process application timers
		app.ProcessTimers()

		// Dispatch before render event
		app.Dispatch(gui.OnBeforeRender, nil)
		dispatchRecursive(gui.OnBeforeRender, nil, app.scene.Children())
		dispatchRecursive(gui.OnBeforeRender, nil, app.guiroot.Children())

		// Renders the current scene and/or gui
		rendered, err := app.renderer.Render(app.camera)
		if err != nil {
			return err
		}

		// Poll input events and process them
		app.wmgr.PollEvents()

		if rendered {
			app.win.SwapBuffers()
		}

		// Dispatch after render event
		app.Dispatch(gui.OnAfterRender, nil)
		dispatchRecursive(gui.OnAfterRender, nil, app.scene.Children())
		dispatchRecursive(gui.OnAfterRender, nil, app.guiroot.Children())

		// Controls the frame rate
		app.frameRater.Wait()
		app.frameCount++
	}

	// Dispose resources
	if app.scene != nil {
		app.scene.DisposeChildren(true)
	}
	if app.guiroot != nil {
		app.guiroot.DisposeChildren(true)
	}

	// Close default audio device
	if app.audioDev != nil {
		al.CloseDevice(app.audioDev)
	}

	// Terminates window manager
	app.wmgr.Terminate()

	// This is important when using the execution tracer
	runtime.UnlockOSThread()
	return nil
}

func dispatchRecursive(evname string, ev interface{}, nodes []core.INode) bool {
	for _, node := range nodes {
		if node != nil {
			if node.Dispatch(evname, ev) || dispatchRecursive(evname, ev, node.Children()) {
				return true
			}
		}
	}
	return false
}

// OpenDefaultAudioDevice opens the default audio device setting it to the current context
func (app *Application) OpenDefaultAudioDevice() error {

	// Opens default audio device
	var err error
	app.audioDev, err = al.OpenDevice("")
	if err != nil {
		return fmt.Errorf("Error: %s opening OpenAL default device", err)
	}

	// Checks for OpenAL effects extension support
	audioEFX := false
	if al.IsExtensionPresent("ALC_EXT_EFX") {
		audioEFX = true
	}

	// Creates audio context with auxiliary sends
	var attribs []int
	if audioEFX {
		attribs = []int{al.MAX_AUXILIARY_SENDS, 4}
	}
	acx, err := al.CreateContext(app.audioDev, attribs)
	if err != nil {
		return fmt.Errorf("Error creating OpenAL context:%s", err)
	}

	// Makes the context the current one
	err = al.MakeContextCurrent(acx)
	if err != nil {
		return fmt.Errorf("Error setting OpenAL context current:%s", err)
	}

	// Logs audio library versions
	app.log.Info("%s version: %s", al.GetString(al.Vendor), al.GetString(al.Version))
	app.log.Info("%s", vorbis.VersionString())
	return nil
}

// Quit requests to terminate the application
// Application will dispatch OnQuit events to registered subscriber which
// can cancel the process by calling CancelDispatch().
func (app *Application) Quit() {

	app.win.SetShouldClose(true)
}

// OnWindowResize is default handler for window resize events.
func (app *Application) OnWindowResize() {

	// Get framebuffer size and sets the viewport accordingly
	width, height := app.win.FramebufferSize()
	app.gl.Viewport(0, 0, int32(width), int32(height))

	// Sets perspective camera aspect ratio
	aspect := float32(width) / float32(height)
	app.Camera().SetAspect(aspect)

	// Sets the GUI root panel size to the size of the framebuffer
	if app.guiroot != nil {
		app.guiroot.SetSize(float32(width), float32(height))
	}
}
