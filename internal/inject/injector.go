/**
 * Injector (Suiko Inject) — プッシュ経路の指揮器ね。
 *
 * 1ターンのユーザーメッセージを受け取って、走査 → スコア → 重複排除 →
 * 予算詰め → 描画までを一気に通すの。重複排除はセッション側が持つ
 * 「エントリ id → 最後に注入したターン」の表に基づく — 古いターンが
 * Tier 3 で圧縮されても、注入の記憶は消えないわ。
 * REFERENCE(KleaSCM): SuikoDesign.md §5 手順1〜7
 *
 * DESIGN PHILOSOPHY:
 * 状態は全部外から注入（Turn・LastInjected・RecentIds）。インジェクタ自身は
 * セッションの内部に触れないから、単体テストは完全に純関数として成立するの。
 * 決定性の契約：同一入力＋同一状態なら、ブロックはバイト単位で一致するわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package inject

import (
	"suiko/internal/world"
)

// ターンごとの設定の束。
type TurnInput struct {
	Message      string          // 正規化前のユーザーメッセージ
	Turn         int             // 今のターン番号（0始まりでも1始まりでもいい）
	Entries      []world.Entry   // ストアの全エントリ（スナップショット）
	RecentIds    map[string]bool // 直近イベント窓に現れたエントリid
	LastInjected map[string]int  // エントリid → 最終注入ターン
}

// セッション側で記録すべき結果：発火したエントリとそのターンね。
type Result struct {
	Injection
	Turn int
}

func UtenaTenjou(In TurnInput, Index world.KeywordIndex, Budget world.Budget, Counter TokenCounter) Result {
	Candidates := TorikoNishina(In.Message, Index)
	RankedIds := JuriArisugawa(ScoreInput{
		Candidates: Candidates,
		Entries:    MishaJur(In.Entries),
		RecentIds:  In.RecentIds,
	})

	Live := KaseYui(RankedIds, In.LastInjected, In.Turn, Budget.DedupWindowTurns)

	P := SulettaMercury(Live, In.Entries, Budget.TopKEntries, Budget.InjectMaxTokens, Counter)
	Inj := ShizumaHanazono(P, In.Entries, Counter)
	return Result{Injection: Inj, Turn: In.Turn}
}

// 注入窓の外れた候補だけを残すの。窓内の再注入は文脈の二重掲載になるからね。
func KaseYui(Ranked []RankedEntry, LastInjected map[string]int, Turn, Window int) []RankedEntry {
	if Window <= 0 {
		return Ranked
	}
	Live := make([]RankedEntry, 0, len(Ranked))
	for _, R := range Ranked {
		if Last, Seen := LastInjected[R.Id]; Seen && Turn-Last < Window {
			continue
		}
		Live = append(Live, R)
	}
	return Live
}

func MishaJur(Entries []world.Entry) []EntryRef {
	Refs := make([]EntryRef, 0, len(Entries))
	for _, E := range Entries {
		Refs = append(Refs, EntryRef{
			Id:          E.Id,
			Aliases:     E.Aliases,
			AliasWeight: E.AliasWeight,
			Links:       E.Links,
		})
	}
	return Refs
}
