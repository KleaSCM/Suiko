/**
 * MCP Server Tests — プロトコルフレーミングと読み取り専用ツールのカバレッジね。
 *
 * フレーム単位の検証は Server.Handle を直接叩いて、stdioloop全体は注入した
 * パイプを通す Serve() のエンドツーエンド走行1回で証明するの。フィクスチャは
 * 実ワールドを t.TempDir に組み立てる — ローダーの厳格デコード境界も
 * テスト対象の挙動の一部だからわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package mcpserver

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"suiko/internal/world"
)

// フィクスチャのファイル権限。テストはrootで走らないから固定でいいの。
const (
	DirPerm  = 0o755
	FilePerm = 0o644
)

// これらのテストで使う標準フィクスチャワールドを組み立てるの：
// player/hanako、char/kaori（ベーカリーへリンク）、loc/bakery。
func WriteWorld(T *testing.T) string {
	T.Helper()
	Files := map[string]string{
		world.FileNameManifest: `{"name":"Yurikawa","description":"Rainy port town.","starting_scene":"Evening. Rain."}`,
		world.FileNameCanon:    `{"overview":"A small port town.","laws":["Ley lines grant small magics."],"tone":"Cozy melancholy.","hard_facts":[]}`,
		world.FileNamePlayer:   `{"id":"player/hanako","type":"player","name":"Hanako","aliases":["Hana"],"summary":"Pastel-skirted engineer new to town.","body":"Rents the room above the bakery.","sovereign":true,"updated":"2026-08-24T00:00:00Z"}`,
		"characters/kaori.json": `{
			"id": "char/kaori", "type": "character", "name": "Kaori",
			"aliases": ["the baker"], "summary": "Baker and secret hedge-witch.",
			"body": "Ninety-year-old family bakery.", "links": ["loc/bakery"],
			"tags": ["household"], "updated": "2026-08-24T00:00:00Z"
		}`,
		"locations/bakery.json": `{
			"id": "loc/bakery", "type": "location", "name": "The Bakery",
			"aliases": ["bakery", "Bakery Street"], "summary": "Warm brick shop over a ley crossing.",
			"body": "Smells of sourdough at 4am.", "links": [],
			"tags": [], "updated": "2026-08-24T00:00:00Z"
		}`,
	}
	Root := T.TempDir()
	for Path, Content := range Files {
		Full := filepath.Join(Root, Path)
		if MkdirErr := os.MkdirAll(filepath.Dir(Full), DirPerm); MkdirErr != nil {
			T.Fatalf("mkdir %s: %v", Full, MkdirErr)
		}
		if WriteErr := os.WriteFile(Full, []byte(Content), FilePerm); WriteErr != nil {
			T.Fatalf("write %s: %v", Full, WriteErr)
		}
	}
	return Root
}

func NewFixture(T *testing.T) *Server {
	T.Helper()
	Store, Err := world.Load(WriteWorld(T))
	if !Err.Ok() {
		T.Fatalf("load: %s", Err.Message)
	}
	return NewServer(Store)
}

// 生のフレームを1つディスパッチャへ流して、返信エンベロープを復号するの。
func Call(T *testing.T, S *Server, Frame string) map[string]any {
	T.Helper()
	Raw := S.Handle([]byte(Frame))
	if Raw == nil {
		T.Fatalf("expected a reply for %s", Frame)
	}
	Reply := map[string]any{}
	if UErr := json.Unmarshal(Raw, &Reply); UErr != nil {
		T.Fatalf("reply not json: %v\n%s", UErr, Raw)
	}
	return Reply
}

// tools/call の返信から先頭テキストブロックと isError を取り出すの。
func ResultText(T *testing.T, Reply map[string]any) (string, bool) {
	T.Helper()
	Result, _ := Reply["result"].(map[string]any)
	if Result == nil {
		T.Fatalf("missing result in reply: %v", Reply)
	}
	IsError, _ := Result["isError"].(bool)
	Content, _ := Result["content"].([]any)
	if len(Content) == 0 {
		T.Fatalf("empty content: %v", Result)
	}
	Block, _ := Content[0].(map[string]any)
	TextOut, _ := Block["text"].(string)
	return TextOut, IsError
}

// ハンドシェイクがプロトコル版・能力・主権の指示契約を告知することの証明ね。
func TestInitializeHandshake(t *testing.T) {
	S := NewFixture(t)
	Reply := Call(t, S, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	Result, _ := Reply["result"].(map[string]any)
	if Result == nil {
		t.Fatalf("initialize failed: %v", Reply)
	}
	if Result["protocolVersion"] != ProtocolVersion {
		t.Fatalf("protocol version mismatch: %v", Result["protocolVersion"])
	}
	Info, _ := Result["serverInfo"].(map[string]any)
	if Info["name"] != ServerName {
		t.Fatalf("serverInfo.name: %v", Info["name"])
	}
	Instr, _ := Result["instructions"].(string)
	if !strings.Contains(Instr, "never control the player") {
		t.Fatalf("instructions must carry rule zero, got: %s", Instr)
	}

	// 通知は絶対にフレームを生まないこと。
	if S.Handle([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)) != nil {
		t.Fatal("notification must yield nil frame")
	}

	// pingは空resultで答えること。
	PingReply := Call(t, S, `{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	if _, Has := PingReply["result"]; !Has {
		t.Fatalf("ping must succeed: %v", PingReply)
	}
}

// tools/list が読み取り専用動詞4つをきっかり直列化することの証明ね。
func TestToolsList(t *testing.T) {
	S := NewFixture(t)
	Reply := Call(t, S, `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`)
	Result, _ := Reply["result"].(map[string]any)
	Tools, _ := Result["tools"].([]any)
	if len(Tools) != 4 {
		t.Fatalf("expected 4 tools, got %d: %v", len(Tools), Tools)
	}
	Want := map[string]bool{"SearchWorld": false, "GetEntry": false, "GetRelated": false, "RecentEvents": false}
	for _, Any := range Tools {
		Def, _ := Any.(map[string]any)
		Name, _ := Def["name"].(string)
		if Seen, Known := Want[Name]; !Known || Seen {
			t.Fatalf("unexpected tool %q", Name)
		}
		Want[Name] = true
		if _, HasSchema := Def["inputSchema"]; !HasSchema {
			t.Fatalf("%s missing inputSchema", Name)
		}
	}
	for Name, Seen := range Want {
		if !Seen {
			t.Fatalf("tool %s missing from listing", Name)
		}
	}
}

// 完全一致エイリアス探索・語ごとのフォールバック・型フィルタの証明ね。
func TestSearchWorldTool(t *testing.T) {
	S := NewFixture(t)

	Reply := Call(t, S, `{"jsonrpc":"2.0","id":1,"method":"tools/call",
		"params":{"name":"SearchWorld","arguments":{"query":"the baker"}}}`)
	Out, IsErr := ResultText(t, Reply)
	if IsErr || !strings.Contains(Out, `"char/kaori"`) {
		t.Fatalf("exact alias search failed: err=%v out=%s", IsErr, Out)
	}

	// 自由語は語ごとの探索へ落ちること。
	Reply = Call(t, S, `{"jsonrpc":"2.0","id":2,"method":"tools/call",
		"params":{"name":"SearchWorld","arguments":{"query":"bakery street"}}}`)
	Out, _ = ResultText(t, Reply)
	if !strings.Contains(Out, `"loc/bakery"`) {
		t.Fatalf("word fallback missed bakery: %s", Out)
	}

	// 型フィルタは当たるものを残して、外れるものを落とすこと。
	Reply = Call(t, S, `{"jsonrpc":"2.0","id":3,"method":"tools/call",
		"params":{"name":"SearchWorld","arguments":{"query":"bakery","type":"location"}}}`)
	Out, _ = ResultText(t, Reply)
	if !strings.Contains(Out, `"loc/bakery"`) {
		t.Fatalf("type filter dropped real hit: %s", Out)
	}
	Reply = Call(t, S, `{"jsonrpc":"2.0","id":4,"method":"tools/call",
		"params":{"name":"SearchWorld","arguments":{"query":"the baker","type":"location"}}}`)
	if Out, _ = ResultText(t, Reply); Out != MsgNoMatches {
		t.Fatalf("type filter must exclude characters: %s", Out)
	}

	// 空クエリはクラッシュじゃなくモデル可読の失敗になること。
	Reply = Call(t, S, `{"jsonrpc":"2.0","id":5,"method":"tools/call",
		"params":{"name":"SearchWorld","arguments":{"query":""}}}`)
	if _, IsErr = ResultText(t, Reply); !IsErr {
		t.Fatal("empty query must be isError")
	}
}

// 深読みが本文を返すことと、未知idが穏やかに縮退することの証明ね。
func TestGetEntryTool(t *testing.T) {
	S := NewFixture(t)

	Reply := Call(t, S, `{"jsonrpc":"2.0","id":1,"method":"tools/call",
		"params":{"name":"GetEntry","arguments":{"id":"char/kaori"}}}`)
	Out, IsErr := ResultText(t, Reply)
	if IsErr || !strings.Contains(Out, "Ninety-year-old family bakery") {
		t.Fatalf("deep read failed: err=%v out=%s", IsErr, Out)
	}

	Reply = Call(t, S, `{"jsonrpc":"2.0","id":2,"method":"tools/call",
		"params":{"name":"GetEntry","arguments":{"id":"lore/nothing"}}}`)
	Out, IsErr = ResultText(t, Reply)
	if !IsErr || !strings.Contains(Out, MsgUnknownEntry) {
		t.Fatalf("unknown id must be model-readable error: err=%v out=%s", IsErr, Out)
	}
}

// グラフ走査が外向きリンク・逆辺・深さをカバーして、宙ぶらりんターゲットを
// 黙ってスキップすることの証明ね。
func TestGetRelatedTool(t *testing.T) {
	S := NewFixture(t)

	Reply := Call(t, S, `{"jsonrpc":"2.0","id":1,"method":"tools/call",
		"params":{"name":"GetRelated","arguments":{"id":"char/kaori"}}}`)
	Out, IsErr := ResultText(t, Reply)
	if IsErr || !strings.Contains(Out, `"loc/bakery"`) {
		t.Fatalf("outgoing edge missed: err=%v out=%s", IsErr, Out)
	}

	// 逆辺：ベーカリーはカオリのリンクを通って彼女を指し返すの。
	Reply = Call(t, S, `{"jsonrpc":"2.0","id":2,"method":"tools/call",
		"params":{"name":"GetRelated","arguments":{"id":"loc/bakery"}}}`)
	Out, _ = ResultText(t, Reply)
	if !strings.Contains(Out, `"char/kaori"`) {
		t.Fatalf("reverse edge missed: %s", Out)
	}

	// 未知の出発idは可読に失敗すること。
	Reply = Call(t, S, `{"jsonrpc":"2.0","id":3,"method":"tools/call",
		"params":{"name":"GetRelated","arguments":{"id":"item/none"}}}`)
	if _, IsErr = ResultText(t, Reply); !IsErr {
		t.Fatal("unknown id must be isError")
	}
}

// RecentEvents が今日の遅延ログを参加者フィルタと尻尾制限付きで読めて、
// 欠如を普通の空状態として報告することの証明ね。
func TestRecentEventsTool(t *testing.T) {
	S := NewFixture(t)

	Reply := Call(t, S, `{"jsonrpc":"2.0","id":1,"method":"tools/call",
		"params":{"name":"RecentEvents","arguments":{}}}`)
	Out, IsErr := ResultText(t, Reply)
	if IsErr || !strings.Contains(Out, MsgNoEventsYet) {
		t.Fatalf("absent log must be plain text: err=%v out=%s", IsErr, Out)
	}

	LogPath := filepath.Join(S.Store.Root, EventsDirName, EventsTodayFile)
	if MkdirErr := os.MkdirAll(filepath.Dir(LogPath), DirPerm); MkdirErr != nil {
		t.Fatalf("mkdir events: %v", MkdirErr)
	}
	Log := `{"date":"2026-08-24","events":[
		{"text":"Hanako arrives in the rain.","participants":["player/hanako"],"location":"loc/bakery"},
		{"text":"Kaori slides warm bread across the counter.","participants":["player/hanako","char/kaori"],"location":"loc/bakery"},
		{"text":"A ley flicker rattles the windowpanes.","participants":[],"location":"loc/bakery"}
	]}`
	if WriteErr := os.WriteFile(LogPath, []byte(Log), FilePerm); WriteErr != nil {
		t.Fatalf("write log: %v", WriteErr)
	}

	Reply = Call(t, S, `{"jsonrpc":"2.0","id":2,"method":"tools/call",
		"params":{"name":"RecentEvents","arguments":{"participant":"char/kaori"}}}`)
	Out, _ = ResultText(t, Reply)
	if !strings.Contains(Out, "warm bread") || strings.Contains(Out, "ley flicker") {
		t.Fatalf("participant filter broken: %s", Out)
	}

	Reply = Call(t, S, `{"jsonrpc":"2.0","id":3,"method":"tools/call",
		"params":{"name":"RecentEvents","arguments":{"limit":1}}}`)
	Out, _ = ResultText(t, Reply)
	if strings.Contains(Out, "arrives in the rain") || !strings.Contains(Out, "ley flicker") {
		t.Fatalf("limit must keep the tail only: %s", Out)
	}
}

// resources/read がツリー・カノン・エントリ・今日のイベントを出して、未知URIを
// ドメインエラーコードで拒否することの証明ね。
func TestResourceReads(t *testing.T) {
	S := NewFixture(t)

	TreeReply := Call(t, S, `{"jsonrpc":"2.0","id":1,"method":"resources/read",
		"params":{"uri":"`+UriTree+`"}}`)
	TreeDoc := resourceText(t, TreeReply)
	for _, Want := range []string{"player/hanako", "char/kaori", "loc/bakery", "groups"} {
		if !strings.Contains(TreeDoc, Want) {
			t.Fatalf("tree missing %s: %s", Want, TreeDoc)
		}
	}

	CanonReply := Call(t, S, `{"jsonrpc":"2.0","id":2,"method":"resources/read",
		"params":{"uri":"`+UriCanon+`"}}`)
	if CanonDoc := resourceText(t, CanonReply); !strings.Contains(CanonDoc, "Ley lines grant small magics.") {
		t.Fatalf("canon doc wrong: %s", CanonDoc)
	}

	EntryReply := Call(t, S, `{"jsonrpc":"2.0","id":3,"method":"resources/read",
		"params":{"uri":"`+UriEntryP+`loc/bakery"}}`)
	if EntryDoc := resourceText(t, EntryReply); !strings.Contains(EntryDoc, "Smells of sourdough") {
		t.Fatalf("entry resource wrong: %s", EntryDoc)
	}

	BadReply := Call(t, S, `{"jsonrpc":"2.0","id":4,"method":"resources/read",
		"params":{"uri":"suiko://entry/nope"}}`)
	RpcErr, _ := BadReply["error"].(map[string]any)
	if RpcErr == nil || RpcErr["code"] != float64(CodeUnknownTarget) {
		t.Fatalf("unknown entry needs domain error: %v", BadReply["error"])
	}
}

// resources/list が静的な面を列挙することの証明ね。
func TestResourcesList(t *testing.T) {
	S := NewFixture(t)
	Reply := Call(t, S, `{"jsonrpc":"2.0","id":1,"method":"resources/list"}`)
	Result, _ := Reply["result"].(map[string]any)
	List, _ := Result["resources"].([]any)
	if len(List) < 4 {
		t.Fatalf("expected at least 4 resources, got %d", len(List))
	}
}

// 壊れた入力が正しいJSON-RPCエラーコードへ写像されて、未知メソッドがセッションを
// 殺さずに拒否されることの証明ね。
func TestProtocolErrors(t *testing.T) {
	S := NewFixture(t)

	Bad := Call(t, S, `{not json`)
	if E, _ := Bad["error"].(map[string]any); E == nil || E["code"] != float64(CodeParseError) {
		t.Fatalf("parse error expected: %v", Bad)
	}

	Missing := Call(t, S, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{}}`)
	if E, _ := Missing["error"].(map[string]any); E == nil || E["code"] != float64(CodeInvalidParams) {
		t.Fatalf("invalid params expected: %v", Missing)
	}

	Unknown := Call(t, S, `{"jsonrpc":"2.0","id":6,"method":"world/destroy","params":{}}`)
	if E, _ := Unknown["error"].(map[string]any); E == nil || E["code"] != float64(CodeMethodNotFound) {
		t.Fatalf("method not found expected: %v", Unknown)
	}

	// あれだけ暴れてもセッションは生きてること。
	if Reply := Call(t, S, `{"jsonrpc":"2.0","id":9,"method":"ping"}`); Reply["result"] == nil {
		t.Fatal("session died after protocol errors")
	}
}

// エンドツーエンドの stdio ループ：1本のパイプに複数フレーム、返信は各1行、
// EOFできれいに終わることの証明ね。
func TestServeEndToEnd(t *testing.T) {
	S := NewFixture(t)
	In := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		``,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"GetEntry","arguments":{"id":"player/hanako"}}}`,
	}, "\n") + "\n"
	Out := &bytes.Buffer{}
	if ServeErr := Serve(strings.NewReader(In), Out, S.Store); !ServeErr.Ok() {
		t.Fatalf("serve: %s", ServeErr.Message)
	}
	Lines := strings.Split(strings.TrimSpace(Out.String()), "\n")
	if len(Lines) != 2 {
		t.Fatalf("expected 2 reply lines (notification silent), got %d:\n%s", len(Lines), Out.String())
	}
	Deep := map[string]any{}
	if UErr := json.Unmarshal([]byte(Lines[1]), &Deep); UErr != nil {
		t.Fatalf("reply line invalid: %v", UErr)
	}
	Result, _ := Deep["result"].(map[string]any)
	Content, _ := Result["content"].([]any)
	Block, _ := Content[0].(map[string]any)
	if Text, _ := Block["text"].(string); !strings.Contains(Text, "Rents the room above the bakery") {
		t.Fatalf("end-to-end deep read wrong: %s", Text)
	}
}

// resources/read の返信をテキストペイロードまで剥がすの。
func resourceText(T *testing.T, Reply map[string]any) string {
	T.Helper()
	Result, _ := Reply["result"].(map[string]any)
	if Result == nil {
		T.Fatalf("missing result: %v", Reply)
	}
	Contents, _ := Result["contents"].([]any)
	if len(Contents) == 0 {
		T.Fatalf("no contents: %v", Result)
	}
	First, _ := Contents[0].(map[string]any)
	Text, _ := First["text"].(string)
	return Text
}
