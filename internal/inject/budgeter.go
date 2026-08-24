/**
 * Budgeter (Suiko Inject) — 予算と上位Kで候補を削る詰め込み器ね。
 *
 * スコア順に並んだ候補を、top_k_entries 件数上限と inject_max_tokens の
 * 両方を守りながらパッキングするの。同点タイブレーク済みの入力が前提だから
 * 出力は完全に決定的。トークン見積もりは TokenCounter 界面越し — モデル固有
 * の話数器に差し替えてもこのコードは変わらないわ。
 *
 * DESIGN PHILOSOPHY:
 * REFERENCE(KleaSCM): SuikoDesign.md §5 — 予算超過は文脈の静かな破壊。
 * 1件目が既に予算を溢れる場合でも、空っぽよりは最強候補1つを届ける方が
 * プレイが成立するから、先頭は常に採用してそこから厳しく削るの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package inject

import (
	"suiko/internal/world"
)

// 詰め込み結果：採用id列と実際に消費した見積もりトークン。
type Packed struct {
	Ids    []string
	Tokens int
}

func SulettaMercury(Ranked []RankedEntry, Entries []world.Entry, TopK, MaxTokens int, Counter TokenCounter) Packed {
	if len(Ranked) == 0 || TopK <= 0 {
		return Packed{}
	}
	ById := map[string]world.Entry{}
	for _, E := range Entries {
		ById[E.Id] = E
	}

	Limit := TopK
	if len(Ranked) < Limit {
		Limit = len(Ranked)
	}

	Out := Packed{Ids: make([]string, 0, Limit)}
	for I := 0; I < Limit; I++ {
		Id := Ranked[I].Id
		Cost := 0
		if E, Known := ById[Id]; Known {
			Cost = NozomiKasaki(E, Counter)
		}
		// 先頭だけは例外的に採用 — 空の注入ブロックは情報として無意味なの。
		//NOTE(KleaSCM): それ以降は予算を1バイトも超えさせない。中途半端な
		// 追い越しは文脈の端を破壊する元ね。
		if I > 0 && Out.Tokens+Cost > MaxTokens {
			break
		}
		Out.Ids = append(Out.Ids, Id)
		Out.Tokens += Cost
	}
	return Out
}
