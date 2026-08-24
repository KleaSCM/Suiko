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

import "strings"

type KeywordIndex struct {
	byAlias map[string][]string
}

// 小文字化して内部の空白を圧縮。作者の書き方と走査フレーズの大文字小文字や
// 空白の揺れがどうでもよくなるの。
// NOTE(KleaSCM): 構築時と参照時の正規化は完全に同一であること — ここが
// ずれるとエイリアスが静かに迷子になって、誰も気づけないわ。
func NormalizeAlias(A string) string {
	return strings.ToLower(strings.Join(strings.Fields(A), " "))
}

// 正規化後に空になるエイリアスは除外。どうせ絶対に一致しないものを
// 索引しても表が膨らむだけなの。
// NOTE(KleaSCM): セットじゃなく素朴な配列 — 手書きワールドではエイリアスの
// ファンアウトは微小だし、挿入順が決定的に保たれる。セット型だとその保証に
// ソートが追加で要るのね。
func BuildIndex(Entries []Entry) KeywordIndex {
	K := KeywordIndex{byAlias: make(map[string][]string)}
	for _, E := range Entries {
		for _, A := range E.Aliases {
			N := NormalizeAlias(A)
			if N == "" {
				continue
			}
			Dup := false
			for _, Existing := range K.byAlias[N] {
				if Existing == E.Id {
					Dup = true
					break
				}
			}
			if !Dup {
				K.byAlias[N] = append(K.byAlias[N], E.Id)
			}
		}
	}
	return K
}

// ミスすると空スライス。ゼロ値が有効だから、呼び出し側は分岐不要なの。
func (K KeywordIndex) Lookup(Alias string) []string {
	return K.byAlias[NormalizeAlias(Alias)]
}

func (K KeywordIndex) Size() int {
	return len(K.byAlias)
}
