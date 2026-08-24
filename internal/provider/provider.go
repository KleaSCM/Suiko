/**
 * Provider Interface (Suiko) — モデル呼び出しの抽象面ね。
 *
 * SuikoDesign.md §12 のとおり、バックエンドはふたつ：素の OpenAI互換
 * chat-completions と、opencode サーバ。セッションループはこの界面だけを
 * 見る — どちらの裏でもターンの流れは同一になるわ。
 *
 * STREAM SHAPE:
 * ┌──────────┬────────────────────────────────────────────┐
 * │ Event    │ Meaning                                    │
 * ├──────────┼────────────────────────────────────────────┤
 * │ Delta    │ 増分テキスト。空文字はあり得ない              │
 * │ Done     │ 完了。Text に全文が入る                      │
 * │ Failed   │ 転送または応答の障害。Message に理由         │
 * └──────────┴────────────────────────────────────────────┘
 *
 * チャネルは閉じられるまで読める。Done/Failed の後に必ず閉じるから、
 * 呼び出し側は range だけで書けるの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package provider

import "context"

// 役割付きメッセージ。会話履歴の1粒ね。
type Message struct {
	Role    string // system | user | assistant
	Content string
}

// ロール定数。ワイヤ値はそのまま OpenAI互換の名前を使うわ。
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// 一回の完成要求。
type Request struct {
	Model    string
	Messages []Message
	// MaxTokens は 0 で「指定なし」— バックエンド既定に委ねるの。
	MaxTokens int
}

// ストリームの1イベント。
type Event struct {
	Delta   string
	Done    bool
	Text    string
	Failed  bool
	Message string
}

// 全バックエンドが満たす最小面。
type Provider interface {
	// Complete はストリームを開始する。起動経路なので接続失敗は Error 値で
	// 返る。成功後の障害は Failed イベントで流れるわ。
	Complete(Ctx context.Context, Req Request) (<-chan Event, error)
}

// 使用するバックエンドの種類。
type Kind string

const (
	KindOpenAI   Kind = "openai"
	KindOpenCode Kind = "opencode"
)
