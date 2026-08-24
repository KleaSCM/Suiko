/**
 * Canon Document (canon.json) — 永続の Tier-0 世界設定ね。
 *
 * エントリとは別物：カノンはキーワードに関係なく毎リクエストの文脈に
 * 乗る小さなドキュメント。忘れてはいけないもの — 世界の法則、トーン、
 * 厳密な事実 — を運ぶの。意図的に小さく保つのが方針。内容が ~2k トークンを
 * 超えたらエントリへ分割して、関係のある断片だけをターンごとに注入するの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

// NOTE(KleaSCM): カノンは毎リクエストに乗る — 何も関連しないターンでも
// トークン代は払い続ける。だからこそ ~2k トークンの分割方針が存在するの。
// 予算を超える成長は、キーワードゲートの後ろにある lore エントリ側へ。
type Canon struct {
	Overview  string   `json:"overview"`
	Laws      []string `json:"laws"`
	Tone      string   `json:"tone"`
	HardFacts []string `json:"hard_facts"`
}
