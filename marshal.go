package napcat_ws

import (
	"encoding/json"
	"errors"
)

func marshalSegment[T any](typ string, data T) (MessageSegment, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return MessageSegment{}, err
	}
	return MessageSegment{Type: typ, Data: b}, nil
}

// MarshalDeleteMessage 编码撤回消息
func MarshalDeleteMessage(messageID int64) ([]byte, error) {
	req := WSRequest[DeleteMsgParams]{
		Action: "delete_msg",
		Params: DeleteMsgParams{MessageID: messageID},
	}
	return json.Marshal(req)
}

// MarshalGroupBan 编码群禁言消息
func MarshalGroupBan(groupID, userID, duration int64) ([]byte, error) {
	req := WSRequest[GroupBanParams]{
		Action: "set_group_ban",
		Params: GroupBanParams{
			GroupID:  groupID,
			UserID:   userID,
			Duration: duration,
		},
	}
	return json.Marshal(req)
}

// MarshalGroupTextMsg 编码群文本消息
func MarshalGroupTextMsg(groupID int64, text string) ([]byte, error) {
	if groupID <= 0 {
		return nil, errors.New("invalid groupID")
	}

	req := WSRequest[SendGroupTextParams]{
		Action: ActionSendGroupMsg,
		Params: SendGroupTextParams{
			GroupID: groupID,
			Message: text,
		},
	}
	return json.Marshal(req)
}

// MarshalAtMsg 编码群 @ 消息（@qq + 文本）
func MarshalAtMsg(groupID, qq int64, text string) ([]byte, error) {
	if groupID <= 0 {
		return nil, errors.New("invalid groupID")
	}

	atSeg, err := marshalSegment(TypeAt, AtData{QQ: qq})
	if err != nil {
		return nil, err
	}
	textSeg, err := marshalSegment(TypeText, TextData{Text: text})
	if err != nil {
		return nil, err
	}

	req := WSRequest[SendGroupSegmentsParams]{
		Action: ActionSendGroupMsg,
		Params: SendGroupSegmentsParams{
			GroupID: groupID,
			Message: []MessageSegment{atSeg, textSeg},
		},
	}
	return json.Marshal(req)
}

// MarshalGroupAudioMsg 编码群语音消息
// path 可以是 url 或本地路径
func MarshalGroupAudioMsg(groupID int64, path string) ([]byte, error) {
	if groupID <= 0 || path == "" {
		return nil, errors.New("invalid groupID or empty path")
	}

	seg, err := marshalSegment(TypeAudio, AudioVideoData{File: path})
	if err != nil {
		return nil, err
	}

	req := WSRequest[SendGroupSegmentsParams]{
		Action: ActionSendGroupMsg,
		Params: SendGroupSegmentsParams{
			GroupID: groupID,
			Message: []MessageSegment{seg},
		},
	}
	return json.Marshal(req)
}

// MarshalGroupVideoMsg 编码群视频消息
// path 可以是 url 或本地路径
func MarshalGroupVideoMsg(groupID int64, path string) ([]byte, error) {
	if groupID <= 0 || path == "" {
		return nil, errors.New("invalid groupID or empty path")
	}

	seg, err := marshalSegment(TypeVideo, AudioVideoData{File: path})
	if err != nil {
		return nil, err
	}

	req := WSRequest[SendGroupSegmentsParams]{
		Action: ActionSendGroupMsg,
		Params: SendGroupSegmentsParams{
			GroupID: groupID,
			Message: []MessageSegment{seg},
		},
	}
	return json.Marshal(req)
}

// MarshalGroupImgMsg 编码群图片消息
// path 可以是 url 或本地路径
func MarshalGroupImgMsg(groupID int64, path string) ([]byte, error) {
	if groupID <= 0 || path == "" {
		return nil, errors.New("invalid groupID or empty image URL")
	}

	seg, err := marshalSegment(TypeImage, ImageData{
		File:    path,
		Summary: "[图片]",
	})
	if err != nil {
		return nil, err
	}

	req := WSRequest[SendGroupSegmentsParams]{
		Action: ActionSendGroupMsg,
		Params: SendGroupSegmentsParams{
			GroupID: groupID,
			Message: []MessageSegment{seg},
		},
	}
	return json.Marshal(req)
}

// MarshalGroupFileMsg 编码群文件消息
// name 是文件名；path 可以是 url 或本地路径
func MarshalGroupFileMsg(groupID int64, path string, name string) ([]byte, error) {
	if groupID <= 0 || path == "" || name == "" {
		return nil, errors.New("invalid groupID, empty path or name")
	}

	seg, err := marshalSegment(TypeFile, FileData{File: path, Name: name})
	if err != nil {
		return nil, err
	}

	req := WSRequest[SendGroupSegmentsParams]{
		Action: ActionSendGroupMsg,
		Params: SendGroupSegmentsParams{
			GroupID: groupID,
			Message: []MessageSegment{seg},
		},
	}
	return json.Marshal(req)
}

// MarshalGroupFaceMsg 编码群表情消息
// faceID 参考: https://bot.q.qq.com/wiki/develop/api-v2/openapi/emoji/model.html#EmojiType
func MarshalGroupFaceMsg(groupID int64, faceID int64) ([]byte, error) {
	if groupID <= 0 {
		return nil, errors.New("invalid groupID")
	}

	seg, err := marshalSegment(TypeFace, FaceData{ID: faceID})
	if err != nil {
		return nil, err
	}

	req := WSRequest[SendGroupSegmentsParams]{
		Action: ActionSendGroupMsg,
		Params: SendGroupSegmentsParams{
			GroupID: groupID,
			Message: []MessageSegment{seg},
		},
	}
	return json.Marshal(req)
}

// MarshalGroupReplyMsg 编码群回复消息
// messageID 是回复的消息 ID，text 是回复文本
func MarshalGroupReplyMsg(groupID int64, messageID int64, text string) ([]byte, error) {
	if groupID <= 0 || messageID <= 0 {
		return nil, errors.New("invalid groupID or messageID")
	}

	replySeg, err := marshalSegment(TypeReply, ReplyData{ID: messageID})
	if err != nil {
		return nil, err
	}
	textSeg, err := marshalSegment(TypeText, TextData{Text: text})
	if err != nil {
		return nil, err
	}

	req := WSRequest[SendGroupSegmentsParams]{
		Action: ActionSendGroupMsg,
		Params: SendGroupSegmentsParams{
			GroupID: groupID,
			Message: []MessageSegment{replySeg, textSeg},
		},
	}
	return json.Marshal(req)
}

// MarshalGroupMusicMsg 编码群音乐消息
func MarshalGroupMusicMsg(groupID int64, musicType string, musicID string) ([]byte, error) {
	if groupID <= 0 || musicID == "" {
		return nil, errors.New("invalid groupID or musicID")
	}

	seg, err := marshalSegment(TypeMusic, MusicData{Type: musicType, ID: musicID})
	if err != nil {
		return nil, err
	}

	req := WSRequest[SendGroupSegmentsParams]{
		Action: ActionSendGroupMsg,
		Params: SendGroupSegmentsParams{
			GroupID: groupID,
			Message: []MessageSegment{seg},
		},
	}
	return json.Marshal(req)
}

// MarshalLikeMsg 编码点赞消息
func MarshalLikeMsg(userID int64, times int) ([]byte, error) {
	req := WSRequest[LikeMsgParams]{
		Action: "send_like",
		Params: LikeMsgParams{
			UserID: userID,
			Times:  times,
		},
	}
	return json.Marshal(req)
}

// MarshalGetGroupFileURL 获取群文件链接
func MarshalGetGroupFileURL[T ID](groupID T, fileID string, busID ...int64) ([]byte, error) {
	if fileID == "" {
		return nil, errors.New("empty file_id")
	}

	switch v := any(groupID).(type) {
	case string:
		if v == "" {
			return nil, errors.New("empty group_id")
		}
	case int:
		if v <= 0 {
			return nil, errors.New("invalid group_id")
		}
	case int64:
		if v <= 0 {
			return nil, errors.New("invalid group_id")
		}
	}

	params := GetGroupFileURLParams[T]{
		GroupID: groupID,
		FileID:  fileID,
	}
	if len(busID) > 0 {
		params.BusID = busID[0]
	}

	req := WSRequest[GetGroupFileURLParams[T]]{
		Action: "get_group_file_url",
		Params: params,
	}
	return json.Marshal(req)
}

// MarshalGetPrivateFileURL 获取私聊文件下载链接
func MarshalGetPrivateFileURL(fileID string) ([]byte, error) {
	if fileID == "" {
		return nil, errors.New("empty file_id")
	}
	req := WSRequest[GetPrivateFileURLParams]{
		Action: "get_private_file_url",
		Params: GetPrivateFileURLParams{FileID: fileID},
	}
	return json.Marshal(req)
}

// MarshalGetGroupMemberList 获取群成员列表
func MarshalGetGroupMemberList(groupID int64) ([]byte, error) {
	if groupID <= 0 {
		return nil, errors.New("invalid groupID")
	}
	req := WSRequest[GetGroupMemberListParams]{
		Action: "get_group_member_list",
		Params: GetGroupMemberListParams{GroupID: groupID},
	}
	return json.Marshal(req)
}
