package winevent

import (
	"fmt"
	"log"
	"syscall"

	"golang.org/x/sys/windows"
)

const eventTpl = "Global\\%s_%s_%d" // Global\Ns_Evt_Id

const drainEvent = "__drain"

type EventHandler func()

type WinEvent struct {
	// Namespace of the event
	Namespace string

	// id is used as part of the event registration to avoid conflict
	// with other instances running on the same machine. This is set to
	// pid by default.
	id int

	events   []windows.Handle
	handlers []EventHandler

	stop bool
}

func NewWinEvent(namespace string) *WinEvent {
	return &WinEvent{
		Namespace: namespace,
		id:        syscall.Getpid(),
	}
}

// SetId overrides the default id (pid)
func (w *WinEvent) SetId(id int) {
	w.id = id
}

func (w *WinEvent) Register(eventName string, handler EventHandler) error {
	evtPtr, err := GetEventPtr(w.Namespace, eventName, w.id)
	if err != nil {
		return err
	}

	evtHandle, err := windows.CreateEvent(nil, 0, 0, evtPtr)
	if err != nil {
		return err
	}

	w.events = append(w.events, evtHandle)
	w.handlers = append(w.handlers, handler)
	return nil
}

// Start waiting for events. Additional registration will NOT taking effect after Go is called.
func (w *WinEvent) Start() error {
	if err := w.registerDrain(); err != nil {
		return err
	}

	w.stop = false

	evtCh := make(chan int)
	go func() {
		for {
			evtId, err := windows.WaitForMultipleObjects(w.events, false, windows.INFINITE)
			if err != nil {
				log.Println("wait for event errs:", err)
			}
			if w.stop {
				close(evtCh)
				return
			} else {
				evtCh <- int(evtId)
			}
		}
	}()

	var evtId int
	for {
		evtId = <-evtCh
		if w.stop {
			return nil
		}
		w.handlers[evtId]()
	}
}

func (w *WinEvent) Stop() {
	w.stop = true
	_ = SetEvent(w.Namespace, drainEvent, syscall.Getpid())
}

func (w *WinEvent) registerDrain() error {
	return w.Register(drainEvent, func() {})
}

func GetEventStr(namespace string, event string, id int) string {
	return fmt.Sprintf(eventTpl, namespace, event, id)
}

func GetEventPtr(namespace string, event string, id int) (*uint16, error) {
	return windows.UTF16PtrFromString(GetEventStr(namespace, event, id))
}

// SetEvent signals the event. If error is windows.ERROR_FILE_NOT_FOUND,
// it means the event is not created (no process is registered for that specific event).
func SetEvent(namespace string, event string, id int) error {
	evtPtr, err := GetEventPtr(namespace, event, id)
	if err != nil {
		return err
	}

	evtHandle, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, evtPtr)
	if err != nil {
		return err
	}

	return windows.SetEvent(evtHandle)
}
