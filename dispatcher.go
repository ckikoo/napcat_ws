package napcat_ws

import (
	"runtime/debug"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

type PrivateHandler func(*PrivateMessageEvent)
type GroupHandler func(*GroupMessageEvent)
type ConnectHandler func(*websocket.Conn)
type DisconnectHandler func(*websocket.Conn, error)
type ErrorHandler func(error)
type PanicHandler func(PanicInfo)

type MessageMiddleware func(*MessageEvent, func())
type PrivateMiddleware func(*PrivateMessageEvent, func())
type GroupMiddleware func(*GroupMessageEvent, func())

type PanicInfo struct {
	Recovered any
	Stack     []byte
}

type Dispatcher struct {
	mu sync.RWMutex

	privateHandlers    []PrivateHandler
	groupHandlers      []GroupHandler
	messageMiddleware  []MessageMiddleware
	privateMiddleware  []PrivateMiddleware
	groupMiddleware    []GroupMiddleware
	connectHandlers    []ConnectHandler
	disconnectHandlers []DisconnectHandler
	errorHandlers      []ErrorHandler
	panicHandlers      []PanicHandler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

func (d *Dispatcher) OnPrivate(h PrivateHandler) {
	if h == nil {
		return
	}
	d.mu.Lock()
	d.privateHandlers = append(d.privateHandlers, h)
	d.mu.Unlock()
}

func (d *Dispatcher) Use(mw ...MessageMiddleware) {
	var cleaned []MessageMiddleware
	for _, m := range mw {
		if m != nil {
			cleaned = append(cleaned, m)
		}
	}
	if len(cleaned) == 0 {
		return
	}
	d.mu.Lock()
	d.messageMiddleware = append(d.messageMiddleware, cleaned...)
	d.mu.Unlock()
}

func (d *Dispatcher) UsePrivate(mw ...PrivateMiddleware) {
	var cleaned []PrivateMiddleware
	for _, m := range mw {
		if m != nil {
			cleaned = append(cleaned, m)
		}
	}
	if len(cleaned) == 0 {
		return
	}
	d.mu.Lock()
	d.privateMiddleware = append(d.privateMiddleware, cleaned...)
	d.mu.Unlock()
}

func (d *Dispatcher) UseGroup(mw ...GroupMiddleware) {
	var cleaned []GroupMiddleware
	for _, m := range mw {
		if m != nil {
			cleaned = append(cleaned, m)
		}
	}
	if len(cleaned) == 0 {
		return
	}
	d.mu.Lock()
	d.groupMiddleware = append(d.groupMiddleware, cleaned...)
	d.mu.Unlock()
}

func (d *Dispatcher) OnPrivateIf(filter func(*PrivateMessageEvent) bool, h PrivateHandler) {
	if h == nil {
		return
	}
	if filter == nil {
		d.OnPrivate(h)
		return
	}
	d.OnPrivate(func(e *PrivateMessageEvent) {
		if filter(e) {
			h(e)
		}
	})
}

func (d *Dispatcher) OnPrivateText(h PrivateHandler) {
	d.OnPrivateIf(func(e *PrivateMessageEvent) bool { return e.HasText() }, h)
}
func (d *Dispatcher) OnPrivateFile(h PrivateHandler) {
	d.OnPrivateIf(func(e *PrivateMessageEvent) bool { return e.HasFile() }, h)
}
func (d *Dispatcher) OnPrivateImage(h PrivateHandler) {
	d.OnPrivateIf(func(e *PrivateMessageEvent) bool { return e.HasImage() }, h)
}
func (d *Dispatcher) OnPrivateAudio(h PrivateHandler) {
	d.OnPrivateIf(func(e *PrivateMessageEvent) bool { return e.HasAudio() }, h)
}
func (d *Dispatcher) OnPrivateVideo(h PrivateHandler) {
	d.OnPrivateIf(func(e *PrivateMessageEvent) bool { return e.HasVideo() }, h)
}

func (d *Dispatcher) OnGroup(h GroupHandler) {
	if h == nil {
		return
	}
	d.mu.Lock()
	d.groupHandlers = append(d.groupHandlers, h)
	d.mu.Unlock()
}

func (d *Dispatcher) OnGroupIf(filter func(*GroupMessageEvent) bool, h GroupHandler) {
	if h == nil {
		return
	}
	if filter == nil {
		d.OnGroup(h)
		return
	}
	d.OnGroup(func(e *GroupMessageEvent) {
		if filter(e) {
			h(e)
		}
	})
}

func (d *Dispatcher) OnGroupText(h GroupHandler) {
	d.OnGroupIf(func(e *GroupMessageEvent) bool { return e.HasText() }, h)
}
func (d *Dispatcher) OnGroupFile(h GroupHandler) {
	d.OnGroupIf(func(e *GroupMessageEvent) bool { return e.HasFile() }, h)
}
func (d *Dispatcher) OnGroupImage(h GroupHandler) {
	d.OnGroupIf(func(e *GroupMessageEvent) bool { return e.HasImage() }, h)
}
func (d *Dispatcher) OnGroupAudio(h GroupHandler) {
	d.OnGroupIf(func(e *GroupMessageEvent) bool { return e.HasAudio() }, h)
}
func (d *Dispatcher) OnGroupVideo(h GroupHandler) {
	d.OnGroupIf(func(e *GroupMessageEvent) bool { return e.HasVideo() }, h)
}
func (d *Dispatcher) OnGroupAtMe(h GroupHandler) {
	d.OnGroupIf(func(e *GroupMessageEvent) bool { return e.IsAtMe() }, h)
}

func (d *Dispatcher) OnConnect(h ConnectHandler) {
	if h == nil {
		return
	}
	d.mu.Lock()
	d.connectHandlers = append(d.connectHandlers, h)
	d.mu.Unlock()
}

func (d *Dispatcher) OnDisconnect(h DisconnectHandler) {
	if h == nil {
		return
	}
	d.mu.Lock()
	d.disconnectHandlers = append(d.disconnectHandlers, h)
	d.mu.Unlock()
}

func (d *Dispatcher) OnError(h ErrorHandler) {
	if h == nil {
		return
	}
	d.mu.Lock()
	d.errorHandlers = append(d.errorHandlers, h)
	d.mu.Unlock()
}

func (d *Dispatcher) OnPanic(h PanicHandler) {
	if h == nil {
		return
	}
	d.mu.Lock()
	d.panicHandlers = append(d.panicHandlers, h)
	d.mu.Unlock()
}

func (d *Dispatcher) HandleBytes(data []byte) error {
	raw := gjson.ParseBytes(data)
	metaType := raw.Get("meta_event_type").String()
	if metaType == "heartbeat" || metaType == "lifecycle" {
		return nil
	}

	if raw.Get("post_type").String() != "message" {
		return nil
	}

	switch raw.Get("message_type").String() {
	case "private":
		d.dispatchPrivate(newPrivateMessageEvent(raw))
	case "group":
		d.dispatchGroup(newGroupMessageEvent(raw))
	}
	return nil
}

func (d *Dispatcher) dispatchPrivate(evt *PrivateMessageEvent) {
	handlers := d.snapshotPrivate()
	privateMW := d.snapshotPrivateMiddleware()
	messageMW := d.snapshotMessageMiddleware()

	runHandlers := func() {
		for _, h := range handlers {
			d.safeCall(func() { h(evt) })
		}
	}

	runPrivate := func() {
		d.runPrivateMiddleware(evt, privateMW, runHandlers)
	}

	d.runMessageMiddleware(&evt.MessageEvent, messageMW, runPrivate)
}

func (d *Dispatcher) dispatchGroup(evt *GroupMessageEvent) {
	handlers := d.snapshotGroup()
	groupMW := d.snapshotGroupMiddleware()
	messageMW := d.snapshotMessageMiddleware()

	runHandlers := func() {
		for _, h := range handlers {
			d.safeCall(func() { h(evt) })
		}
	}

	runGroup := func() {
		d.runGroupMiddleware(evt, groupMW, runHandlers)
	}

	d.runMessageMiddleware(&evt.MessageEvent, messageMW, runGroup)
}

func (d *Dispatcher) snapshotPrivate() []PrivateHandler {
	d.mu.RLock()
	handlers := append([]PrivateHandler(nil), d.privateHandlers...)
	d.mu.RUnlock()
	return handlers
}

func (d *Dispatcher) snapshotGroup() []GroupHandler {
	d.mu.RLock()
	handlers := append([]GroupHandler(nil), d.groupHandlers...)
	d.mu.RUnlock()
	return handlers
}

func (d *Dispatcher) snapshotMessageMiddleware() []MessageMiddleware {
	d.mu.RLock()
	handlers := append([]MessageMiddleware(nil), d.messageMiddleware...)
	d.mu.RUnlock()
	return handlers
}

func (d *Dispatcher) snapshotPrivateMiddleware() []PrivateMiddleware {
	d.mu.RLock()
	handlers := append([]PrivateMiddleware(nil), d.privateMiddleware...)
	d.mu.RUnlock()
	return handlers
}

func (d *Dispatcher) snapshotGroupMiddleware() []GroupMiddleware {
	d.mu.RLock()
	handlers := append([]GroupMiddleware(nil), d.groupMiddleware...)
	d.mu.RUnlock()
	return handlers
}

func (d *Dispatcher) safeCall(fn func()) {
	_ = d.safeCallWithPanic(fn)
}

func (d *Dispatcher) safeCallWithPanic(fn func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			d.firePanic(PanicInfo{
				Recovered: r,
				Stack:     debug.Stack(),
			})
		}
	}()
	fn()
	return false
}

func (d *Dispatcher) runMessageMiddleware(evt *MessageEvent, mws []MessageMiddleware, final func()) {
	d.runMiddleware(len(mws), func(i int, next func()) {
		mws[i](evt, next)
	}, func() bool {
		return d.safeCallWithPanic(func() { final() })
	})
}

func (d *Dispatcher) runPrivateMiddleware(evt *PrivateMessageEvent, mws []PrivateMiddleware, final func()) {
	d.runMiddleware(len(mws), func(i int, next func()) {
		mws[i](evt, next)
	}, func() bool {
		return d.safeCallWithPanic(func() { final() })
	})
}

func (d *Dispatcher) runGroupMiddleware(evt *GroupMessageEvent, mws []GroupMiddleware, final func()) {
	d.runMiddleware(len(mws), func(i int, next func()) {
		mws[i](evt, next)
	}, func() bool {
		return d.safeCallWithPanic(func() { final() })
	})
}

func (d *Dispatcher) runMiddleware(n int, call func(i int, next func()), final func() bool) {
	if n <= 0 {
		_ = final()
		return
	}

	var run func(i int)
	run = func(i int) {
		if i >= n {
			_ = final()
			return
		}

		called := false
		next := func() {
			if called {
				return
			}
			called = true
			run(i + 1)
		}

		panicked := d.safeCallWithPanic(func() { call(i, next) })
		if panicked && !called {
			next()
		}
	}

	run(0)
}

func (d *Dispatcher) fireConnect(conn *websocket.Conn) {
	d.mu.RLock()
	handlers := append([]ConnectHandler(nil), d.connectHandlers...)
	d.mu.RUnlock()
	for _, h := range handlers {
		d.safeCall(func() { h(conn) })
	}
}

func (d *Dispatcher) fireDisconnect(conn *websocket.Conn, err error) {
	d.mu.RLock()
	handlers := append([]DisconnectHandler(nil), d.disconnectHandlers...)
	d.mu.RUnlock()
	for _, h := range handlers {
		d.safeCall(func() { h(conn, err) })
	}
}

func (d *Dispatcher) fireError(err error) {
	d.mu.RLock()
	handlers := append([]ErrorHandler(nil), d.errorHandlers...)
	d.mu.RUnlock()
	for _, h := range handlers {
		d.safeCall(func() { h(err) })
	}
}

func (d *Dispatcher) firePanic(info PanicInfo) {
	d.mu.RLock()
	handlers := append([]PanicHandler(nil), d.panicHandlers...)
	d.mu.RUnlock()
	for _, h := range handlers {
		func() {
			defer func() { _ = recover() }()
			h(info)
		}()
	}
}
