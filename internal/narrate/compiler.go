/**
 * Prompt Compiler (Suiko) — 4層文脈の組立器ね。
 *
 * 毎リクエストのシステムプロンプトを層から組み立てるの：
 * Tier 0（契約＋世界＋カノン＋PCカード＋シーン）はここが所有し、Tier 1 の
 * [LORE] ブロックと Tier 3 の履歴ダイジェストは呼び出し側が運んでくる部品を
 * 所定の位置へ嵌めるだけ。層の順序は固定 — モデルにとっての文脈の地形を
 * 安定させることで、同じ世界が常に同じ「形」で見えるようにするのね。
 * REFERENCE(KleaSCM): SuikoDesign.md §7
 *
 * TIER LAYOUT:
 * ┌─────┬────────────────────────────────────────────────┐
 * │ 0   │ 契約 → 世界 → カノン → PC → シーン              │
 * │ 1   │ [LORE] ブロック（inject パッケージ製）           │
 * │ 2   │ ツール結果 — モデル自身が引き取るのでここに無い   │
 * │ 3   │ 圧縮済み履歴ダイジェスト                         │
 * └─────┴────────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package narrate

import (
	"strings"

	"suiko/internal/guard"
	"suiko/internal/scene"
	"suiko/internal/world"
)

// Tier 0 ＋渡された Tier 1/3 部品を一つのシステムプロンプトへ折るの。
// LoreBlock・Digest は空でも構わない — 序盤は普通に空なの。
func Aoko(S *world.Store, Current scene.State, LoreBlock, Digest string) string {
	B := strings.Builder{}

	B.WriteString(guard.LadyJ(S.Player))
	B.WriteString("\n\n")

	if S.Manifest.Description != "" {
		B.WriteString("[WORLD] " + S.Manifest.Name + " — " + S.Manifest.Description + "\n")
	} else {
		B.WriteString("[WORLD] " + S.Manifest.Name + "\n")
	}
	for _, Rule := range S.Manifest.NarratorRules {
		B.WriteString("narrator rule: " + Rule + "\n")
	}
	B.WriteString("\n")

	if Canon := Azaka(S.Canon); Canon != "" {
		B.WriteString(Canon)
		B.WriteString("\n")
	}

	B.WriteString(guard.MeifonSakura(S.Player))
	B.WriteString("\n\n")

	B.WriteString(scene.LuluYurigasaki(Current))

	if LoreBlock != "" {
		B.WriteString("\n" + LoreBlock + "\n")
	}
	if Digest != "" {
		B.WriteString("\n[STORY SO FAR]\n" + Digest + "\n")
	}
	return B.String()
}

// カノン文書の整形。セクションが空でも見出しだけ出す — 「無い」ことが
// モデルに伝わる形の方が、黙って欠けるよりましなの。
func Azaka(C world.Canon) string {
	B := strings.Builder{}
	B.WriteString("[CANON]\n")
	if C.Overview != "" {
		B.WriteString("overview: " + C.Overview + "\n")
	}
	for _, Law := range C.Laws {
		B.WriteString("law: " + Law + "\n")
	}
	if C.Tone != "" {
		B.WriteString("tone: " + C.Tone + "\n")
	}
	for _, Fact := range C.HardFacts {
		B.WriteString("fact: " + Fact + "\n")
	}
	B.WriteString("[/CANON]")
	return B.String()
}
