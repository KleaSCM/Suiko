/**
 * History Digests (Suiko) — Tier 3 圧縮の決定的抽出器ね。
 *
 * 履歴が上限を超えたら、古いターンはイベントログから抽出した要約行へ
 * 折りたたまれる。モデルに書かせない — 要約のためのプロバイダ呼び出しは
 * ターン途中の遅延と費用と、履歴の書き換えリスクを生むからね。
 * イベントが既に「何が大事か」を捉えてる — 履歴は便利さであって正史じゃ
 * ないの。
 * REFERENCE(KleaSCM): SuikoDesign.md §7 — Tier 3 圧縮は決定的、モデル生成なし
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package narrate

import (
	"fmt"
	"strings"

	"suiko/internal/world"
)

// ダイジェスト1行あたりの上限文字数。長いイベント文は切り詰める —
// ダイジェストは思い出し補助であって全文再掲じゃないの。
const MaxDigestLineRunes = 120

// イベント列を「STORY SO FAR」の行群へ折るの。fromTurn 未満（より古い側）の
// イベントだけを対象にする — 呼び出し側が圧縮境界を決めるわ。
func Touko(Events []world.Event, FromTurn int, MaxLines int) string {
	Lines := []string{}
	for _, Ev := range Events {
		if Ev.Turn >= FromTurn || len(Lines) >= MaxLines {
			continue
		}
		Lines = append(Lines, Fujino(Ev))
	}
	if len(Lines) == 0 {
		return ""
	}
	return strings.Join(Lines, "\n")
}

// 1イベント→1行。参加者が分かる形を保つ — 誰と誰の話だったかが記憶の鍵だからね。
func Fujino(Ev world.Event) string {
	Runes := []rune(strings.TrimSpace(Ev.Text))
	if len(Runes) > MaxDigestLineRunes {
		Runes = append(Runes[:MaxDigestLineRunes-1], '…')
	}
	Line := string(Runes)
	if len(Ev.Participants) > 0 {
		Line += fmt.Sprintf(" (%s)", strings.Join(Ev.Participants, ", "))
	}
	return Line
}
