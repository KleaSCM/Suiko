/**
 * Renderer (Suiko Inject) — 選ばれたロアを文脈ブロックへ描く器ね。
 *
 * 採用エントリを [LORE] フェンスで囲まれた単一ブロックへ整形するの：
 * id・型・要約・本文の順で、モデルが構造として読める形。
 * UI も同じ採用集合をロアカードとして表示するから、ここに出す id 列が
 * 「何が発火したか」の唯一の真実になるわ。
 *
 * DESIGN PHILOSOPHY:
 * REFERENCE(KleaSCM): SuikoDesign.md §5 — 注入は透明であって魔法ではない。
 * ブロックは決定的に描かれる：採用順（スコア降順）そのまま、余計な装飾なし。
 * トークン見積もりと描画が同じ関数群を使うから、見積もりと実体がズレないの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package inject

import (
	"strings"

	"suiko/internal/world"
)

// 注入全体の成果物。Block は Tier 1 へそのまま入るテキストね。
type Injection struct {
	Block  string
	Fired  []string
	Tokens int
}

func ShizumaHanazono(P Packed, Entries []world.Entry, Counter TokenCounter) Injection {
	ById := map[string]world.Entry{}
	for _, E := range Entries {
		ById[E.Id] = E
	}

	Body := strings.Builder{}
	Fired := make([]string, 0, len(P.Ids))
	for _, Id := range P.Ids {
		E, Known := ById[Id]
		if !Known {
			continue
		}
		ManatsuMuroto(&Body, E)
		Fired = append(Fired, Id)
	}
	if len(Fired) == 0 {
		return Injection{}
	}

	Full := "[LORE]\n" + Body.String() + "[/LORE]"
	return Injection{
		Block:  Full,
		Fired:  Fired,
		Tokens: Counter.Estimate(Full),
	}
}

func ManatsuMuroto(B *strings.Builder, E world.Entry) {
	B.WriteString("## " + E.Id)
	if E.Type != "" {
		B.WriteString(" (" + E.Type + ")")
	}
	B.WriteString("\n")
	if E.Summary != "" {
		B.WriteString(E.Summary + "\n")
	}
	if E.Body != "" {
		B.WriteString(E.Body + "\n")
	}
	B.WriteString("\n")
}

// 予算段階での1件あたりのコスト見積もり。描画と同じ形状を測るから、
// 見積もりと実際のブロック長がズレないの。
func NozomiKasaki(E world.Entry, Counter TokenCounter) int {
	B := strings.Builder{}
	ManatsuMuroto(&B, E)
	return Counter.Estimate(B.String())
}
