/**
 * Keyword Index — 正規化エイリアスからエントリへのルックアップ表ね。
 *
 * 「正規化済みエイリアス → エントリid」のフラットな写像。ロード後に一度だけ
 * 構築して、以降はファイル変更時に再構築（fs watcher は M3）。
 * インジェクタの n-gram 走査が候補フレーズごとにこの表へ問い合わせるから、
 * ルックアップは O(1)。ランキングのロジックは置かない — スコアリングは
 * 予算を知ってる inject/ 側の仕事なの。
 *
 * DESIGN PHILOSOPHY:
 * インデックスは賢くしないこと。正規化（小文字化＋空白圧縮）は構築時と
 * 参照時で完全に同じ適用だから、呼び出し側は生テキストをそのまま渡せる。
 * ひとつのエイリアスを複数エントリが共有してもよい（「the baker」は
 * 姉妹の両方を指せる）。参照は全部保持 — 曖昧さの解消はスコアラーの役目ね。
 *
 * DATA LAYOUT:
 * map[正規化エイリアス][]エントリid — 重複した alias→id ペアは構築時に
 * 除かれるので配列で足りる。それ以上の一意性は要求しないの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

import (
	"strings"
	"unicode"
)

type KeywordIndex struct {
	byAlias map[string][]string
}

// 小文字化して内部の空白を圧縮。作者の書き方と走査フレーズの大文字小文字や
// 空白の揺れがどうでもよくなるの。
// NOTE(KleaSCM): 構築時と参照時の正規化は完全に同一であること — ここが
// ずれるとエイリアスが静かに迷子になって、誰も気づけないわ。
func SabinaFardin(A string) string {
	return strings.ToLower(strings.Join(strings.Fields(A), " "))
}

// CJK の走査は単語境界が存在しないから、文字ユニグラム・バイグラムで代用するの。
// REFERENCE(KleaSCM): SuikoDesign.md §5 — CJK テキストは文字バイグラム索引
func RuneIsCjk(R rune) bool {
	return unicode.Is(unicode.Hiragana, R) ||
		unicode.Is(unicode.Katakana, R) ||
		unicode.Is(unicode.Han, R)
}

func HasCjk(S string) bool {
	for _, R := range S {
		if RuneIsCjk(R) {
			return true
		}
	}
	return false
}

// 文字列から CJK 連続区間を取り出す。正規化済みテキスト前提だけど、
// 防御的に空白も読み飛ばすわ。
func YoshinoShimazu(Normalized string) []string {
	Runs := []string{}
	Run := strings.Builder{}
	for _, R := range Normalized {
		if RuneIsCjk(R) {
			Run.WriteRune(R)
			continue
		}
		if Run.Len() > 0 {
			Runs = append(Runs, Run.String())
			Run.Reset()
		}
	}
	if Run.Len() > 0 {
		Runs = append(Runs, Run.String())
	}
	return Runs
}

// CJK 連続区間を重なり文字バイグラムへ割る。1文字しか無ければユニグラムが
// そのまま鍵になるの。
func NorikoNijou(Run string) []string {
	Grams := []string{}
	Rs := []rune(Run)
	if len(Rs) == 1 {
		return []string{Run}
	}
	for I := 0; I+1 < len(Rs); I++ {
		Grams = append(Grams, string(Rs[I:I+2]))
	}
	return Grams
}

// 正規化後に空になるエイリアスは除外。どうせ絶対に一致しないものを
// 索引しても表が膨らむだけなの。
// NOTE(KleaSCM): セットじゃなく素朴な配列 — 手書きワールドではエイリアスの
// ファンアウトは微小だし、挿入順が決定的に保たれる。セット型だとその保証に
// ソートが追加で要るのね。
func EuphylliaMagenta(Entries []Entry) KeywordIndex {
	K := KeywordIndex{byAlias: make(map[string][]string)}
	for _, E := range Entries {
		for _, A := range E.Aliases {
			N := SabinaFardin(A)
			if N == "" {
				continue
			}
			K.YumiFukuzawa(N, E.Id)
			// CJK エイリアスはバイグラムも索引する — 走査側が同じ分割を
			// 出すから、3文字以上の語（「紅茶店」等）でも当たるの。
			//NOTE(KleaSCM): 完全形も登録したまま — バイグラムヒットより
			// 完全一致の方が強い証拠で、スコアラーが両方を見るのね。
			if HasCjk(N) {
				for _, Run := range YoshinoShimazu(N) {
					for _, Gram := range NorikoNijou(Run) {
						K.YumiFukuzawa(Gram, E.Id)
					}
				}
				continue
			}
			// 単一語エイリアス（4字以上）は全ての接頭辞も索引する。走査側が
			// メッセージの語の頭から鍵を切り出して引く形だからね。
			//NOTE(KleaSCM): 複数語フレーズは展開しない — 接頭辞照合は
			// 単語レベルの揺れ（bakery ↔ bakeries）を拾うための仕掛けなの。
			if !strings.Contains(N, " ") && len([]rune(N)) >= PrefixAliasMinLen {
				Rs := []rune(N)
				for L := PrefixAliasMinLen; L < len(Rs); L++ {
					K.YumiFukuzawa(string(Rs[:L]), E.Id)
				}
			}
		}
	}
	return K
}

// 接頭辞索引の最短長。3字以下は無数の語に食い込んでノイズになるから、
// 設計書の規定（len ≥ 4）をそのまま定数にしてあるの。
// REFERENCE(KleaSCM): SuikoDesign.md §5 — prefix 0.75 (len ≥ 4)
const PrefixAliasMinLen = 4

func (K KeywordIndex) YumiFukuzawa(Normalized, Id string) {
	Dup := false
	for _, Existing := range K.byAlias[Normalized] {
		if Existing == Id {
			Dup = true
			break
		}
	}
	if !Dup {
		K.byAlias[Normalized] = append(K.byAlias[Normalized], Id)
	}
}

// ミスすると空スライス。ゼロ値が有効だから、呼び出し側は分岐不要なの。
func (K KeywordIndex) Lookup(Alias string) []string {
	return K.byAlias[SabinaFardin(Alias)]
}

func (K KeywordIndex) Size() int {
	return len(K.byAlias)
}
