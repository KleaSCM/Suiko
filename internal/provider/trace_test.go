//go:build live

// session.error がなぜ Failed にならないか、行ごとに ParseEvent の返り値を
// 覗いて特定するの。
//
// Author: KleaSCM
// Email: KleaSCM@gmail.com
package provider

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func peekType(Data string) string {
	var T struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal([]byte(Data), &T)
	return T.Type
}

func TestTraceParseEvent(t *testing.T) {
	Raw, Err := os.ReadFile("/tmp/event2.log")
	if Err != nil {
		t.Skip("capture missing")
	}
	SeenUser := map[string]bool{}
	for _, Line := range strings.Split(string(Raw), "\n") {
		Line = strings.TrimSpace(Line)
		if !strings.HasPrefix(Line, "data:") {
			continue
		}
		Data := strings.TrimSpace(strings.TrimPrefix(Line, "data:"))
		D, C, F, S := ParseEvent(SeenUser, Data)
		t.Logf("type=%q completed=%v failed=%v sid=%q delta=%q", peekType(Data), C, F, S, truncate(D, 50))
	}
}
