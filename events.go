package napcat

import (
	"strings"

	"github.com/tidwall/gjson"
)

type Sender struct {
	UserID   int64
	Nickname string
	Card     string
	Role     string
}

// BaseEvent is the minimal common fields for OneBot v11 events.
type BaseEvent struct {
	Time     int64
	SelfID   int64
	PostType string
	Raw      gjson.Result
}

type MessageEvent struct {
	BaseEvent
	MessageType   string
	SubType       string
	UserID        int64
	MessageID     int64
	MessageSeq    int64
	RealID        int64
	RealSeq       int64
	ReadID        string
	RawMessage    string
	Font          int64
	MessageFormat string
	Sender        Sender
	Segments      []gjson.Result
}

type PrivateMessageEvent struct {
	MessageEvent
	TargetID int64
}

type GroupMessageEvent struct {
	MessageEvent
	GroupID   int64
	GroupName string
}

func (e *MessageEvent) GetText() string {
	var parts []string
	for _, seg := range e.Segments {
		if seg.Get("type").String() == "text" {
			parts = append(parts, seg.Get("data.text").String())
		}
	}
	return strings.Join(parts, "")
}

// HasSegmentType reports whether the message contains a segment of the given type
// (e.g. "text", "file", "image").
func (e *MessageEvent) HasSegmentType(typ string) bool {
	for _, seg := range e.Segments {
		if seg.Get("type").String() == typ {
			return true
		}
	}
	return false
}

func (e *MessageEvent) HasText() bool  { return e.HasSegmentType(TypeText) }
func (e *MessageEvent) HasFile() bool  { return e.HasSegmentType(TypeFile) }
func (e *MessageEvent) HasImage() bool { return e.HasSegmentType(TypeImage) }
func (e *MessageEvent) HasAudio() bool { return e.HasSegmentType(TypeAudio) }
func (e *MessageEvent) HasVideo() bool { return e.HasSegmentType(TypeVideo) }

func (e *MessageEvent) FirstSegment(typ string) (gjson.Result, bool) {
	for _, seg := range e.Segments {
		if seg.Get("type").String() == typ {
			return seg, true
		}
	}
	return gjson.Result{}, false
}

func (e *MessageEvent) SegmentsByType(typ string) []gjson.Result {
	var out []gjson.Result
	for _, seg := range e.Segments {
		if seg.Get("type").String() == typ {
			out = append(out, seg)
		}
	}
	return out
}

func (e *MessageEvent) GetFile() (name, fileID string, size int64, ok bool) {
	for _, seg := range e.Segments {
		if seg.Get("type").String() == "file" {
			return seg.Get("data.file").String(),
				seg.Get("data.file_id").String(),
				seg.Get("data.file_size").Int(),
				true
		}
	}
	return
}

func (e *GroupMessageEvent) IsAtMe() bool {
	for _, seg := range e.Segments {
		if seg.Get("type").String() == "at" {
			qq := seg.Get("data.qq")
			if qq.Type == gjson.String && qq.String() == "all" {
				return true
			}
			if qq.Int() == e.SelfID {
				return true
			}
		}
	}
	return false
}

func newBaseEvent(raw gjson.Result) BaseEvent {
	return BaseEvent{
		Time:     raw.Get("time").Int(),
		SelfID:   raw.Get("self_id").Int(),
		PostType: raw.Get("post_type").String(),
		Raw:      raw,
	}
}

func newMessageEvent(raw gjson.Result) MessageEvent {
	return MessageEvent{
		BaseEvent:     newBaseEvent(raw),
		MessageType:   raw.Get("message_type").String(),
		SubType:       raw.Get("sub_type").String(),
		UserID:        raw.Get("user_id").Int(),
		MessageID:     raw.Get("message_id").Int(),
		MessageSeq:    raw.Get("message_seq").Int(),
		RealID:        raw.Get("real_id").Int(),
		RealSeq:       raw.Get("real_seq").Int(),
		ReadID:        raw.Get("read_id").String(),
		RawMessage:    raw.Get("raw_message").String(),
		Font:          raw.Get("font").Int(),
		MessageFormat: raw.Get("message_format").String(),
		Sender: Sender{
			UserID:   raw.Get("sender.user_id").Int(),
			Nickname: raw.Get("sender.nickname").String(),
			Card:     raw.Get("sender.card").String(),
			Role:     raw.Get("sender.role").String(),
		},
		Segments: raw.Get("message").Array(),
	}
}

func newPrivateMessageEvent(raw gjson.Result) *PrivateMessageEvent {
	return &PrivateMessageEvent{
		MessageEvent: newMessageEvent(raw),
		TargetID:     raw.Get("target_id").Int(),
	}
}

func newGroupMessageEvent(raw gjson.Result) *GroupMessageEvent {
	return &GroupMessageEvent{
		MessageEvent: newMessageEvent(raw),
		GroupID:      raw.Get("group_id").Int(),
		GroupName:    raw.Get("group_name").String(),
	}
}
