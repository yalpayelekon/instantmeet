package services

import "testing"

func TestHTTPHost(t *testing.T) {
	cases := map[string]string{
		"ws://livekit:7880":             "http://livekit:7880",
		"wss://livekit.example.com":     "https://livekit.example.com",
		"http://livekit:7880":           "http://livekit:7880",
		"https://livekit.example.com":   "https://livekit.example.com",
	}
	for in, want := range cases {
		if got := httpHost(in); got != want {
			t.Fatalf("httpHost(%q)=%q want %q", in, got, want)
		}
	}
}

func TestIsRoomMissing(t *testing.T) {
	if !isRoomMissing(errString("twirp error not_found: room does not exist")) {
		t.Fatal("expected missing room")
	}
	if isRoomMissing(errString("connection refused")) {
		t.Fatal("did not expect missing room")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
