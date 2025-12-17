package napcat_ws

import (
	"github.com/gorilla/websocket"
)

// SendMsg 发送 Websocket 消息
func SendMsg(conn *websocket.Conn, message []byte) error {
	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return err
	}
	return nil
}
