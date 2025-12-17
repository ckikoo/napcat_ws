package napcat_ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func wsURL(httpURL string) string {
	if strings.HasPrefix(httpURL, "https://") {
		return "wss://" + strings.TrimPrefix(httpURL, "https://")
	}
	if strings.HasPrefix(httpURL, "http://") {
		return "ws://" + strings.TrimPrefix(httpURL, "http://")
	}
	return httpURL
}

func TestBotCall_GetGroupFileURL(t *testing.T) {
	t.Parallel()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req struct {
				Action string          `json:"action"`
				Params json.RawMessage `json:"params"`
				Echo   string          `json:"echo"`
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				return
			}
			var (
				resp []byte
				ok   bool
			)
			switch req.Action {
			case "get_group_file_url":
				resp, _ = json.Marshal(WSResponse[GetGroupFileURLData]{
					Status:  StatusOK,
					Retcode: 0,
					Data: GetGroupFileURLData{
						URL: "http://example.com/file",
					},
					Message: "ok",
					Wording: "ok",
					Echo:    req.Echo,
				})
				ok = true
			case "get_group_member_list":
				resp, _ = json.Marshal(WSResponse[[]GroupMemberInfo]{
					Status:  StatusOK,
					Retcode: 0,
					Data: []GroupMemberInfo{
						{
							GroupID:  123,
							UserID:   10001,
							Nickname: "tester",
							Card:     "card",
							Role:     "member",
						},
					},
					Message: "ok",
					Wording: "ok",
					Echo:    req.Echo,
				})
				ok = true
			case "get_private_file_url":
				resp, _ = json.Marshal(WSResponse[GetPrivateFileURLData]{
					Status:  StatusOK,
					Retcode: 0,
					Data: GetPrivateFileURLData{
						URL: "http://example.com/private",
					},
					Message: "ok",
					Wording: "ok",
					Echo:    req.Echo,
				})
				ok = true
			case ActionSendPrivateMsg:
				resp, _ = json.Marshal(WSResponse[SendMsgData]{
					Status:  StatusOK,
					Retcode: 0,
					Data: SendMsgData{
						MessageID: 42,
					},
					Message: "ok",
					Wording: "ok",
					Echo:    req.Echo,
				})
				ok = true
			}
			if !ok {
				continue
			}
			_ = conn.WriteMessage(websocket.TextMessage, resp)
		}
	}))
	defer srv.Close()

	connected := make(chan struct{})
	var once sync.Once

	bot := New(wsURL(srv.URL), WithRetryDelay(10*time.Millisecond))
	bot.OnConnect(func(*websocket.Conn) { once.Do(func() { close(connected) }) })

	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	done := make(chan error, 1)
	go func() { done <- bot.Run(runCtx) }()

	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for connect")
	}

	callCtx, cancelCall := context.WithTimeout(context.Background(), time.Second)
	defer cancelCall()

	var got WSResponse[GetGroupFileURLData]
	if err := bot.Call(callCtx, "get_group_file_url", GetGroupFileURLParams[string]{GroupID: "123", FileID: "abc"}, &got); err != nil {
		t.Fatalf("Call() error: %v", err)
	}
	if got.Status != StatusOK || got.Retcode != 0 {
		t.Fatalf("unexpected status=%q retcode=%d", got.Status, got.Retcode)
	}
	if got.Data.URL != "http://example.com/file" {
		t.Fatalf("unexpected url: %q", got.Data.URL)
	}
	if got.Echo == "" {
		t.Fatal("expected echo")
	}

	url, err := bot.GetGroupFileURL(callCtx, 123, "abc")
	if err != nil {
		t.Fatalf("GetGroupFileURL() error: %v", err)
	}
	if url != "http://example.com/file" {
		t.Fatalf("unexpected url: %q", url)
	}

	members, err := bot.GetGroupMemberList(callCtx, 123)
	if err != nil {
		t.Fatalf("GetGroupMemberList() error: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("unexpected members len: %d", len(members))
	}
	if members[0].UserID != 10001 {
		t.Fatalf("unexpected member user_id: %d", members[0].UserID)
	}

	url, err = bot.GetPrivateFileURL(callCtx, "abc")
	if err != nil {
		t.Fatalf("GetPrivateFileURL() error: %v", err)
	}
	if url != "http://example.com/private" {
		t.Fatalf("unexpected url: %q", url)
	}

	msgID, err := bot.SendPrivateMsg(callCtx, 10001, "hi")
	if err != nil {
		t.Fatalf("SendPrivateMsg() error: %v", err)
	}
	if msgID != 42 {
		t.Fatalf("unexpected message_id: %d", msgID)
	}

	cancelRun()
	select {
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Run() to exit")
	case <-done:
	}
}
