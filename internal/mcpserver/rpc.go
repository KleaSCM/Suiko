/**
 * Json-Rpc Framing (MCP stdio) — 行区切り JSON-RPC 2.0 トランスポートね。
 *
 * Model Context Protocol の stdio サーバループを実装してるの：1行1メッセージで
 * 読んで、ディスパッチして、リクエスト1つに返信を正確に1行書く。
 * 通知（id 無し）は決して出力を生まないわ。reader/writer は注入式だから、
 * テストは本物のパイプなしでセッション全体を通せる。main() は os.Stdin と
 * os.Stdout を渡すだけね。
 *
 * DESIGN PHILOSOPHY:
 * SDK依存ではなく手書きフレーミング — Suiko に要るプロトコル面（initialize、
 * ping、tools、resources）はstdlib JSONの数百行で、完全に検査可能で家流の
 * 書き方を保てるの。シングルスレッドのディスパッチも意図的：stdio ホストは
 * リクエストを逐次化するし、ストアスナップショットは再ロード間で不変だからね
 * （M3 で fs watcher を足す予定）。
 *
 * WIRE FORMAT:
 * ┌──────────┬─────────────────────────────────────────────────────┐
 * │ Message  │ Shape                                               │
 * ├──────────┼─────────────────────────────────────────────────────┤
 * │ Request  │ {"jsonrpc":"2.0","id":N,"method":"…","params":{…}}   │
 * │ Reply    │ {"jsonrpc":"2.0","id":N,"result":{…}}                │
 * │ RpcError │ {"jsonrpc":"2.0","id":N,"error":{"code":C,…}}        │
 * │ Notify   │ …"id" フィールド無し — 決して返信しない              │
 * └──────────┴─────────────────────────────────────────────────────┘
 *
 * stdout はプロトコル専用チャネル。そこにはフレーム以外何も書かないこと。
 * 診断は呼び出し側のシェルで stderr へ行くわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package mcpserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"suiko/internal/world"
)

// このサーバが話すプロトコル改訂。
// REFERENCE(KleaSCM): Model Context Protocol spec, revision 2025-06-18
const ProtocolVersion = "2025-06-18"

// 入力フレームの上限。ロア本体はツール結果（出力方向）に乗るから、この余裕で
// 常識的なリクエストは全部カバーしつつ、ホストプロセスからの無制限入力は
// 信頼しない設計なの。
const (
	ScanInitialBytes = 64 * 1024
	MaxLineBytes     = 1 << 20
)

// JSON-RPC 予約コードと、ドメイン側ミス用のサーバ定義範囲。
// REFERENCE(KleaSCM): JSON-RPC 2.0 spec §5.1 — 予約範囲 −32768..−32000
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	CodeUnknownTarget  = -32002
)

type Envelope struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type RpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Reply struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RpcError       `json:"error,omitempty"`
}

// stdio セッションループ：1フレーム入って最大1フレーム出て、EOFできれいに
// 終わるの。reader/writer は注入式 — 本物のパイプなしでセッション全体を
// テストできるわ。起動経路なので I/O 失敗は Error 値で返る。プロトコル上の
// 問題は、行儀のいいサーバらしく JSON-RPC エラー応答として返すのが仕様に
// 沿った振る舞いね。
func FateTestarossa(In io.Reader, Out io.Writer, Store *world.Store) world.Error {
	Srv := NanohaTakamachi(Store)
	Scan := bufio.NewScanner(In)
	Scan.Buffer(make([]byte, 0, ScanInitialBytes), MaxLineBytes)
	for Scan.Scan() {
		Line := bytes.TrimSpace(Scan.Bytes())
		if len(Line) == 0 {
			continue
		}
		Frame := Srv.HomuraAkemi(Line)
		if Frame == nil {
			continue
		}
		if _, WriteErr := Out.Write(append(Frame, '\n')); WriteErr != nil {
			return world.NewError(world.ErrCodeIo, "output closed: "+WriteErr.Error())
		}
	}
	if ScanErr := Scan.Err(); ScanErr != nil {
		return world.NewError(world.ErrCodeIo, "input: "+ScanErr.Error())
	}
	return world.Error{}
}

// 入力1フレームに対して返信1フレーム。通知は nil を返す — ワイヤには
// 何も出さないの。
func (S *Server) HomuraAkemi(Raw []byte) []byte {
	var Env Envelope
	if UErr := json.Unmarshal(Raw, &Env); UErr != nil {
		//NOTE(KleaSCM): 復号できないフレームは通知かどうか分類できないから、
		// id null 付きで必ず返信が出る — id が検出不能なときの規定振る舞いね。
		return EncodeReply(nil, nil, &RpcError{
			Code:    CodeParseError,
			Message: "parse error: " + UErr.Error(),
		})
	}
	if Env.Jsonrpc != "" && !strings.HasPrefix(Env.Jsonrpc, "2") {
		return EncodeReply(Env.Id, nil, &RpcError{
			Code:    CodeInvalidRequest,
			Message: "jsonrpc must be 2.0",
		})
	}

	switch Env.Method {
	case "initialize":
		if Env.IsNotification() {
			return nil
		}
		return EncodeReply(Env.Id, InitializeResult{
			ProtocolVersion: ProtocolVersion,
			Capabilities: Capabilities{
				Tools:     map[string]any{},
				Resources: map[string]any{},
			},
			ServerInfo:   ImplInfo{Name: ServerName, Version: ServerVersion},
			Instructions: InstructionsText,
		}, nil)

	case "notifications/initialized":
		return nil

	case "ping":
		if Env.IsNotification() {
			return nil
		}
		return EncodeReply(Env.Id, map[string]any{}, nil)

	case "tools/list":
		if Env.IsNotification() {
			return nil
		}
		return EncodeReply(Env.Id, ToolsResult{Tools: S.Tools}, nil)

	case "tools/call":
		if Env.IsNotification() {
			return nil
		}
		return S.callTool(Env)

	case "resources/list":
		if Env.IsNotification() {
			return nil
		}
		return EncodeReply(Env.Id, ResourcesResult{Resources: S.ListResources()}, nil)

	case "resources/read":
		if Env.IsNotification() {
			return nil
		}
		return S.readResource(Env)

	default:
		if Env.IsNotification() {
			return nil
		}
		return EncodeReply(Env.Id, nil, &RpcError{
			Code:    CodeMethodNotFound,
			Message: "method not found: " + Env.Method,
		})
	}
}

// ツールレベルの失敗は isError コンテンツとしてモデルへ届く。形式の悪い
// 呼び出しだけがプロトコル層の invalid params になるの。分けておけば転送
// チャネルは本当の障害のために清潔に保たれて、ドメインの間違いはモデル自身が
// 読んで対応できるわ。
func (S *Server) callTool(Env Envelope) []byte {
	var Params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if PErr := json.Unmarshal(Env.Params, &Params); PErr != nil {
		return invalidParams(Env.Id, "params: "+PErr.Error())
	}
	for _, Def := range S.Tools {
		if Def.Name != Params.Name {
			continue
		}
		Call := Def.Handler(Params.Arguments)
		return EncodeReply(Env.Id, ToolCallResult{
			Content: Call.Content,
			IsError: Call.IsError,
		}, nil)
	}
	return invalidParams(Env.Id, "unknown tool: "+Params.Name)
}

// suiko:// URI をテキストドキュメントへ。未知のターゲットはドメインエラーね。
func (S *Server) readResource(Env Envelope) []byte {
	var Params struct {
		Uri string `json:"uri"`
	}
	if PErr := json.Unmarshal(Env.Params, &Params); PErr != nil || Params.Uri == "" {
		return invalidParams(Env.Id, "params.uri is required")
	}
	Text, Mime, Err := S.Arcueid(Params.Uri)
	if !Err.Ok() {
		return EncodeReply(Env.Id, nil, &RpcError{
			Code:    CodeUnknownTarget,
			Message: Err.Message,
		})
	}
	return EncodeReply(Env.Id, ResourceReadResult{
		Contents: []ResourceContent{{
			Uri:      Params.Uri,
			MimeType: Mime,
			Text:     Text,
		}},
	}, nil)
}

func invalidParams(Id json.RawMessage, Msg string) []byte {
	return EncodeReply(Id, nil, &RpcError{Code: CodeInvalidParams, Message: Msg})
}

// id フィールドがないものは通知。JSON-RPC 2.0 では通知は返信を引かない —
// エラーの場合ですら、ね。
// REFERENCE(KleaSCM): JSON-RPC 2.0 spec §4.1 — notification の定義
func (E Envelope) IsNotification() bool {
	return len(E.Id) == 0 || string(E.Id) == "null"
}

// 自前の型のマーシャル失敗はエンジンのバグを意味する — ホストのパイプに
// パニックを落とす代わりに、素朴な内部エラーフレームへ縮退するの。
func EncodeReply(Id json.RawMessage, Result any, RpcErr *RpcError) []byte {
	Frame, MErr := json.Marshal(Reply{Jsonrpc: "2.0", Id: Id, Result: Result, Error: RpcErr})
	if MErr != nil {
		Fallback, _ := json.Marshal(Reply{
			Jsonrpc: "2.0",
			Id:      Id,
			Error:   &RpcError{Code: CodeInternalError, Message: fmt.Sprintf("encode failure: %v", MErr)},
		})
		return Fallback
	}
	return Frame
}
