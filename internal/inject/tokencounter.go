/**
 * Token Counter (Suiko Inject) — 注入予算のトークン見積もり界面ね。
 *
 * インジェクタは予算をトークンで受け取るけど、正確なトークン化はモデルごとの
 * 話数器の仕事。この界面がその差を吸収するの：既定実装はバイト数を3で割る
 * 保守的な見積もり（英日混在テキストでの安全側）。opencode 経由で本物の
 * 話数器に接続するときも、インジェクタ側のコードは一切変わらないわ。
 *
 * DESIGN PHILOSOPHY:
 * REFERENCE(KleaSCM): SuikoDesign.md §5 — トークン計算は Provider 側の
 * TokenCounter 界面の背後に抽象済み。bytes/3 は EN/JA 混在で保守的すぎる
 * ぐらいがちょうどいい：予算超過は文脈の静かな破壊よりずっとましなの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package inject

// テキストをトークン数へ概算する界面。
type TokenCounter interface {
	Estimate(Text string) int
}

// 既定の見積もり：バイト数 ÷ 3。日本語（3バイト/字、だいたい1〜2トークン/字）
// と英語（4バイト/語、だいたい1.3トークン/語）の混在で安全側に倒してあるの。
// MATH(KleaSCM): JA ≈ 3 bytes → ~2 tok (÷1.5)、EN ≈ 1 byte/char → ~0.25 tok/char。
// ÷3 は両端をカバーする単一の保守的定数になるわ。
type ByteDivThree struct{}

func (ByteDivThree) Estimate(Text string) int {
	if len(Text) == 0 {
		return 0
	}
	return len(Text) / 3
}
