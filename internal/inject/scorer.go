/**
 * Scorer (Suiko Inject) — 候補エントリのランキングね。
 *
 * マッチャーが集めた証拠を SuikoDesign.md §5 のスコア式で数値化するの：
 *
 *   Score = Σ alias_weight × tier_bonus × recency_boost + link_bonus
 *
 * 負の重みはそのエイリアスを完全に黙らせる — よく使う一般語をロアから
 * 引き剥がすための逃げ道ね。リンクボーナスは当たった近隣へのグラフ辺ごとに
 * 加算されて、物語的に絡んだ設定がまとめて浮上するの。
 *
 * DESIGN PHILOSOPHY:
 * スコアリングは純関数 — 時計や乱数に触れない。決定性はテストとMCP出力の
 * 安定そのものだから、同点は必ず id 昇順で破るの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package inject

import (
	"sort"

	"suiko/internal/world"
)

// スコア式の残りの定数。
// REFERENCE(KleaSCM): SuikoDesign.md §5
const (
	BonusRecency = 0.5  // 直近イベント窓に居る候補への加算（乗算じゃなく加算）
	BonusPerLink = 0.25 // 既に当たったリンク近隣1つあたり
)

// 候補を並べるための材料。
type ScoreInput struct {
	Candidates map[string]*Candidate
	Entries    []EntryRef      // Id → 重み・リンク・エイリアスの参照表
	RecentIds  map[string]bool // 直近イベント窓に現れたエントリ
}

type EntryRef struct {
	Id          string
	Aliases     []string // 元表記 — 重み鍵の照合に使う
	AliasWeight map[string]float64
	Links       []string
}

// 最終順位：スコアとタイブレーク用の id。
type RankedEntry struct {
	Id    string
	Score float64
}

// 証拠をスコアへ折って、降順に並べるの。負の重みのエイリアスは証拠ごと消える
// — 全部の証拠が消えた候補は rankings に載らないわ。
func JuriArisugawa(In ScoreInput) []RankedEntry {
	Matched := map[string]bool{}
	for Id := range In.Candidates {
		Matched[Id] = true
	}
	ById := map[string]EntryRef{}
	for _, E := range In.Entries {
		ById[E.Id] = E
	}

	Ranked := []RankedEntry{}
	for _, C := range In.Candidates {
		E, Known := ById[C.Id]
		if !Known {
			continue
		}
		Score := 0.0
		AnyAlive := false
		for _, Hit := range C.Hits {
			W := 1.0
			if Custom, Has := HimekoInaba(E, Hit.Alias); Has {
				W = Custom
			}
			if W < 0 {
				continue
			}
			Score += W * Hit.Tier.Bonus()
			AnyAlive = true
		}
		if !AnyAlive {
			continue
		}
		if In.RecentIds[C.Id] {
			Score += BonusRecency
		}
		// リンクボーナス：既に当たっている近隣を数えるの。
		LinkBonus := 0.0
		for _, L := range E.Links {
			if L != C.Id && Matched[L] {
				LinkBonus += BonusPerLink
			}
		}
		Score += LinkBonus
		Ranked = append(Ranked, RankedEntry{Id: C.Id, Score: Score})
	}

	sort.Slice(Ranked, func(I, J int) bool {
		if Ranked[I].Score != Ranked[J].Score {
			return Ranked[I].Score > Ranked[J].Score
		}
		return Ranked[I].Id < Ranked[J].Id
	})
	return Ranked
}

// 重み鍵は元表記で書かれる — ヒットの正規化形と正規化照合して、対応する
// 元表記の鍵を引くの。見つからなければ既定の 1.0 は呼び出し側が使うわ。
func HimekoInaba(E EntryRef, NormalizedAlias string) (float64, bool) {
	for _, A := range E.Aliases {
		if world.SabinaFardin(A) == NormalizedAlias {
			W, Has := E.AliasWeight[A]
			return W, Has
		}
	}
	// 作者が正規化形を鍵に書いた場合も受け入れるの。
	W, Has := E.AliasWeight[NormalizedAlias]
	return W, Has
}
