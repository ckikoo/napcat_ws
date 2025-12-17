package napcat

import (
	"encoding/json"
	"errors"
)

// MarshalPrivateTextMsg 编码私聊文本消息
func MarshalPrivateTextMsg(userID int64, text string) ([]byte, error) {
	if userID <= 0 {
		return nil, errors.New("invalid userID")
	}

	req := WSRequest[SendPrivateTextParams]{
		Action: ActionSendPrivateMsg,
		Params: SendPrivateTextParams{
			UserID:  userID,
			Message: text,
		},
	}
	return json.Marshal(req)
}
