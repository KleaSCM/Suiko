/**
 * Player Card Parser (Suiko World) — キャラシートを賢く読む面ね。
 *
 * 作者が用意してきたキャラクターシートは形がバラバラ：SillyTavern風の
 * 「· Identity: …」行、[SECTION] 見出し、ただの段落。このパーサーは
 * それらを全部受け入れて、セクション分解・要約抽出・エイリアス拾集を
 * するの。空欄は容認 — 名前だけでも世界に入れるのが設計意図だわ。
 * REFERENCE(KleaSCM): SuikoDesign.md §9 layer 1 — PCは作者がアプリで生む
 *
 * PARSE LAYOUT:
 * ┌──────────────────┬─────────────────────────────────────────────┐
 * │ Input pattern    │ Meaning                                     │
 * ├──────────────────┼─────────────────────────────────────────────┤
 * │ · Label: text    │ 箇条書きセクション（中黒・ハイフン・ビレッジ）│
 * │ [LABEL]          │ 見出し行、以降の行がその節                   │
 * │ Label: text      │ プレーンなラベル行                           │
 * │ other lines      │ 前の節の続きとして本文に保持                  │
 * └──────────────────┴────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

import (
	"strings"
)

// 解析結果。全部のフィールドが空になり得る — 呼び出し側が既定値を埋めるわ。
type PlayerCard struct {
	Name     string
	Aliases  []string
	Summary  string // Identity（無ければ最初の節）から抽出
	Body     string // 整形済み全文 — セクション見出しを保持したまま
	Sections []Section
}

// 一つの節。順序を保って本文再構成に使うの。
type Section struct {
	Label string
	Text  string
}

// 既知の節ラベルと正規名の対応表。大文字小文字・記装飾は無視するわ。
var KnownSections = []string{
	"name", "alias", "aliases", "aka",
	"identity", "appearance", "physical", "clothing", "outfit",
	"persona", "personality", "magic", "abilities", "drive", "goal",
	"summary", "background", "history",
}

// 自由テキストを解析してカードへ折るの。何もなくても有効なゼロ値を返す。
func AzusaNakano(Raw string) PlayerCard {
	Card := PlayerCard{Aliases: []string{}}
	if strings.TrimSpace(Raw) == "" {
		return Card
	}

	// Pass 1: 行を行種別へ分類しながら節へ束ねるの。
	Sections := []Section{}
	Current := -1
	for _, Line := range strings.Split(Raw, "\n") {
		Label, Text, IsHeader := splitHeader(Line)
		switch {
		case IsHeader:
			Sections = append(Sections, Section{Label: Label, Text: Text})
			Current = len(Sections) - 1
		case Current >= 0:
			Sections[Current].Text += "\n" + Line
		default:
			// 見出し前の本文 — 匿名の冒頭節として保持するわ。
			Sections = append(Sections, Section{Label: "", Text: Line})
			Current = 0
		}
	}

	// Pass 2: 節から名前・エイリアス・要約を抜き、本文を組み立てるわ。
	var Body strings.Builder
	for _, S := range Sections {
		switch canonical(S.Label) {
		case "name":
			if Card.Name == "" {
				Card.Name = strings.TrimSpace(S.Text)
			}
		case "alias", "aliases", "aka":
			for _, A := range strings.FieldsFunc(S.Text, func(R rune) bool { return R == ',' || R == '/' }) {
				if T := strings.TrimSpace(A); T != "" {
					Card.Aliases = append(Card.Aliases, T)
				}
			}
		case "summary":
			if Card.Summary == "" {
				Card.Summary = clipSentence(S.Text)
			}
		}
		if S.Label != "" {
			Body.WriteString("[" + strings.ToUpper(S.Label) + "] " + strings.TrimSpace(S.Text) + "\n")
		} else if T := strings.TrimSpace(S.Text); T != "" {
			Body.WriteString(T + "\n")
		}
	}
	Card.Body = strings.TrimSpace(Body.String())

	// 要約が未抽出なら Identity 節、それも無ければどれかの節の頭から取るの。
	if Card.Summary == "" {
		for _, S := range Sections {
			if canonical(S.Label) == "identity" {
				Card.Summary = clipSentence(S.Text)
				break
			}
		}
	}
	if Card.Summary == "" {
		for _, S := range Sections {
			if T := clipSentence(S.Text); T != "" {
				Card.Summary = T
				break
			}
		}
	}

	// 名前は「Identity: 22-year-old …」のような文にも潜んでない —
	// 明示行しか取らない。呼び出し側がフォーム入力で補う設計ね。
	return Card
}

// 行頭の「· Label:」「- [LABEL]」「Label:」を剥がすの。ヘッダじゃない行は
// 何も剥さず false を返すわ。
func splitHeader(Line string) (Label, Text string, IsHeader bool) {
	T := strings.TrimSpace(Line)
	T = strings.TrimLeft(T, "·•-*–— ")
	T = strings.TrimSpace(T)
	T = strings.TrimPrefix(T, "[")
	Colon := strings.Index(T, ":")
	if Colon <= 0 || Colon > 24 {
		return "", Line, false
	}
	Label = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(T[:Colon], "]")))
	Text = strings.TrimSpace(T[Colon+1:])
	if !isKnownLabel(Label) {
		return "", Line, false
	}
	return Label, Text, true
}

// ラベルが既知語で始まるか（"aliases (how NPCs call you)" のような注釈許容）。
func isKnownLabel(Label string) bool {
	for _, K := range KnownSections {
		if Label == K || strings.HasPrefix(Label, K+" ") || strings.HasPrefix(K, Label) {
			return true
		}
	}
	return false
}

func canonical(Label string) string {
	for _, K := range KnownSections {
		if strings.HasPrefix(Label, K) {
			return K
		}
	}
	return Label
}

func clipSentence(Text string) string {
	Clean := strings.TrimSpace(strings.ReplaceAll(Text, "\n", " "))
	Runes := []rune(Clean)
	if len(Runes) > 140 {
		Runes = append(Runes[:139], '…')
	}
	return string(Runes)
}
