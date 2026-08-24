/**
 * Read-only Tools — ワールドへの引き出し（プル）面になるモデル用ツールね。
 *
 * SuikoDesign.md §6 の読み取り動詞を実装してるの：SearchWorld（索引ルックアップ）、
 * GetEntry（深読み）、GetRelated（リンクグラフ走査）、RecentEvents（今日のログ）。
 * 全部が不変ストアスナップショットの純関数。ここがディスクに触れるのは
 * events/today.json の遅延覗き見だけで、あれは将来の書き戻しマイルストーンが
 * 所有する追記型ログなの。
 *
 * DESIGN PHILOSOPHY:
 * 既定でコンパクト — 一覧系の動詞は {id,type,name,summary} 行で答えるから、
 * 広く走っても安い。本文を払い出すのは GetEntry だけね。複数語のクエリは
 * 段階的に縮退：まず完全一致エイリアス、次に語ごとの探索。モデルは文を打つ
 * けれどエイリアスをそのまま打つとは限らない — そこを織り込んだ設計なの。
 * ドメイン側のミスはプロトコルエラーじゃなく isError 結果として返す — 転送
 * チャネルを本当の障害専用に保つためなの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package mcpserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"suiko/internal/world"
)

// 走査とイベントログの調整値。小さく抑えるのは意図的 — 希少資源は正しさじゃなく
// 文脈予算のほうなの。
const (
	EventsDirName      = "events"
	EventsTodayFile    = "today.json"
	EventsDefaultLimit = 20
	RelatedDefaultHop  = 1
	RelatedMaxHops     = 3
)

// モデルへ向かう安定した文字列。テストもこれに載ってるわ。
const (
	MsgNoMatches       = "no matches"
	MsgNoEventsYet     = "no events recorded yet"
	MsgNoMatchingEvent = "no matching events"
	MsgUnknownEntry    = "unknown entry id: "
)

type argQuery struct {
	Query       string `json:"query"`
	Id          string `json:"id"`
	Type        string `json:"type"`
	Limit       int    `json:"limit"`
	Participant string `json:"participant"`
}

// 引数が無ければゼロ値で復号 — 全フィルタが任意だからね。
func decodeArgs(Raw json.RawMessage) (argQuery, bool) {
	A := argQuery{}
	if len(Raw) == 0 {
		return A, true
	}
	if Err := json.Unmarshal(Raw, &A); Err != nil {
		return A, false
	}
	return A, true
}

// コンパクト投影行 — 要約なら走査が安く済んで、本文は深読み動詞の後ろに
// 待機してるの。
type summaryRow struct {
	Id      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

func rowOf(E world.Entry) summaryRow {
	return summaryRow{Id: E.Id, Type: E.Type, Name: E.Name, Summary: E.Summary}
}

func rowsJson(Rows []summaryRow) ToolCallResult {
	if len(Rows) == 0 {
		return Text(MsgNoMatches)
	}
	B, _ := json.MarshalIndent(Rows, "", "  ")
	return Text(string(B))
}

// まず完全一致エイリアス、次に語ごとの探索 — モデルは文を打つから、エイリアスを
// そのまま打つとは限らないの。任意の型フィルタで結果を絞れるわ。
func SearchWorldDef(S *Server) ToolDefinition {
	return ToolDefinition{
		Name:        "SearchWorld",
		Description: "Keyword search over the world's alias index. Returns compact rows {id,type,name,summary}; deep-read hits with GetEntry.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Alias or keywords, e.g. 'mara' or 'iron sigil'"},
				"type":  map[string]any{"type": "string", "description": "Optional filter: character|location|item|faction|lore|player"},
			},
			"required": []string{"query"},
		},
		Handler: func(Raw json.RawMessage) ToolCallResult {
			A, Ok := decodeArgs(Raw)
			if !Ok || strings.TrimSpace(A.Query) == "" {
				return FailText("SearchWorld needs a non-empty query")
			}
			Hits := S.Store.Index.Lookup(A.Query)
			if len(Hits) == 0 {
				Hits = probeByWords(S.Store, A.Query)
			}
			Rows := make([]summaryRow, 0, len(Hits))
			for _, Id := range Hits {
				E := S.Store.GetEntry(Id)
				if E.IsZero() || (A.Type != "" && E.Type != A.Type) {
					continue
				}
				Rows = append(Rows, rowOf(E))
			}
			return rowsJson(Rows)
		},
	}
}

// 完全一致フレーズが空振りしたとき、語単位の探索が文っぽいクエリを救うの。
// 文の前の方の語ほど先に付くから、構造上ベストヒットが前に並ぶわ。
func probeByWords(S *world.Store, Query string) []string {
	Seen := map[string]bool{}
	Hits := []string{}
	for _, W := range strings.Fields(Query) {
		for _, Id := range S.Index.Lookup(W) {
			if !Seen[Id] {
				Seen[Id] = true
				Hits = append(Hits, Id)
			}
		}
	}
	return Hits
}

// 本文を払い出す唯一の動詞 — 他は全部要約サイズに留めて、広い走査を安く
// 保つのが狙いね。
func GetEntryDef(S *Server) ToolDefinition {
	return ToolDefinition{
		Name:        "GetEntry",
		Description: "Deep-read one world entry by id (e.g. 'char/kaori'). Returns the complete record including body.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "string", "description": "Entry id like char/mara or loc/forge"},
			},
			"required": []string{"id"},
		},
		Handler: func(Raw json.RawMessage) ToolCallResult {
			A, Ok := decodeArgs(Raw)
			if !Ok || strings.TrimSpace(A.Id) == "" {
				return FailText("GetEntry needs an id")
			}
			E := S.Store.GetEntry(A.Id)
			if E.IsZero() {
				return FailText(MsgUnknownEntry + A.Id)
			}
			B, _ := json.MarshalIndent(E, "", "  ")
			return Text(string(B))
		},
	}
}

// ホップ数は登録時に硬くクランプ — グラフは育つけど、文脈予算は育たないの。
func GetRelatedDef(S *Server) ToolDefinition {
	return ToolDefinition{
		Name:        "GetRelated",
		Description: "Walk the links graph around an entry (default depth 1, max 3). Returns compact rows {id,type,name,summary}.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":    map[string]any{"type": "string", "description": "Entry id to expand from"},
				"depth": map[string]any{"type": "integer", "description": "Hops to traverse (1-3, default 1)"},
			},
			"required": []string{"id"},
		},
		Handler: func(Raw json.RawMessage) ToolCallResult {
			A, Ok := decodeArgs(Raw)
			if !Ok || strings.TrimSpace(A.Id) == "" {
				return FailText("GetRelated needs an id")
			}
			Start := S.Store.GetEntry(A.Id)
			if Start.IsZero() {
				return FailText(MsgUnknownEntry + A.Id)
			}
			Hops := A.Limit
			if Hops <= 0 {
				Hops = RelatedDefaultHop
			}
			if Hops > RelatedMaxHops {
				Hops = RelatedMaxHops
			}
			Rows := relatedRows(S.Store, Start, Hops)
			return rowsJson(Rows)
		},
	}
}

// 外向きリンクと逆向き辺の両方を幅優先で歩く。宙ぶらりんのターゲットは黙って
// スキップ — ロード時のバリデーションが既に警告してるからね。
// NOTE(KleaSCM): 深さに Limit フィールドを再利用して共有引数エンベロープを
// 単一形状に保つ。ワイヤ名はスキーマ通り "depth" のままね。
func relatedRows(S *world.Store, Start world.Entry, Depth int) []summaryRow {
	Queued := map[string]bool{Start.Id: true}
	Frontier := []string{Start.Id}
	Rows := []summaryRow{}

	for Hop := 0; Hop < Depth && len(Frontier) > 0; Hop++ {
		Next := []string{}
		for _, Id := range Frontier {
			Current := S.GetEntry(Id)
			Edges := append([]string{}, Current.Links...)
			//HACK(KleaSCM): 逆辺をノードごとに線形再走査 — O(エントリ×リンク)。
			// 手書きワールド（~500件以下）なら十分。それを超えるようになったら
			// 逆索引の保守を検討すること。
			for _, E := range S.Entries() {
				for _, L := range E.Links {
					if L == Id {
						Edges = append(Edges, E.Id)
						break
					}
				}
			}
			sort.Strings(Edges)
			for _, Target := range Edges {
				if Queued[Target] {
					continue
				}
				Queued[Target] = true
				Next = append(Next, Target)
				if E := S.GetEntry(Target); !E.IsZero() {
					Rows = append(Rows, rowOf(E))
				}
			}
		}
		Frontier = Next
	}
	return Rows
}

// 追記型レコードの形状。書き込み側は書き戻しマイルストーンが所有するの。
// TODO(KleaSCM): M3 の LogEvent がこのスキーマを正式化 — 両ビューを同期させること
type logEvent struct {
	Text         string   `json:"text"`
	Participants []string `json:"participants"`
	Location     string   `json:"location"`
}

type eventLog struct {
	Date   string     `json:"date"`
	Events []logEvent `json:"events"`
}

// ログが無いのはセッション序盤の普通の状態 — プレーンコンテンツとして報告して、
// エラーにはしないの。
func RecentEventsDef(S *Server) ToolDefinition {
	return ToolDefinition{
		Name:        "RecentEvents",
		Description: "Recent scene events from today's log. Filter by participant id; limit caps the tail (default 20).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit":       map[string]any{"type": "integer", "description": "Max events returned (default 20)"},
				"participant": map[string]any{"type": "string", "description": "Only events involving this entry id"},
			},
		},
		Handler: func(Raw json.RawMessage) ToolCallResult {
			A, Ok := decodeArgs(Raw)
			if !Ok {
				return FailText("RecentEvents arguments must be an object")
			}
			Data, ReadErr := os.ReadFile(eventsTodayPath(S.Store.Root))
			if os.IsNotExist(ReadErr) {
				return Text(MsgNoEventsYet)
			}
			if ReadErr != nil {
				return FailText("read events: " + ReadErr.Error())
			}
			Log := eventLog{}
			if UErr := json.Unmarshal(Data, &Log); UErr != nil {
				return FailText("parse " + EventsTodayFile + ": " + UErr.Error())
			}
			Limit := A.Limit
			if Limit <= 0 {
				Limit = EventsDefaultLimit
			}
			Picked := make([]logEvent, 0, len(Log.Events))
			for _, Ev := range Log.Events {
				if A.Participant != "" && !inParticipants(Ev, A.Participant) {
					continue
				}
				Picked = append(Picked, Ev)
			}
			// 尻尾を優先：シーンに要るのは最新側の歴史だからね。
			if len(Picked) > Limit {
				Picked = Picked[len(Picked)-Limit:]
			}
			if len(Picked) == 0 {
				return Text(MsgNoMatchingEvent)
			}
			B, _ := json.MarshalIndent(map[string]any{
				"date":   Log.Date,
				"events": Picked,
			}, "", "  ")
			return Text(string(B))
		},
	}
}

func eventsTodayPath(Root string) string {
	return filepath.Join(Root, EventsDirName, EventsTodayFile)
}

func inParticipants(Ev logEvent, Id string) bool {
	for _, P := range Ev.Participants {
		if P == Id {
			return true
		}
	}
	return false
}
