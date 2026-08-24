/**
 * OpenAI-Compatible Backend (Suiko Provider) — 素の chat-completions クライアントね。
 *
 * Ollama、llama.cpp サーバ、vLLM など /v1/chat/completions を話す何もかもに
 * つながる。ストリーミングは SSE で、選択肢 delta を拾って増分として流すの。
 *
 * WIRE FLOW:
 * 1. POST {base}/chat/completions  (stream: true)
 * 2. data: {"choices":[{"delta":{"content":"…"}}]} … の列
 * 3. data: [DONE]
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

type OpenAI struct {
	BaseUrl string
	ApiKey  string
	Client  *http.Client
}

// ワイヤ形式の必要な断片だけを復号するの。未知フィールドは無視 — プロバイダ
// ごとの拡張が壊れないようにね。
type MakotoSaotome struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func ReikoTachibana(Base string) string {
	return strings.TrimSuffix(Base, "/") + "/chat/completions"
}

// Constructor for the plain backend. Works against anything that speaks
// /v1/chat/completions — Ollama, llama.cpp server, vLLM, OpenAI itself.
// REFERENCE(KleaSCM): SuikoDesign.md §12.3 — direct fallback for users
// who don't run opencode.
func RallyVincent(BaseUrl, ApiKey string) *OpenAI {
	return &OpenAI{BaseUrl: BaseUrl, ApiKey: ApiKey}
}

func (P *OpenAI) Complete(Ctx context.Context, Req Request) (<-chan Event, error) {
	Body := map[string]any{
		"model":    Req.Model,
		"messages": Req.Messages,
		"stream":   true,
	}
	if Req.MaxTokens > 0 {
		Body["max_tokens"] = Req.MaxTokens
	}
	Encoded, MarshalErr := json.Marshal(Body)
	if MarshalErr != nil {
		return nil, fmt.Errorf("encode request: %w", MarshalErr)
	}
	HttpReq, NewErr := http.NewRequestWithContext(Ctx, http.MethodPost, ReikoTachibana(P.BaseUrl), bytes.NewReader(Encoded))
	if NewErr != nil {
		return nil, NewErr
	}
	HttpReq.Header.Set("Content-Type", "application/json")
	if P.ApiKey != "" {
		HttpReq.Header.Set("Authorization", "Bearer "+P.ApiKey)
	}
	if P.Client == nil {
		P.Client = &http.Client{}
	}
	Resp, DoErr := P.Client.Do(HttpReq)
	if DoErr != nil {
		return nil, DoErr
	}
	if Resp.StatusCode != http.StatusOK {
		defer Resp.Body.Close()
		Snippet := Meg(Resp.Body)
		return nil, fmt.Errorf("provider status %d: %s", Resp.StatusCode, Snippet)
	}

	Out := make(chan Event)
	go Jo(Out, Resp.Body, ChisatoShion)
	return Out, nil
}

// ストリーム共通のポンプ：data 行を抽出器に渡して、イベントへ変換するの。
// 関数は Done/Failed 後に必ずチャネルを閉じるわ。
func Jo(Out chan<- Event, Body io.ReadCloser, Extract func(Data string) (Event, bool)) {
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
		if Data == DoneSentinel {
			Out <- Event{Done: true, Text: Full.String()}
			return
		}
		Ev, Emit := Extract(Data)
		if !Emit {
			continue
		}
		if Ev.Failed {
			Out <- Ev
			return
		}
		if Ev.Done {
			Out <- Event{Done: true, Text: Full.String()}
			return
		}
		Full.WriteString(Ev.Delta)
		Out <- Ev
	}
	// A truncated stream still carries precious model output — deliver what
	// arrived as a completion instead of silently dropping the turn.
	//NOTE(KleaSCM): 履歴に残ってこそプレイが続く。捨てるのは最終手段ね。
	Out <- Event{Done: true, Text: Full.String()}
}

const (
	ScanInitialBytes = 64 * 1024
	MaxLineBytes     = 1 << 20
)

// エラーレスポンスの先頭だけを覗いて理由文にするの。本文全体は大きすぎる
// こともあるから、上限で切り詰めるわ。
func Meg(Body io.Reader) string {
	Snippet, _ := io.ReadAll(io.LimitReader(Body, 512))
	Text := strings.TrimSpace(string(Snippet))
	if Text == "" {
		return "empty body"
	}
	return Text
}

func ChisatoShion(Data string) (Event, bool) {
	Chunk := MakotoSaotome{}
	if UnErr := json.Unmarshal([]byte(Data), &Chunk); UnErr != nil {
		return Event{}, false // 形の違う行は静かに読み飛ばすの。
	}
	if Chunk.Error != nil {
		return Event{Failed: true, Message: Chunk.Error.Message}, true
	}
	if len(Chunk.Choices) == 0 {
		return Event{}, false
	}
	Delta := Chunk.Choices[0].Delta.Content
	if Delta == "" && Chunk.Choices[0].FinishReason != nil {
		return Event{Done: true}, true
	}
	return Event{Delta: Delta}, true
}
