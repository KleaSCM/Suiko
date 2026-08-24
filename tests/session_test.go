/**
 * Provider・Session Tests — 輸送路と回転軸の意図付きカバレッジね。
 *
 * プロバイダは本物のネットに触れない — SSE 形状を再現した偽サーバと
 * 偽プロバイダで証明する。セッションは注入→履歴→要約圧縮→審査行列まで
 * 通しで回すわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"suiko/internal/provider"
	"suiko/internal/session"
	"suiko/internal/world"
)

// 偽プロバイダ：受けた system の内容を記憶して、決まった応答を1回で流すの。
type FakeProvider struct {
	LastSystem  string
	LastHistory []provider.Message
	Reply       string
}

func (F *FakeProvider) Complete(Ctx context.Context, Req provider.Request) (<-chan provider.Event, error) {
	for _, M := range Req.Messages {
		if M.Role == provider.RoleSystem {
			F.LastSystem = M.Content
			continue
		}
		F.LastHistory = append(F.LastHistory, M)
	}
	Out := make(chan provider.Event, 2)
	Out <- provider.Event{Delta: F.Reply}
	Out <- provider.Event{Done: true, Text: F.Reply}
	close(Out)
	return Out, nil
}

func NewSessionForTest(T *testing.T, Reply string) (*session.Session, *FakeProvider) {
	T.Helper()
	S := LoadFixture(T)
	P := &FakeProvider{Reply: Reply}
	return session.KyoukoToudou(S, P), P
}

// ターンが文脈を組んで、履歴を伸ばして、発火idを返すことの通し証明ね。
func TestSessionTurnAssemblesContext(t *testing.T) {
	Sess, P := NewSessionForTest(t, "The forge glows warmly.")
	Res, Err := Sess.YukiFukuzawa(context.Background(), "I walk to the bakery.", nil)
	if Err != nil {
		t.Fatalf("turn: %v", Err)
	}
	if Res.Text != "The forge glows warmly." {
		t.Fatalf("reply text wrong: %q", Res.Text)
	}
	if len(Res.Fired) == 0 || !strings.Contains(strings.Join(Res.Fired, ","), "loc/bakery") {
		t.Fatalf("bakery alias must fire: %v", Res.Fired)
	}
	if !strings.Contains(P.LastSystem, "[CANON]") {
		t.Fatalf("system missing canon: %s", P.LastSystem[:80])
	}
	if !strings.Contains(P.LastSystem, "[LORE]") {
		t.Fatal("system must carry the lore block")
	}
	// 二ターン目は履歴を持ってくること。
	P.LastHistory = nil
	if _, Err = Sess.YukiFukuzawa(context.Background(), "I warm my hands.", nil); Err != nil {
		t.Fatalf("turn2: %v", Err)
	}
	if len(P.LastHistory) != 3 { // user, assistant, user ね。
		t.Fatalf("history not carried: %d messages", len(P.LastHistory))
	}
}

// 履歴上限を超えると古い半分が要約へ折り畳まれることの証明ね。
func TestSessionCompressesHistory(t *testing.T) {
	Sess, _ := NewSessionForTest(t, "ok")
	Sess.HistoryCap = 4
	for I := 0; I < 6; I++ {
		if _, Err := Sess.YukiFukuzawa(context.Background(), "turn text", nil); Err != nil {
			t.Fatalf("turn %d: %v", I, Err)
		}
	}
	if len(Sess.History) > 4 {
		t.Fatalf("history exceeded cap: %d", len(Sess.History))
	}
	if Sess.Digest == "" {
		t.Fatal("compression must produce a digest")
	}
}

func TestSessionDedupSuppressesRepeatInjection(t *testing.T) {
	Sess, P := NewSessionForTest(t, "ok")
	if _, Err := Sess.YukiFukuzawa(context.Background(), "at the bakery", nil); Err != nil {
		t.Fatal(Err)
	}
	if !strings.Contains(P.LastSystem, "[LORE]") {
		t.Fatal("first turn must inject")
	}
	// 直後の同じ話題は重複排除窓に沈むの。
	P.LastSystem = ""
	if _, Err := Sess.YukiFukuzawa(context.Background(), "still at the bakery", nil); Err != nil {
		t.Fatal(Err)
	}
	if strings.Contains(P.LastSystem, "[LORE]") {
		t.Fatal("repeat injection must be deduped within window")
	}
}

// OpenAI互換SSEの復号：増分、[DONE]、エラー行の扱いを偽サーバで証明するわ。
func TestOpenAiBackendParsesStream(t *testing.T) {
	Server := httptest.NewServer(http.HandlerFunc(func(W http.ResponseWriter, R *http.Request) {
		W.Header().Set("Content-Type", "text/event-stream")
		W.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"he\"}}]}\n\n"))
		W.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"llo\"}}]}\n\n"))
		W.Write([]byte("data: [DONE]\n\n"))
	}))
	defer Server.Close()

	P := &provider.OpenAI{BaseUrl: Server.URL}
	Stream, Err := P.Complete(context.Background(), provider.Request{Model: "test"})
	if Err != nil {
		t.Fatalf("complete: %v", Err)
	}
	Text := ""
	for Ev := range Stream {
		if Ev.Failed {
			t.Fatalf("stream failed: %s", Ev.Message)
		}
		if Ev.Done {
			Text = Ev.Text
		}
	}
	if Text != "hello" {
		t.Fatalf("stream text wrong: %q", Text)
	}
}

func TestOpenAiBackendSurfacesErrorBody(t *testing.T) {
	Server := httptest.NewServer(http.HandlerFunc(func(W http.ResponseWriter, R *http.Request) {
		http.Error(W, "model exploded", http.StatusInternalServerError)
	}))
	defer Server.Close()

	P := &provider.OpenAI{BaseUrl: Server.URL}
	_, Err := P.Complete(context.Background(), provider.Request{Model: "test"})
	if Err == nil || !strings.Contains(Err.Error(), "model exploded") {
		t.Fatalf("error body must surface: %v", Err)
	}
}

// opencode バックエンド：セッション採番 → prompt_async → イベントバス購読の
// ワイヤ順序を偽サーバで通しで証明するね。
func TestOpencodeBackendFullFlow(t *testing.T) {
	Mux := http.NewServeMux()
	SessionSeen := false
	Mux.HandleFunc("POST /session", func(W http.ResponseWriter, R *http.Request) {
		SessionSeen = true
		W.Write([]byte(`{"id":"sess-1"}`))
	})
	var PromptBody map[string]any
	Mux.HandleFunc("POST /session/sess-1/prompt_async", func(W http.ResponseWriter, R *http.Request) {
		json.NewDecoder(R.Body).Decode(&PromptBody)
		W.WriteHeader(http.StatusNoContent)
	})
	Mux.HandleFunc("GET /event", func(W http.ResponseWriter, R *http.Request) {
		W.Header().Set("Content-Type", "text/event-stream")
		W.Write([]byte("data: {\"type\":\"message.part.updated\",\"properties\":{\"part\":{\"text\":\"sta\",\"sessionID\":\"sess-1\"}}}\n\n"))
		W.Write([]byte("data: {\"type\":\"message.part.updated\",\"properties\":{\"part\":{\"text\":\"start\",\"sessionID\":\"sess-1\"}}}\n\n"))
		W.Write([]byte("data: {\"type\":\"other.event\",\"properties\":{}}\n\n"))
		W.Write([]byte("data: {\"type\":\"message.idle\",\"properties\":{\"sessionID\":\"sess-1\"}}\n\n"))
	})
	Server := httptest.NewServer(Mux)
	defer Server.Close()

	P := provider.SorawoKamikoshi(Server.URL)
	Stream, Err := P.Complete(context.Background(), provider.Request{
		Model:    "prov/model-x",
		Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}},
	})
	if Err != nil {
		t.Fatalf("complete: %v", Err)
	}
	Text := ""
	for Ev := range Stream {
		if Ev.Done {
			Text = Ev.Text
		}
	}
	if !SessionSeen {
		t.Fatal("session must be created first")
	}
	if Text != "start" {
		t.Fatalf("cumulative parts must fold into full text: %q", Text)
	}
	System, _ := PromptBody["system"].(string)
	if System != "" {
		t.Fatalf("no system expected for bare request: %q", System)
	}
	Tools, _ := PromptBody["tools"].(map[string]any)
	if Tools == nil {
		t.Fatal("tools must be explicitly disabled")
	}
	Model, _ := PromptBody["model"].(map[string]any)
	if Model["modelID"] != "model-x" || Model["providerID"] != "prov" {
		t.Fatalf("model split wrong: %v", Model)
	}
}

// 審査行列：自動承認OFFでは滞留し、Accept で通る。Reject は黙って捨てるわ。
func TestReviewQueueGatesWrites(t *testing.T) {
	R := session.NewReview(false)
	Applied := false
	Id, Err := R.Submit("add loc/mill", "add-entry", 3, func() world.Error {
		Applied = true
		return world.Error{}
	})
	if Err.Ok() && Applied {
		t.Fatal("gated write must not apply immediately")
	}
	if len(R.List()) != 1 {
		t.Fatal("write must wait in queue")
	}
	if AcceptErr := R.Accept(Id); !AcceptErr.Ok() || !Applied {
		t.Fatal("accept must apply the write")
	}
	if len(R.List()) != 0 {
		t.Fatal("queue must drain after accept")
	}

	Id2, _ := R.Submit("log x", "log-event", 4, func() world.Error { return world.Error{} })
	R.Reject(Id2)
	if len(R.List()) != 0 {
		t.Fatal("rejected write must leave the queue")
	}
	if AcceptErr := R.Accept(Id2); !AcceptErr.Ok() {
		t.Fatal("double-processing unknown id must stay calm")
	}
}

func TestReviewAutoAcceptAppliesImmediately(t *testing.T) {
	R := session.NewReview(true)
	Applied := false
	_, Err := R.Submit("x", "log-event", 1, func() world.Error {
		Applied = true
		return world.Error{}
	})
	if !Err.Ok() || !Applied || len(R.List()) != 0 {
		t.Fatal("auto accept must bypass the queue")
	}
}
