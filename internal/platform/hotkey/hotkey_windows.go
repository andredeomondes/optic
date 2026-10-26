//go:build windows

package hotkey

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
	procPostThreadMsgW   = user32.NewProc("PostThreadMessageW")
	procGetCurrentThread = kernel32.NewProc("GetCurrentThreadId")
)

const (
	wmUser     = 0x0400
	wmHotkey   = 0x0312
	wmApp      = 0x8000
	wmRun      = wmApp + 1 // wakes the loop to run a queued call
	pmNoRemove = 0x0000
)

type msg struct {
	hwnd     uintptr
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	ptX, ptY int32
	private  uint32
}

// Supported reports whether global hotkeys work on this OS.
func Supported() bool { return true }

// win32 registers hotkeys for the calling thread (hwnd NULL).
type win32 struct{}

func (win32) register(id int, mods, vk uint32) bool {
	r, _, _ := procRegisterHotKey.Call(0, uintptr(id), uintptr(mods), uintptr(vk))
	return r != 0
}

func (win32) unregister(id int) {
	procUnregisterHotKey.Call(0, uintptr(id))
}

// thread is the locked OS thread that owns the hotkeys and their message queue.
type thread struct {
	tid    uintptr
	calls  chan func() // at most one pending: Manager calls are serialized
	quit   bool
	exited chan struct{}
}

// Start registers hotkeys on a dedicated OS thread; onFire runs in its own goroutine.
func Start(onFire func(id string)) Manager {
	t := &thread{calls: make(chan func(), 1), exited: make(chan struct{})}
	m := newManager(win32{}, onFire, t.run, t.stop)
	ready := make(chan struct{})
	go t.loop(ready, m.fire)
	<-ready
	return m
}

func (t *thread) loop(ready chan struct{}, fire func(int)) {
	runtime.LockOSThread() // never unlocked: the thread ends with this goroutine
	defer close(t.exited)
	t.tid, _, _ = procGetCurrentThread.Call()
	var m msg
	// Create the message queue before anyone posts to it.
	procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, wmUser, wmUser, pmNoRemove)
	close(ready)
	for !t.quit {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 { // WM_QUIT or error
			return
		}
		switch m.message {
		case wmHotkey:
			fire(int(m.wParam))
		case wmRun:
			select {
			case f := <-t.calls:
				f()
			default:
			}
		}
	}
}

// run executes f on the hotkey thread and waits for it.
func (t *thread) run(f func()) bool {
	done := make(chan struct{})
	select {
	case <-t.exited:
		return false
	case t.calls <- func() { f(); close(done) }:
	}
	r, _, _ := procPostThreadMsgW.Call(t.tid, wmRun, 0, 0)
	if r == 0 {
		select {
		case <-t.calls: // take it back
			return false
		default: // the loop already took it
		}
	}
	select {
	case <-done:
		return true
	case <-t.exited:
		return false
	}
}

// stop ends the loop; the locked thread exits with it.
func (t *thread) stop() {
	if t.run(func() { t.quit = true }) {
		<-t.exited
	}
}
