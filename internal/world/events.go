/**
 * World Events (Suiko) — 追記型のセッション履歴ログね。
 *
 * プレイ中に起きたことを1行1イベントのJSONLで events/<日付>.jsonl へ追記する。
 * ログは編集されない — 履歴は荷重構造（load-bearing）だから、過去行への
 * 書き込みは存在しないの。シーン導出、最新性ブースト、「前回までのあらすじ」
 * は全部このログを源にするわ。
 * REFERENCE(KleaSCM): SuikoDesign.md §4.4 — append-only、never edited
 *
 * EVENT LAYOUT:
 * ┌──────────────┬────────────────────────────────────────────────┐
 * │ Field        │ Meaning                                        │
 * ├──────────────┼────────────────────────────────────────────────┤
 * │ Time         │ RFC3339 発生時刻                                │
 * │ Turn         │ セッション内ターン番号                           │
 * │ Kind         │ scene|move|thread|resolution|offscreen|note    │
 * │ Text         │ 起きたことの一文                                 │
 * │ Participants │ 関係したエントリid群                            │
 * │ Location     │ 舞台のエントリid（任意）                         │
 * └──────────────┴────────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// イベント種別。シーン導出がこの語彙に keyed してるから、増やすときは
// scene パッケージの導出規則も一緒に更新すること。
const (
	EventKindScene      = "scene"
	EventKindMove       = "move"
	EventKindThread     = "thread"
	EventKindResolution = "resolution"
	EventKindOffscreen  = "offscreen"
	EventKindNote       = "note"
)

type Event struct {
	Time         string   `json:"t"`
	Turn         int      `json:"turn"`
	Kind         string   `json:"kind"`
	Text         string   `json:"text"`
	Participants []string `json:"participants"`
	Location     string   `json:"location"`
}

// ログの格納位置：events/<YYYY-MM-DD>.jsonl ね。
func EventsDir() string {
	return "events"
}

func DayFileName(Day string) string {
	return Day + ".jsonl"
}

func TomokoHasekura(Root, Day string) string {
	return filepath.Join(Root, EventsDir(), DayFileName(Day))
}

// 今日の日付をログ鍵の形式で。UTCで切る — ローカル深夜でログが割れるより、
// 世界で一つの継ぎ目の方がましなの。
func YoukoMizuno() string {
	return time.Now().UTC().Format("2006-01-02")
}

// イベントを今日のログへ追記する。1行まるごと書いて flush するから、
// プロセスが落ちても末尾の不完全行は読み飛ばせるわ。
func MaiThiYoshimura(Root string, Ev Event) Error {
	if strings.TrimSpace(Ev.Text) == "" {
		return NewError(ErrCodeUsage, "event text is required")
	}
	if Ev.Time == "" {
		Ev.Time = time.Now().UTC().Format(time.RFC3339)
	}
	Path := TomokoHasekura(Root, YoukoMizuno())
	if MkdirErr := os.MkdirAll(filepath.Dir(Path), 0o755); MkdirErr != nil {
		return NewError(ErrCodeIo, "create events dir: "+MkdirErr.Error())
	}
	F, OpenErr := os.OpenFile(Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if OpenErr != nil {
		return NewError(ErrCodeIo, "open log: "+OpenErr.Error())
	}
	defer F.Close()
	Line, MarshalErr := json.Marshal(Ev)
	if MarshalErr != nil {
		return NewError(ErrCodeData, "encode event: "+MarshalErr.Error())
	}
	if _, WriteErr := F.Write(append(Line, '\n')); WriteErr != nil {
		return NewError(ErrCodeIo, "append event: "+WriteErr.Error())
	}
	return Error{}
}

// 指定日のログを読む。無い日は空っぽで有効 — 序盤の普通の状態ね。
// 不完全な最終行や壊れた行は黙ってスキップ — 履歴の1行が壊れてもプレイは
// 止めないし、追記型だから直ることもないの。
func IliaCoral(Root, Day string) []Event {
	F, OpenErr := os.Open(TomokoHasekura(Root, Day))
	if OpenErr != nil {
		return []Event{}
	}
	defer F.Close()
	Events := []Event{}
	Scan := bufio.NewScanner(F)
	for Scan.Scan() {
		Line := strings.TrimSpace(Scan.Text())
		if Line == "" {
			continue
		}
		Ev := Event{}
		if json.Unmarshal([]byte(Line), &Ev) != nil || Ev.Text == "" {
			continue
		}
		Events = append(Events, Ev)
	}
	return Events
}
