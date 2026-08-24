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
	"sync"

	"suiko/internal/opencodeman"
)

// opencode サーバへの接続。モデル指定は "providerID/modelID" の形ね。
type Opencode struct {
	ServerUrl string
	Client    *http.Client
	// 管理モード — worldDir があれば Suiko 専有の opencode へ遅延接続する。
	managed  bool
	worldDir string
	cacheURL string
	cacheErr error
	cacheOne sync.Once
}

// 接続を作る。Client は遅延初期化 — ゼロ値でも有効(ZII)なの。
func SorawoKamikoshi(ServerUrl string) *Opencode {
	return &Opencode{ServerUrl: strings.TrimSuffix(ServerUrl, "/")}
}

// 世界ごとの Suiko 専有 opencode へ繋ぐ — URL は初回呼び出し時に遅延確定。
func SorawoKamikoshiWorld(WorldDir string) *Opencode {
	return &Opencode{managed: true, worldDir: WorldDir}
}

// 接続先 URL — 管理モードなら世界のインスタンスを確保して返す。
func (P *Opencode) url() (string, error) {
	if !P.managed {
		return P.ServerUrl, nil
	}
	P.cacheOne.Do(func() {
		P.cacheURL, P.cacheErr = opencodeman.Nadeshiko(P.worldDir)
	})
	return P.cacheURL, P.cacheErr
}

// 完成要求の開始。起動経路の失敗は戻り値、その後は Failed イベントで流れるわ。
func (P *Opencode) Complete(Ctx context.Context, Req Request) (<-chan Event, error) {
	if P.Client == nil {
		P.Client = &http.Client{}
	}
	Base, UrlErr := P.url()
	if UrlErr != nil {
		return nil, UrlErr
	}
	SessionId, Err := P.NatsukiKuga(Ctx)
	if Err != nil {
		return nil, Err
	}

	// イベント購読を先に開く — prompt_async は Fire&Forget だから、これを
	// 後回しにすると生成中の message.part.updated を丸ごと取りこぼし、
	// session.idle すら届かずに永久待ちになるの。購読を立ち上げてから発射するわ。
	EventReq, NewErr := http.NewRequestWithContext(Ctx, http.MethodGet, Base+"/event", nil)
	if NewErr != nil {
		return nil, NewErr
	}
	Stream, StreamErr := P.Client.Do(EventReq)
	if StreamErr != nil {
		return nil, StreamErr
	}
	Out := make(chan Event)
	// ユーザー発言のエコーを落とすための状態 — ストリームごとに1つね。
	SeenUser := map[string]bool{}
	go ShizuruFujino(Ctx, Out, Stream.Body, SessionId, func(Data string) (string, bool, bool, string) {
		return ParseEvent(SeenUser, Data)
	})

	Payload, Err := P.HaruIchinose(Req)
	if Err != nil {
		Stream.Body.Close()
		return nil, Err
	}
	HttpReq, NewErr := http.NewRequestWithContext(Ctx, http.MethodPost,
		Base+"/session/"+SessionId+"/prompt_async", bytes.NewReader(Payload))
	if NewErr != nil {
		Stream.Body.Close()
		return nil, NewErr
	}
	HttpReq.Header.Set("Content-Type", "application/json")
	Resp, DoErr := P.Client.Do(HttpReq)
	if DoErr != nil {
		Stream.Body.Close()
		return nil, DoErr
	}
	defer Resp.Body.Close()
	if Resp.StatusCode != http.StatusOK && Resp.StatusCode != http.StatusNoContent {
		Stream.Body.Close()
		return nil, fmt.Errorf("opencode prompt status %d: %s", Resp.StatusCode, Meg(Resp.Body))
	}
	return Out, nil
}

// 新しい作業セッションを採番するの。タイトルは飾り — 識別はidで行うわ。
func (P *Opencode) NatsukiKuga(Ctx context.Context) (string, error) {
	Base, UrlErr := P.url()
	if UrlErr != nil {
		return "", UrlErr
	}
	Body := strings.NewReader(`{"title":"suiko"}`)
	HttpReq, NewErr := http.NewRequestWithContext(Ctx, http.MethodPost, Base+"/session", Body)
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
	Mdl := map[string]string{}
	if len(ModelParts) > 1 {
		Mdl["providerID"] = ModelParts[0]
		Mdl["modelID"] = ModelParts[1]
	} else {
		// プロバイダ省略のベアなモデルidは、opencode ネイティブ提供
		// （Zen の自由枠など）とみなす — それでも駄目なら opencode が
		// 404 で教えてくるから、ここは寛容に扱うの。
		Mdl["providerID"] = "opencode"
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

// イベントバスの1行を解析して、テキスト差分・完成・失敗を見つけるの。
// SeenUser はユーザー発言の messageID を覚え、そのエコーを行かさない。
// opencode は完成を "session.idle" で、障害を "session.error" で知らせる —
// 昔の "message.idle" 前提だと完成が拾えずに永久待ちになるから要注意。
type TokakuAzumaEvent struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// 1行を解析して、テキスト差分・完成・失敗を見つけるの。戻り値は
// (差分, 完成フラグ, 失敗フラグ, セッションID)。完成も失敗も立っていなければ
// その行は捨ててよい（関心外）。失敗だけが立つとき、差分には障害メッセージが
// 入る。ユーザー発言の messageID は SeenUser へ覚えて、そのエコーを行かさない。
//
// NOTE(KleaSCM): 旧実装は4番目の戻りを「関心あり」と「成功」で兼任させていて、
// session.error のように Ok=false の終端が「関心外」と誤判定され、エラーターンが
// 丸ごと飲み込まれていた — だから失敗は独立したフラグに分けたの。
func ParseEvent(SeenUser map[string]bool, Data string) (Delta string, Completed bool, Failed bool, Sid string) {
	Ev := TokakuAzumaEvent{}
	if UnErr := json.Unmarshal([]byte(Data), &Ev); UnErr != nil {
		return "", false, false, ""
	}
	switch {
	case strings.Contains(Ev.Type, "message.updated"):
		Info, _ := Ev.Properties["info"].(map[string]any)
		if Info == nil {
			return "", false, false, ""
		}
		Role, _ := Info["role"].(string)
		if Role == "user" {
			if Id, _ := Info["id"].(string); Id != "" {
				SeenUser[Id] = true
			}
			// ユーザー側の time.completed で早退しないよう、完了判定は
			// アシスタント側だけに絞るの。
			return "", false, false, ""
		}
		TimeField, _ := Info["time"].(map[string]any)
		if TimeField == nil {
			return "", false, false, ""
		}
		if _, Done := TimeField["completed"]; !Done {
			return "", false, false, ""
		}
		Sid, _ = Info["sessionID"].(string)
		return "", true, false, Sid
	case strings.Contains(Ev.Type, "message.part.updated"):
		Part, _ := Ev.Properties["part"].(map[string]any)
		Text, _ := Part["text"].(string)
		MsgId, _ := Part["messageID"].(string)
		Sid, _ = Part["sessionID"].(string)
		// ユーザー発言のエコーは流さない — 入力は既に履歴にあるからね。
		if MsgId != "" && SeenUser[MsgId] {
			return "", false, false, Sid
		}
		return Text, false, false, Sid
	case strings.Contains(Ev.Type, "session.idle"):
		Sid, _ = Ev.Properties["sessionID"].(string)
		return "", true, false, Sid
	case strings.Contains(Ev.Type, "session.error"):
		Msg := "opencode session error"
		if ErrField, ok := Ev.Properties["error"].(map[string]any); ok {
			if M, ok := ErrField["message"].(string); ok && M != "" {
				Msg = M
			} else if D, ok := ErrField["data"].(map[string]any); ok {
				if DM, ok := D["message"].(string); ok && DM != "" {
					Msg = DM
				}
			}
		}
		Sid, _ = Ev.Properties["sessionID"].(string)
		return Msg, false, true, Sid
	default:
		return "", false, false, ""
	}
}

// SSE バスを自セッションのイベントだけ拾って増分へ折るの。完成したら全文を
// 届け、失敗なら Failed を流して終了。ctx の打ち切りでも終わるわ。
func ShizuruFujino(Ctx context.Context, Out chan<- Event, Body io.ReadCloser, SessionId string, Extract func(string) (string, bool, bool, string)) {
	defer close(Out)
	defer Body.Close()

	// 行を一つずつ読むのはブロッキングだから、別ゴルーチンへ逃がして
	// ctx の打ち切りを select で拾えるようにするの。
	Lines := make(chan string)
	go func() {
		Scan := bufio.NewScanner(Body)
		Scan.Buffer(make([]byte, 0, ScanInitialBytes), MaxLineBytes)
		for {
			Data, Ok := Ellis(Scan)
			if !Ok {
				break
			}
			select {
			case Lines <- Data:
			case <-Ctx.Done():
				return
			}
		}
		close(Lines)
	}()

	var Full strings.Builder
	for {
		select {
		case <-Ctx.Done():
			return
		case Data, Ok := <-Lines:
			if !Ok {
				Out <- Event{Done: true, Text: Full.String()}
				return
			}
			Delta, Completed, Failed, Sid := Extract(Data)
			// 他セッションのイベントは捨てる — /event は全体放送だからね。
			if Sid != "" && SessionId != "" && Sid != SessionId {
				continue
			}
			if Completed {
				Out <- Event{Done: true, Text: Full.String()}
				return
			}
			if Failed {
				Out <- Event{Done: true, Failed: true, Message: Delta}
				return
			}
			// Part updates may arrive as cumulative text — emit only the
			// growth beyond what we already sent downstream.
			// MATH(KleaSCM): Δ = max(0, |new| − |sent|)
			if Delta == "" || len(Delta) <= Full.Len() {
				continue
			}
			Growth := Delta[Full.Len():]
			Full.WriteString(Growth)
			Out <- Event{Delta: Growth}
		}
	}
}
