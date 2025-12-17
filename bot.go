package napcat_ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

var ErrNotConnected = errors.New("napcat: not connected")

type Bot struct {
	url    string
	dialer *websocket.Dialer
	header http.Header

	retryDelay     time.Duration
	writeTimeout   time.Duration
	incomingBuffer int
	outgoingBuffer int

	dispatcher *Dispatcher

	connMu sync.RWMutex
	conn   *websocket.Conn
	sendQ  chan sendReq

	callMu  sync.Mutex
	calls   map[string]chan callResult
	echoSeq uint64
}

type sendReq struct {
	ctx     context.Context
	payload []byte
}

type Option func(*Bot)

func WithDialer(dialer *websocket.Dialer) Option {
	return func(b *Bot) {
		if dialer != nil {
			b.dialer = dialer
		}
	}
}

func WithHeader(header http.Header) Option {
	return func(b *Bot) {
		if header == nil {
			b.header = nil
			return
		}
		b.header = header.Clone()
	}
}

func WithRetryDelay(delay time.Duration) Option {
	return func(b *Bot) {
		b.retryDelay = delay
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(b *Bot) {
		b.writeTimeout = timeout
	}
}

func WithIncomingBuffer(size int) Option {
	return func(b *Bot) {
		b.incomingBuffer = size
	}
}

func WithOutgoingBuffer(size int) Option {
	return func(b *Bot) {
		b.outgoingBuffer = size
	}
}

func WithDispatcher(dispatcher *Dispatcher) Option {
	return func(b *Bot) {
		if dispatcher != nil {
			b.dispatcher = dispatcher
		}
	}
}

func New(url string, opts ...Option) *Bot {
	b := &Bot{
		url:            url,
		dialer:         websocket.DefaultDialer,
		retryDelay:     5 * time.Second,
		writeTimeout:   10 * time.Second,
		incomingBuffer: 256,
		outgoingBuffer: 256,
		dispatcher:     NewDispatcher(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	if b.dialer == nil {
		b.dialer = websocket.DefaultDialer
	}
	if b.retryDelay <= 0 {
		b.retryDelay = 5 * time.Second
	}
	if b.incomingBuffer <= 0 {
		b.incomingBuffer = 256
	}
	if b.outgoingBuffer <= 0 {
		b.outgoingBuffer = 256
	}
	if b.dispatcher == nil {
		b.dispatcher = NewDispatcher()
	}
	b.sendQ = make(chan sendReq, b.outgoingBuffer)
	return b
}

func (b *Bot) Dispatcher() *Dispatcher {
	return b.dispatcher
}

func (b *Bot) Use(mw ...MessageMiddleware) {
	b.dispatcher.Use(mw...)
}

func (b *Bot) UsePrivate(mw ...PrivateMiddleware) {
	b.dispatcher.UsePrivate(mw...)
}

func (b *Bot) UseGroup(mw ...GroupMiddleware) {
	b.dispatcher.UseGroup(mw...)
}

func (b *Bot) OnPrivate(h PrivateHandler) {
	b.dispatcher.OnPrivate(h)
}

func (b *Bot) OnPrivateIf(filter func(*PrivateMessageEvent) bool, h PrivateHandler) {
	b.dispatcher.OnPrivateIf(filter, h)
}

func (b *Bot) OnPrivateText(h PrivateHandler)  { b.dispatcher.OnPrivateText(h) }
func (b *Bot) OnPrivateFile(h PrivateHandler)  { b.dispatcher.OnPrivateFile(h) }
func (b *Bot) OnPrivateImage(h PrivateHandler) { b.dispatcher.OnPrivateImage(h) }
func (b *Bot) OnPrivateAudio(h PrivateHandler) { b.dispatcher.OnPrivateAudio(h) }
func (b *Bot) OnPrivateVideo(h PrivateHandler) { b.dispatcher.OnPrivateVideo(h) }

func (b *Bot) OnGroup(h GroupHandler) {
	b.dispatcher.OnGroup(h)
}

func (b *Bot) OnGroupIf(filter func(*GroupMessageEvent) bool, h GroupHandler) {
	b.dispatcher.OnGroupIf(filter, h)
}

func (b *Bot) OnGroupText(h GroupHandler)  { b.dispatcher.OnGroupText(h) }
func (b *Bot) OnGroupFile(h GroupHandler)  { b.dispatcher.OnGroupFile(h) }
func (b *Bot) OnGroupImage(h GroupHandler) { b.dispatcher.OnGroupImage(h) }
func (b *Bot) OnGroupAudio(h GroupHandler) { b.dispatcher.OnGroupAudio(h) }
func (b *Bot) OnGroupVideo(h GroupHandler) { b.dispatcher.OnGroupVideo(h) }
func (b *Bot) OnGroupAtMe(h GroupHandler)  { b.dispatcher.OnGroupAtMe(h) }

func (b *Bot) OnConnect(h ConnectHandler) {
	b.dispatcher.OnConnect(h)
}

func (b *Bot) OnDisconnect(h DisconnectHandler) {
	b.dispatcher.OnDisconnect(h)
}

func (b *Bot) OnError(h ErrorHandler) {
	b.dispatcher.OnError(h)
}

func (b *Bot) OnPanic(h PanicHandler) {
	b.dispatcher.OnPanic(h)
}

func (b *Bot) Connected() bool {
	return b.getConn() != nil
}

func (b *Bot) Send(ctx context.Context, payload []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if b.getConn() == nil {
		return ErrNotConnected
	}

	req := sendReq{ctx: ctx, payload: payload}
	select {
	case b.sendQ <- req:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type callResult struct {
	payload []byte
	err     error
}

func (b *Bot) Call(ctx context.Context, action string, params any, out any) error {
	if action == "" {
		return errors.New("napcat: empty action")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	echo := fmt.Sprintf("napcat-%d", atomic.AddUint64(&b.echoSeq, 1))
	resCh := make(chan callResult, 1)
	b.registerCall(echo, resCh)
	defer b.unregisterCall(echo)

	payload, err := json.Marshal(WSRequest[any]{
		Action: action,
		Params: params,
		Echo:   echo,
	})
	if err != nil {
		return err
	}

	if err := b.Send(ctx, payload); err != nil {
		return err
	}

	select {
	case res := <-resCh:
		if res.err != nil {
			return res.err
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(res.payload, out)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Bot) SendPrivateMsg(ctx context.Context, userID int64, message string) (int64, error) {
	if userID <= 0 {
		return 0, errors.New("napcat: invalid user_id")
	}

	var resp WSResponse[SendMsgData]
	if err := b.Call(ctx, ActionSendPrivateMsg, SendPrivateTextParams{UserID: userID, Message: message}, &resp); err != nil {
		return 0, err
	}
	if resp.Status != StatusOK || resp.Retcode != 0 {
		msg := resp.Wording
		if msg == "" {
			msg = resp.Message
		}
		if msg != "" {
			return 0, fmt.Errorf("napcat: send_private_msg failed: status=%s retcode=%d msg=%s", resp.Status, resp.Retcode, msg)
		}
		return 0, fmt.Errorf("napcat: send_private_msg failed: status=%s retcode=%d", resp.Status, resp.Retcode)
	}
	return resp.Data.MessageID, nil
}

func (b *Bot) GetPrivateFileURL(ctx context.Context, fileID string) (string, error) {
	var resp WSResponse[GetPrivateFileURLData]
	if err := b.Call(ctx, "get_private_file_url", GetPrivateFileURLParams{FileID: fileID}, &resp); err != nil {
		return "", err
	}
	if resp.Status != StatusOK || resp.Retcode != 0 {
		msg := resp.Wording
		if msg == "" {
			msg = resp.Message
		}
		if msg != "" {
			return "", fmt.Errorf("napcat: get_private_file_url failed: status=%s retcode=%d msg=%s", resp.Status, resp.Retcode, msg)
		}
		return "", fmt.Errorf("napcat: get_private_file_url failed: status=%s retcode=%d", resp.Status, resp.Retcode)
	}
	return resp.Data.URL, nil
}

func (b *Bot) GetGroupFileURL(ctx context.Context, groupID int64, fileID string, busID ...int64) (string, error) {
	if groupID <= 0 {
		return "", errors.New("napcat: invalid group_id")
	}
	if fileID == "" {
		return "", errors.New("napcat: empty file_id")
	}

	params := GetGroupFileURLParams[int64]{GroupID: groupID, FileID: fileID}
	if len(busID) > 0 {
		params.BusID = busID[0]
	}

	var resp WSResponse[GetGroupFileURLData]
	if err := b.Call(ctx, "get_group_file_url", params, &resp); err != nil {
		return "", err
	}
	if resp.Status != StatusOK || resp.Retcode != 0 {
		msg := resp.Wording
		if msg == "" {
			msg = resp.Message
		}
		if msg != "" {
			return "", fmt.Errorf("napcat: get_group_file_url failed: status=%s retcode=%d msg=%s", resp.Status, resp.Retcode, msg)
		}
		return "", fmt.Errorf("napcat: get_group_file_url failed: status=%s retcode=%d", resp.Status, resp.Retcode)
	}
	return resp.Data.URL, nil
}

func (b *Bot) GetGroupMemberList(ctx context.Context, groupID int64) ([]GroupMemberInfo, error) {
	if groupID <= 0 {
		return nil, errors.New("napcat: invalid group_id")
	}

	var resp WSResponse[[]GroupMemberInfo]
	if err := b.Call(ctx, "get_group_member_list", GetGroupMemberListParams{GroupID: groupID}, &resp); err != nil {
		return nil, err
	}
	if resp.Status != StatusOK || resp.Retcode != 0 {
		msg := resp.Wording
		if msg == "" {
			msg = resp.Message
		}
		if msg != "" {
			return nil, fmt.Errorf("napcat: get_group_member_list failed: status=%s retcode=%d msg=%s", resp.Status, resp.Retcode, msg)
		}
		return nil, fmt.Errorf("napcat: get_group_member_list failed: status=%s retcode=%d", resp.Status, resp.Retcode)
	}
	return resp.Data, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	urlStr := strings.TrimSpace(b.url)
	if err := validateWSURL(urlStr); err != nil {
		return fmt.Errorf("napcat: invalid websocket url %q: %w", b.url, err)
	}

	runCtx, cancel := context.WithCancel(ctx)

	var sendWG sync.WaitGroup
	sendWG.Add(1)
	go func() {
		defer sendWG.Done()
		b.sendLoop(runCtx)
	}()
	defer sendWG.Wait()
	defer cancel()

	incoming := make(chan []byte, b.incomingBuffer)
	var dispatchWG sync.WaitGroup
	dispatchWG.Add(1)
	go func() {
		defer dispatchWG.Done()
		for msg := range incoming {
			if err := b.dispatcher.HandleBytes(msg); err != nil {
				b.dispatcher.fireError(err)
			}
		}
	}()
	defer func() {
		close(incoming)
		dispatchWG.Wait()
	}()

	for {
		if err := runCtx.Err(); err != nil {
			return err
		}

		conn, _, err := b.dialer.DialContext(runCtx, urlStr, b.header)
		if err != nil {
			b.dispatcher.fireError(err)
			if !sleepContext(runCtx, b.retryDelay) {
				return runCtx.Err()
			}
			continue
		}

		b.setConn(conn)
		b.dispatcher.fireConnect(conn)

		closed := make(chan struct{})
		go func() {
			select {
			case <-runCtx.Done():
				_ = conn.Close()
			case <-closed:
			}
		}()

		err = b.readLoop(runCtx, conn, incoming)
		close(closed)
		_ = conn.Close()

		b.failAllCalls(err)
		b.setConn(nil)
		b.dispatcher.fireDisconnect(conn, err)

		if err := runCtx.Err(); err != nil {
			return err
		}

		if err != nil {
			b.dispatcher.fireError(err)
		}

		if !sleepContext(runCtx, b.retryDelay) {
			return runCtx.Err()
		}
	}
}

func (b *Bot) readLoop(ctx context.Context, conn *websocket.Conn, incoming chan<- []byte) error {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if IsHeartbeat(message) {
			continue
		}

		if b.maybeDeliverResponse(message) {
			continue
		}

		select {
		case incoming <- message:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (b *Bot) sendLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-b.sendQ:
			if !ok {
				return
			}
			reqCtx := req.ctx
			if reqCtx == nil {
				reqCtx = context.Background()
			}
			if err := reqCtx.Err(); err != nil {
				continue
			}

			conn := b.getConn()
			if conn == nil {
				continue
			}

			deadline := time.Time{}
			if b.writeTimeout > 0 {
				deadline = time.Now().Add(b.writeTimeout)
			}
			if ctxDeadline, ok := reqCtx.Deadline(); ok && (deadline.IsZero() || ctxDeadline.Before(deadline)) {
				deadline = ctxDeadline
			}
			_ = conn.SetWriteDeadline(deadline)
			if err := conn.WriteMessage(websocket.TextMessage, req.payload); err != nil {
				_ = conn.Close()
			}
		}
	}
}

func IsHeartbeat(message []byte) bool {
	t := gjson.GetBytes(message, "meta_event_type").String()
	return t == "heartbeat" || t == "lifecycle"
}

func (b *Bot) setConn(conn *websocket.Conn) {
	b.connMu.Lock()
	b.conn = conn
	b.connMu.Unlock()
}

func (b *Bot) getConn() *websocket.Conn {
	b.connMu.RLock()
	conn := b.conn
	b.connMu.RUnlock()
	return conn
}

func (b *Bot) registerCall(echo string, ch chan callResult) {
	b.callMu.Lock()
	if b.calls == nil {
		b.calls = make(map[string]chan callResult)
	}
	b.calls[echo] = ch
	b.callMu.Unlock()
}

func (b *Bot) unregisterCall(echo string) {
	b.callMu.Lock()
	if b.calls != nil {
		delete(b.calls, echo)
	}
	b.callMu.Unlock()
}

func (b *Bot) maybeDeliverResponse(message []byte) bool {
	status := gjson.GetBytes(message, "status")
	if !status.Exists() {
		return false
	}
	if !gjson.GetBytes(message, "retcode").Exists() {
		return false
	}
	echo := gjson.GetBytes(message, "echo")
	if !echo.Exists() {
		return false
	}
	echoStr := echo.String()
	if echoStr == "" {
		return false
	}
	return b.deliverResponse(echoStr, message)
}

func (b *Bot) deliverResponse(echo string, payload []byte) bool {
	b.callMu.Lock()
	ch := b.calls[echo]
	if ch != nil {
		delete(b.calls, echo)
	}
	b.callMu.Unlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- callResult{payload: payload}:
	default:
	}
	close(ch)
	return true
}

func (b *Bot) failAllCalls(err error) {
	b.callMu.Lock()
	calls := b.calls
	b.calls = nil
	b.callMu.Unlock()
	for _, ch := range calls {
		select {
		case ch <- callResult{err: err}:
		default:
		}
		close(ch)
	}
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func validateWSURL(raw string) error {
	if raw == "" {
		return errors.New("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case "ws", "wss":
	default:
		if u.Scheme == "" {
			return errors.New("missing scheme (expected ws:// or wss://)")
		}
		return fmt.Errorf("unsupported scheme %q (expected ws or wss)", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("missing host")
	}
	return nil
}
