package napcat

import "testing"

func TestDispatcher_OnPrivateTextFile(t *testing.T) {
	t.Parallel()

	d := NewDispatcher()

	var textCount, fileCount int
	d.OnPrivateText(func(*PrivateMessageEvent) { textCount++ })
	d.OnPrivateFile(func(*PrivateMessageEvent) { fileCount++ })

	textMsg := []byte(`{"post_type":"message","message_type":"private","sub_type":"friend","time":1,"self_id":1,"user_id":2,"message_id":3,"message_seq":3,"raw_message":"hi","message_format":"array","sender":{"user_id":2,"nickname":"n"},"message":[{"type":"text","data":{"text":"hi"}}]}`)
	if err := d.HandleBytes(textMsg); err != nil {
		t.Fatalf("HandleBytes(text) error: %v", err)
	}
	if textCount != 1 || fileCount != 0 {
		t.Fatalf("unexpected counts: text=%d file=%d", textCount, fileCount)
	}

	fileMsg := []byte(`{"post_type":"message","message_type":"private","sub_type":"friend","time":1,"self_id":1,"user_id":2,"message_id":3,"message_seq":3,"raw_message":"[file]","message_format":"array","sender":{"user_id":2,"nickname":"n"},"message":[{"type":"file","data":{"file":"a.txt","file_id":"id","file_size":1}}]}`)
	if err := d.HandleBytes(fileMsg); err != nil {
		t.Fatalf("HandleBytes(file) error: %v", err)
	}
	if textCount != 1 || fileCount != 1 {
		t.Fatalf("unexpected counts: text=%d file=%d", textCount, fileCount)
	}
}
