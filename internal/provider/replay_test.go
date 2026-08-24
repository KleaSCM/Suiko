//go:build live

// 実際に取得した 402 ストリームを ShizuruFujino に通して、session.error が
// 本当に Failed イベントになるかを確認するの。これが Failed にならなければ
// エラーターンが無限待ちになるからね。
//
// Author: KleaSCM
// Email: KleaSCM@gmail.com
package provider

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestReplaySessionError(t *testing.T) {
	Raw, Err := os.ReadFile("/tmp/event2.log")
	if Err != nil {
		t.Skip("capture file missing:", Err)
	}
	Body := io.NopCloser(strings.NewReader(string(Raw)))
	Out := make(chan Event)
	SeenUser := map[string]bool{}
	SID := "ses_fcbc22806ffe9UeZWicFJWsbRy"
	go ShizuruFujino(context.Background(), Out, Body, SID, func(Data string) (string, bool, bool, string) {
		return ParseEvent(SeenUser, Data)
	})

	var GotDelta, GotDone, GotFailed int
	var Msg string
	for Ev := range Out {
		if Ev.Delta != "" {
			GotDelta++
		}
		if Ev.Done {
			GotDone++
		}
		if Ev.Failed {
			GotFailed++
			Msg = Ev.Message
		}
	}
	t.Logf("deltas=%d done=%d failed=%d msg=%q", GotDelta, GotDone, GotFailed, Msg)
	if GotFailed == 0 {
		t.Fatalf("session.error was NOT surfaced as Failed — error turns would hang")
	}
}
