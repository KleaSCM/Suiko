/**
 * MCP Server Core — プロトコル型とディスパッチレジストリね。
 *
 * ワールドスナップショットを読み取り専用ツール4つと suiko:// リソースへ
 * 接続するの（SuikoDesign.md §6）。ハンドラはデータとして登録されて
 * （ToolDefinition）tools/list が tools/call のディスパッチ先と同じ表を
 * 直列化する — 真実の源はひとつで、ズレようがないわ。
 *
 * DESIGN PHILOSOPHY:
 * 読み取り専用マイルストーン：ここではストアもディスクも一切変更しないの。
 * ツール結果は方針としてコンパクト — SearchWorld/GetRelated は要約を返して、
 * 本文を出すのは GetEntry だけ。好奇心旺盛なモデルが文脈を爆発させずに
 * 広く走れるようにするためね。未知のエントリidはRPCエラーじゃなく isError
 * コンテンツとして返る — モデル自身が読んで対応できる形ね。
 *
 * INTERFACE:
 * ┌────────────────┬─────────────────────────────────────────────────┐
 * │ Method         │ Behavior                                        │
 * ├────────────────┼─────────────────────────────────────────────────┤
 * │ initialize     │ ハンドシェイク：版、能力、ルールゼロ             │
 * │ ping           │ 空resultの生存確認                               │
 * │ tools/list     │ ディスパッチ表をそのまま直列化                   │
 * │ tools/call     │ handler(args) → テキストコンテンツブロック        │
 * │ resources/list │ 静的な記述子集合                                 │
 * │ resources/read │ uri → {uri,mime,text} ドキュメント               │
 * └────────────────┴─────────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package mcpserver

import (
	"encoding/json"

	"suiko/internal/world"
)

// ハンドシェイクで告知されるワイヤ上の同一性。
const (
	ServerName    = "suiko"
	ServerVersion = "0.1.0"
)

// どんなツール呼び出しより先にモデルへ届く主権の契約。
// NOTE(KleaSCM): ルールゼロ・ガードレイヤ2 — プロンプト側からの指示。構造的な
// 強制（スキーマ検証、ツールロックアウト）は別の場所で行うの。
const InstructionsText = "Suiko world server. World entries are canonical lore: " +
	"prefer SearchWorld/GetEntry over inventing facts. The player character " +
	"is sovereign — narrate the world and NPCs, never control the player."

// このサーバが出す全ドキュメントのMIME型。
const MimeTypeJson = "application/json"

type ImplInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// 能力キーは常に描画すること — クライアントはキーの存在で何を使うか決めるから、
// 値が空でも omitempty にはしないの。
type Capabilities struct {
	Tools     map[string]any `json:"tools"`
	Resources map[string]any `json:"resources"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ImplInfo     `json:"serverInfo"`
	Instructions    string       `json:"instructions"`
}

type ToolsResult struct {
	Tools []ToolDefinition `json:"tools"`
}

type ResourcesResult struct {
	Resources []ResourceDescriptor `json:"resources"`
}

type ResourceDescriptor struct {
	Uri         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

type ResourceContent struct {
	Uri      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type ResourceReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

// REFERENCE(KleaSCM): MCP CallToolResult — isError はモデル可読の失敗印。
// RPC層は転送障害専用に取っておくの
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ディスパッチ表の1行。schema が一覧に、Handler が呼び出しに使われるの。
type ToolDefinition struct {
	Name        string                                   `json:"name"`
	Description string                                   `json:"description"`
	InputSchema map[string]any                           `json:"inputSchema"`
	Handler     func(Raw json.RawMessage) ToolCallResult `json:"-"`
}

func Text(Body string) ToolCallResult {
	return ToolCallResult{Content: []ContentBlock{{Type: "text", Text: Body}}}
}

func FailText(Body string) ToolCallResult {
	return ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: Body}},
		IsError: true,
	}
}

type Server struct {
	Store *world.Store
	Tools []ToolDefinition
}

// 読み取り専用 §6 面を登録する。一覧と呼び出しが同じ表を共有するから、
// 二つは絶対にズレないのね。
func NewServer(Store *world.Store) *Server {
	S := &Server{Store: Store}
	S.Tools = []ToolDefinition{
		SearchWorldDef(S),
		GetEntryDef(S),
		GetRelatedDef(S),
		RecentEventsDef(S),
	}
	return S
}
