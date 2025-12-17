package napcat_ws

import "testing"

var testPrivateMsg = []byte(`{"post_type":"message","message_type":"private","time":1,"self_id":1,"user_id":2,"message":[{"type":"text","data":{"text":"hi"}}]}`)

func TestDispatcher_Middleware_OrderAndNext(t *testing.T) {
	t.Parallel()

	d := NewDispatcher()

	var seq []string
	d.Use(func(_ *MessageEvent, next func()) {
		seq = append(seq, "m1-before")
		next()
		seq = append(seq, "m1-after")
	})
	d.Use(func(_ *MessageEvent, next func()) {
		seq = append(seq, "m2-before")
		next()
		seq = append(seq, "m2-after")
	})
	d.UsePrivate(func(_ *PrivateMessageEvent, next func()) {
		seq = append(seq, "p1-before")
		next()
		seq = append(seq, "p1-after")
	})
	d.OnPrivate(func(*PrivateMessageEvent) { seq = append(seq, "h1") })
	d.OnPrivate(func(*PrivateMessageEvent) { seq = append(seq, "h2") })

	if err := d.HandleBytes(testPrivateMsg); err != nil {
		t.Fatalf("HandleBytes() error: %v", err)
	}

	want := []string{"m1-before", "m2-before", "p1-before", "h1", "h2", "p1-after", "m2-after", "m1-after"}
	if len(seq) != len(want) {
		t.Fatalf("unexpected seq: got=%+v want=%+v", seq, want)
	}
	for i := range want {
		if seq[i] != want[i] {
			t.Fatalf("unexpected seq at %d: got=%+v want=%+v", i, seq, want)
		}
	}
}

func TestDispatcher_Middleware_Abort(t *testing.T) {
	t.Parallel()

	d := NewDispatcher()

	var seq []string
	d.Use(func(_ *MessageEvent, next func()) {
		seq = append(seq, "m-before")
		next()
		seq = append(seq, "m-after")
	})
	d.UsePrivate(func(*PrivateMessageEvent, func()) {
		seq = append(seq, "p-before")
		seq = append(seq, "p-abort")
	})
	d.OnPrivate(func(*PrivateMessageEvent) { seq = append(seq, "handler") })

	if err := d.HandleBytes(testPrivateMsg); err != nil {
		t.Fatalf("HandleBytes() error: %v", err)
	}

	want := []string{"m-before", "p-before", "p-abort", "m-after"}
	if len(seq) != len(want) {
		t.Fatalf("unexpected seq: got=%+v want=%+v", seq, want)
	}
	for i := range want {
		if seq[i] != want[i] {
			t.Fatalf("unexpected seq at %d: got=%+v want=%+v", i, seq, want)
		}
	}
}

func TestDispatcher_Middleware_PanicStillContinues(t *testing.T) {
	t.Parallel()

	d := NewDispatcher()

	var panics int
	d.OnPanic(func(PanicInfo) { panics++ })

	d.UsePrivate(func(*PrivateMessageEvent, func()) { panic("boom") })

	var handled int
	d.OnPrivate(func(*PrivateMessageEvent) { handled++ })

	if err := d.HandleBytes(testPrivateMsg); err != nil {
		t.Fatalf("HandleBytes() error: %v", err)
	}
	if handled != 1 {
		t.Fatalf("expected handled=1, got %d", handled)
	}
	if panics != 1 {
		t.Fatalf("expected panics=1, got %d", panics)
	}
}

func TestDispatcher_Middleware_HandlerPanicIsolated(t *testing.T) {
	t.Parallel()

	d := NewDispatcher()

	var panics int
	d.OnPanic(func(PanicInfo) { panics++ })

	var seq []string
	d.Use(func(_ *MessageEvent, next func()) {
		seq = append(seq, "m-before")
		next()
		seq = append(seq, "m-after")
	})
	d.OnPrivate(func(*PrivateMessageEvent) {
		seq = append(seq, "h1")
		panic("handler panic")
	})
	d.OnPrivate(func(*PrivateMessageEvent) { seq = append(seq, "h2") })

	if err := d.HandleBytes(testPrivateMsg); err != nil {
		t.Fatalf("HandleBytes() error: %v", err)
	}

	want := []string{"m-before", "h1", "h2", "m-after"}
	if len(seq) != len(want) {
		t.Fatalf("unexpected seq: got=%+v want=%+v", seq, want)
	}
	for i := range want {
		if seq[i] != want[i] {
			t.Fatalf("unexpected seq at %d: got=%+v want=%+v", i, seq, want)
		}
	}
	if panics != 1 {
		t.Fatalf("expected panics=1, got %d", panics)
	}
}
