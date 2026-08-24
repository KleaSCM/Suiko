//go:build live

// リアル形状リクエストで生サーバを叩き、イベントの流れを観測するの。
// セッションの system(長文ロア) + 複数 parts を模して、session.idle が
// 本当に来るか / 来ないかを確認するためね。
//
// Author: KleaSCM
// Email: KleaSCM@gmail.com
package provider

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestLiveRealShapeDoesNotHang(t *testing.T) {
	Oc := SorawoKamikoshi("http://127.0.0.1:4096")

	System := `You are Suiko, a narrative engine. ` +
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
		"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
		"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris. " +
		"Duis aute irure dolor in reprehenderit in voluptate velit esse. " +
		"Excepteur sint occaecat cupidatat non proident, sunt in culpa. " +
		"Qui officia deserunt mollit anim id est laborum. " +
		"Repeat x3 to lengthen the system prompt for realism."
	Parts := []Message{
		{Role: RoleUser, Content: "Hello, who are you?"},
		{Role: RoleAssistant, Content: "I am Suiko, your narrator."},
		{Role: RoleUser, Content: "Tell me a single short sentence about a quiet forest."},
	}

	ctx, Cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer Cancel()

	Start := time.Now()
	Stream, Err := Oc.Complete(ctx, Request{
		Model:     "deepseek/deepseek-chat",
		Messages:  append([]Message{{Role: RoleSystem, Content: System}}, Parts...),
		MaxTokens: 512,
	})
	if Err != nil {
		t.Fatalf("Complete returned error up front: %v", Err)
	}

	GotIdle, GotErr := false, false
	var Full string
	for Ev := range Stream {
		Kind := "DELTA"
		if Ev.Done {
			Kind = "DONE"
		}
		if Ev.Failed {
			Kind = "FAILED"
			GotErr = true
		}
		fmt.Printf("[%s] %s (deltaLen=%d failed=%v) :: %q\n", time.Since(Start).Round(time.Millisecond), Kind, len(Ev.Delta), Ev.Failed, truncate(Ev.Message+Ev.Delta, 60))
		if Ev.Delta != "" {
			Full += Ev.Delta
		}
		if Ev.Done {
			GotIdle = true
		}
	}
	fmt.Printf("=== elapsed=%s done=%v failed=%v textLen=%d\n", time.Since(Start).Round(time.Millisecond), GotIdle, GotErr, len(Full))
	if !GotIdle && !GotErr {
		t.Fatalf("NEVER COMPLETED — neither session.idle nor session.error within 120s")
	}
}

func truncate(S string, N int) string {
	R := []rune(S)
	if len(R) <= N {
		return S
	}
	return string(R[:N]) + "…"
}
