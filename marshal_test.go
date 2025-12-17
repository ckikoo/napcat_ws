package napcat_ws

import (
	"encoding/json"
	"testing"
)

func TestMarshalGetGroupFileURL(t *testing.T) {
	t.Parallel()

	b, err := MarshalGetGroupFileURL("123", "file_1")
	if err != nil {
		t.Fatalf("MarshalGetGroupFileURL() error: %v", err)
	}
	var req WSRequest[GetGroupFileURLParams[string]]
	if err := json.Unmarshal(b, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Action != "get_group_file_url" {
		t.Fatalf("unexpected action: %q", req.Action)
	}
	if req.Params.GroupID != "123" || req.Params.FileID != "file_1" {
		t.Fatalf("unexpected params: %+v", req.Params)
	}
	if req.Params.BusID != 0 {
		t.Fatalf("unexpected busid: %d", req.Params.BusID)
	}

	b, err = MarshalGetGroupFileURL("123", "file_1", 99)
	if err != nil {
		t.Fatalf("MarshalGetGroupFileURL() error: %v", err)
	}
	if err := json.Unmarshal(b, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Params.BusID != 99 {
		t.Fatalf("unexpected busid: %d", req.Params.BusID)
	}
}

func TestMarshalGetPrivateFileURL(t *testing.T) {
	t.Parallel()

	b, err := MarshalGetPrivateFileURL("file_1")
	if err != nil {
		t.Fatalf("MarshalGetPrivateFileURL() error: %v", err)
	}
	var req WSRequest[GetPrivateFileURLParams]
	if err := json.Unmarshal(b, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Action != "get_private_file_url" {
		t.Fatalf("unexpected action: %q", req.Action)
	}
	if req.Params.FileID != "file_1" {
		t.Fatalf("unexpected params: %+v", req.Params)
	}
}

func TestMarshalGetGroupMemberList(t *testing.T) {
	t.Parallel()

	b, err := MarshalGetGroupMemberList(123)
	if err != nil {
		t.Fatalf("MarshalGetGroupMemberList() error: %v", err)
	}
	var req WSRequest[GetGroupMemberListParams]
	if err := json.Unmarshal(b, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Action != "get_group_member_list" {
		t.Fatalf("unexpected action: %q", req.Action)
	}
	if req.Params.GroupID != 123 {
		t.Fatalf("unexpected params: %+v", req.Params)
	}
}

func TestMarshalPrivateTextMsg(t *testing.T) {
	t.Parallel()

	b, err := MarshalPrivateTextMsg(123, "hi")
	if err != nil {
		t.Fatalf("MarshalPrivateTextMsg() error: %v", err)
	}
	var req WSRequest[SendPrivateTextParams]
	if err := json.Unmarshal(b, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Action != ActionSendPrivateMsg {
		t.Fatalf("unexpected action: %q", req.Action)
	}
	if req.Params.UserID != 123 || req.Params.Message != "hi" {
		t.Fatalf("unexpected params: %+v", req.Params)
	}
}
