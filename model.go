package napcat_ws

import "encoding/json"

const (
	ActionSendGroupMsg   = "send_group_msg"
	ActionSendPrivateMsg = "send_private_msg"

	TypeText  = "text"
	TypeAt    = "at"
	TypeImage = "image"
	TypeAudio = "record"
	TypeFile  = "file"
	TypeVideo = "video"
	TypeFace  = "face"
	TypeReply = "reply"
	TypeMusic = "music"
)

type WSRequest[T any] struct {
	Action string `json:"action"`
	Params T      `json:"params"`
	Echo   string `json:"echo,omitempty"`
}

type ID interface {
	~string | ~int | ~int64
}

type WSStatus string

const (
	StatusOK     WSStatus = "ok"
	StatusFailed WSStatus = "failed"
)

type WSResponse[T any] struct {
	Status  WSStatus `json:"status"`
	Retcode int64    `json:"retcode"`
	Data    T        `json:"data"`
	Message string   `json:"message,omitempty"`
	Wording string   `json:"wording,omitempty"`
	Echo    string   `json:"echo,omitempty"`
}

type DeleteMsgParams struct {
	MessageID int64 `json:"message_id"`
}

type GroupBanParams struct {
	GroupID  int64 `json:"group_id"`
	UserID   int64 `json:"user_id"`
	Duration int64 `json:"duration"`
}

type SendGroupTextParams struct {
	GroupID int64  `json:"group_id"`
	Message string `json:"message"`
}

type SendPrivateTextParams struct {
	UserID     int64  `json:"user_id"`
	Message    string `json:"message"`
	AutoEscape bool   `json:"auto_escape,omitempty"`
}

type MessageSegment struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type SendGroupSegmentsParams struct {
	GroupID int64            `json:"group_id"`
	Message []MessageSegment `json:"message"`
}

type SendPrivateSegmentsParams struct {
	UserID     int64            `json:"user_id"`
	Message    []MessageSegment `json:"message"`
	AutoEscape bool             `json:"auto_escape,omitempty"`
}

type SendMsgData struct {
	MessageID int64 `json:"message_id"`
}

type AtData struct {
	QQ int64 `json:"qq"`
}

type TextData struct {
	Text string `json:"text"`
}

type AudioVideoData struct {
	File string `json:"file"`
}

type ImageData struct {
	File    string `json:"file"`
	Summary string `json:"summary,omitempty"`
}

type FileData struct {
	File string `json:"file"`
	Name string `json:"name,omitempty"`
}

type FaceData struct {
	ID int64 `json:"id"`
}

type ReplyData struct {
	ID int64 `json:"id"`
}

type MusicData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type LikeMsgParams struct {
	UserID int64 `json:"user_id"`
	Times  int   `json:"times"`
}

type GetGroupMemberListParams struct {
	GroupID int64 `json:"group_id"`
}

type GroupMemberInfo struct {
	GroupID         int64  `json:"group_id"`
	UserID          int64  `json:"user_id"`
	Nickname        string `json:"nickname"`
	Card            string `json:"card"`
	Sex             string `json:"sex"`
	Age             int    `json:"age"`
	Area            string `json:"area"`
	JoinTime        int64  `json:"join_time"`
	LastSentTime    int64  `json:"last_sent_time"`
	Level           string `json:"level"`
	Role            string `json:"role"`
	Unfriendly      bool   `json:"unfriendly"`
	Title           string `json:"title"`
	TitleExpireTime int64  `json:"title_expire_time"`
	CardChangeable  bool   `json:"card_changeable"`
	ShutUpTimestamp int64  `json:"shut_up_timestamp"`
}

type GetGroupFileURLParams[T ID] struct {
	GroupID T      `json:"group_id"`
	FileID  string `json:"file_id"`
	BusID   int64  `json:"busid,omitempty"`
}

type GetGroupFileURLData struct {
	URL string `json:"url"`
}

type GetPrivateFileURLParams struct {
	FileID string `json:"file_id"`
}

type GetPrivateFileURLData struct {
	URL string `json:"url"`
}
