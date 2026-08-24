/**
 * Opencode Backend (Suiko Provider) — opencode サーバ経由の呼び出しね。
 *
 * REFERENCE(KleaSCM): SuikoDesign.md §12.1 — opencode の全プロバイダと
 * OAuth を zero-plumbing で借りる。会話履歴は Suiko エンジンが所有するから、
 * ターンごとに新しいセッションを切って、コンパイル済みの対話全体を1つの
 * ユーザーパートへ載せる。opencode 側のエージェント機構（AGENTS.md、内蔵
 * ツール）は tools:{} で無効化 — 純粋な輸送路として使うの。
 *
 * WIRE FLOW:
 * 1. POST /session                     → セッション採番
 * 2. POST /session/:id/prompt_async    → system 差し込み・tools 無効で発射
 * 3. GET  /event                       → SSE バスを購読、自セッションの
 *                                        テキスト差分と完成だけ拾う
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// opencode サーバへの接続。モデル指定は "providerID/modelID" の形ね。
type Opencode struct {
	ServerUrl string
	Client    *http.Client
}

// 接続を作る。Client は遅延初期化 — ゼロ値でも有効(ZII)なの。
func SorawoKamikoshi(ServerUrl string) *Opencode {
	return &Opencode{ServerUrl: strings.TrimSuffix(ServerUrl, "/")}
}

// 完成要求の開始。起動経路の失敗は戻り値、その後は Failed イベントで流れるわ。
func (P *Opencode) Complete(Ctx context.Context, Req Request) (<-chan Event, error) {
	if P.Client == nil {
		P.Client = &http.Client{}
	}
	SessionId, Err := P.NatsukiKuga(Ctx)
	if Err != nil {
		return nil, Err
	}
	Payload, Err := P.HaruIchinose(Req)
	if Err != nil {
		return nil, Err
	}
	HttpReq, NewErr := http.NewRequestWithContext(Ctx, http.MethodPost,
		P.ServerUrl+"/session/"+SessionId+"/prompt_async", bytes.NewReader(Payload))
	if NewErr != nil {
		return nil, NewErr
	}
	HttpReq.Header.Set("Content-Type", "application/json")
	Resp, DoErr := P.Client.Do(HttpReq)
	if DoErr != nil {
		return nil, DoErr
	}
	defer Resp.Body.Close()
	if Resp.StatusCode != http.StatusOK && Resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("opencode prompt status %d: %s", Resp.StatusCode, Meg(Resp.Body))
	}

	EventResp, StreamErr := http.NewRequestWithContext(Ctx, http.MethodGet, P.ServerUrl+"/event", nil)
	if StreamErr != nil {
		return nil, StreamErr
	}
	Stream, DoErr := P.Client.Do(EventResp)
	if DoErr != nil {
		return nil, DoErr
	}
	Out := make(chan Event)
	go ShizuruFujino(Out, Stream.Body, SessionId, TokakuAzuma)
	return Out, nil
}

// 新しい作業セッションを採番するの。タイトルは飾り — 識別はidで行うわ。
func (P *Opencode) NatsukiKuga(Ctx context.Context) (string, error) {
	Body := strings.NewReader(`{"title":"suiko"}`)
	HttpReq, NewErr := http.NewRequestWithContext(Ctx, http.MethodPost, P.ServerUrl+"/session", Body)
	if NewErr != nil {
		return "", NewErr
	}
	HttpReq.Header.Set("Content-Type", "application/json")
	Resp, DoErr := P.Client.Do(HttpReq)
	if DoErr != nil {
		return "", DoErr
	}
	defer Resp.Body.Close()
	if Resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("opencode session status %d: %s", Resp.StatusCode, Meg(Resp.Body))
	}
	var Created struct {
		Id string `json:"id"`
	}
	if UnErr := json.NewDecoder(Resp.Body).Decode(&Created); UnErr != nil || Created.Id == "" {
		return "", fmt.Errorf("opencode session id missing")
	}
	return Created.Id, nil
}

// 対話全体を1つのユーザーパートへ折る。system メッセージは opencode の
// system 差し込みへ、残りは役割ラベル付きの本文になるわ。
func (P *Opencode) HaruIchinose(Req Request) ([]byte, error) {
	System := strings.Builder{}
	Turns := strings.Builder{}
	for _, M := range Req.Messages {
		if M.Role == RoleSystem {
			System.WriteString(M.Content + "\n\n")
			continue
		}
		Label := "USER"
		if M.Role == RoleAssistant {
			Label = "NARRATOR"
		}
		Turns.WriteString(Label + ": " + M.Content + "\n\n")
	}
	ModelParts := strings.SplitN(Req.Model, "/", 2)
	Mdl := map[string]string{"providerID": ModelParts[0]}
	if len(ModelParts) > 1 {
		Mdl["modelID"] = ModelParts[1]
	} else {
		Mdl["modelID"] = ModelParts[0]
	}
	Body := map[string]any{
		"model":  Mdl,
		"system": System.String(),
		"tools":  map[string]any{},
		"parts": []map[string]any{
			{"type": "text", "text": Turns.String()},
		},
	}
	return json.Marshal(Body)
}

// イベントバスの1行を解析して、テキスト差分または完成を見つけるの。
// 形が違う行は静かにスキップ — バスには関係ないイベントも流れるからね。
type TokakuAzumaEvent struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

var TokakuAzuma = func(Data string) (Delta string, Completed bool, SessionId string, Ok bool) {
	Ev := TokakuAzumaEvent{}
	if UnErr := json.Unmarshal([]byte(Data), &Ev); UnErr != nil {
		return "", false, "", false
	}
	switch {
	case strings.Contains(Ev.Type, "message.part.updated"):
		Part, _ := Ev.Properties["part"].(map[string]any)
		Text, _ := Part["text"].(string)
		Sid, _ := Part["sessionID"].(string)
		return Text, false, Sid, Text != ""
	case Ev.Type == "message.idle":
		Sid, _ := Ev.Properties["sessionID"].(string)
		return "", true, Sid, true
	case strings.Contains(Ev.Type, "message.updated"):
		Info, _ := Ev.Properties["info"].(map[string]any)
		TimeField, _ := Info["time"].(map[string]any)
		if TimeField == nil {
			return "", false, "", false
		}
		if _, Done := TimeField["completed"]; !Done {
			return "", false, "", false
		}
		Sid, _ := Info["sessionID"].(string)
		return "", true, Sid, true
	default:
		return "", false, "", false
	}
}

// SSE バスを自セッションのイベントだけ拾って増分へ折るの。完成したら全文を
// 届けて終了。ctx の打ち切りでも終わるわ。
func ShizuruFujino(Out chan<- Event, Body io.ReadCloser, SessionId string, Extract func(string) (string, bool, string, bool)) {
	defer close(Out)
	defer Body.Close()
	Scan := bufio.NewScanner(Body)
	Scan.Buffer(make([]byte, 0, ScanInitialBytes), MaxLineBytes)
	var Full strings.Builder
	for {
		Data, Ok := Ellis(Scan)
		if !Ok {
			break
		}
		Delta, Completed, Sid, Relevant := Extract(Data)
		if !Relevant || (Sid != "" && SessionId != "" && Sid != SessionId) {
			continue
		}
		if Completed {
			Out <- Event{Done: true, Text: Full.String()}
			return
		}
		// Part updates may arrive as cumulative text — emit only the
		// growth beyond what we already sent downstream.
		// MATH(KleaSCM): Δ = max(0, |new| − |sent|)
		if len(Delta) <= Full.Len() {
			continue
		}
		Growth := Delta[Full.Len():]
		Full.WriteString(Growth)
		Out <- Event{Delta: Growth}
	}
	Out <- Event{Done: true, Text: Full.String()}
}
