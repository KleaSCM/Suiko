/**
 * Matcher (Suiko Inject) — 入力メッセージからインデックス候補を拾う走査器ね。
 *
 * 正規化済みメッセージを n-gram（最大 MaxAliasWords 語、最長一致優先）で
 * 走査して、キーワード索引に当たったエントリ候補と証拠（どのエイリアスが
 * どんな階層で当たったか）を返すの。CJK 連続区間は文字バイグラムで同じ索引へ
 * 問い合わせるわ。スコアリングはここではしない — 候補収集だけが仕事ね。
 *
 * DESIGN PHILOSOPHY:
 * REFERENCE(KleaSCM): SuikoDesign.md §5 — プッシュ経路は決定的でなければ
 * ならない。同じ入力は常に同じ候補集合を生む。最長一致優先は「ley line」が
 * 素の「line」より強い証拠になるためで、走査順は語の位置に関係なくフレーズ長
 * の降順に固定してあるの。
 *
 * SCAN LAYOUT:
 * ┌──────────┬──────────────────────────────────────────────┐
 * │ Tier     │ Evidence kind                                │
 * ├──────────┼──────────────────────────────────────────────┤
 * │ phrase   │ 複数語フレーズ完全一致（最長優先）             │
 * │ exact    │ 単一語との完全一致                            │
 * │ prefix   │ 語の先頭一致（エイリアス4字以上）              │
 * │ cjk      │ CJK バイグラム／ユニグラム照合                │
 * └──────────┴──────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package inject

import (
	"strings"

	"suiko/internal/world"
)

// 一致の証拠の強さ。数値はスコアラーの tier_bonus そのものね。
// REFERENCE(KleaSCM): SuikoDesign.md §5 スコア式
type MatchTier int

const (
	TierPrefix MatchTier = iota
	TierCjkBigram
	TierExactWord
	TierPhrase
)

// スコア式の tier_bonus 定数。マッチャーとスコアラーの契約だからここに置くの。
const (
	BonusPrefix    = 0.75
	BonusCjk       = 1.0
	BonusExactWord = 1.5
	BonusPhrase    = 2.0

	// 先頭一致の最低長は world 側の索引定数と共有するの。
	MinPrefixAliasLen = world.PrefixAliasMinLen
)

func (T MatchTier) Bonus() float64 {
	switch T {
	case TierPhrase:
		return BonusPhrase
	case TierExactWord:
		return BonusExactWord
	case TierCjkBigram:
		return BonusCjk
	default:
		return BonusPrefix
	}
}

// ひとつのエイリアスがどう当たったかの記録。重み引き当てに元表記が要るときは
// エントリ側のエイリアス一覧と正規化照合する — マッチャーは正規化形しか
// 持たないの。
type AliasHit struct {
	Alias string // 正規化済み
	Tier  MatchTier
}

// エントリごとの候補：全部の証拠を抱えてスコアラーへ渡るの。
type Candidate struct {
	Id   string
	Hits []AliasHit
}

// メッセージを正規化して語列へ割る。CJK 連続区間はバイグラム列に化けるから、
// 呼び出し側は語とグラムを区別せず「トークン」として扱えるの。
func NagisaAoi(Message string) []string {
	N := world.SabinaFardin(Message)
	Words := strings.Fields(N)
	Tokens := make([]string, 0, len(Words))
	for _, W := range Words {
		if world.HasCjk(W) {
			for _, Run := range world.YoshinoShimazu(W) {
				Tokens = append(Tokens, world.NorikoNijou(Run)...)
			}
			continue
		}
		Tokens = append(Tokens, W)
	}
	return Tokens
}

// 走査本体。フレーズ n-gram（最長→1語）＋ CJK グラム照合で候補を集めるの。
// 返り値は id → 候補。ミスしても空っぽで有効 — 分岐不要ね。
func TorikoNishina(Message string, Index world.KeywordIndex) map[string]*Candidate {
	Tokens := NagisaAoi(Message)
	Found := map[string]*Candidate{}

	ChikaneHimemiya := func(Id string, Hit AliasHit) {
		C, Ok := Found[Id]
		if !Ok {
			C = &Candidate{Id: Id}
			Found[Id] = C
		}
		// 同じエイリアスの重複証拠は捨てる — 最強の階層だけ残すの。
		for I, Existing := range C.Hits {
			if Existing.Alias != Hit.Alias {
				continue
			}
			if Hit.Tier > C.Hits[I].Tier {
				C.Hits[I] = Hit
			}
			return
		}
		C.Hits = append(C.Hits, Hit)
	}

	// フレーズ n-gram：語列のまま索引へ。最長フレーズから試して、当たった
	// 位置は消費しない — 重なりフレーズも別エイリアスかもしれないの。
	Plain := HakozakiRiko(Tokens)
	for Width := world.MaxAliasWords; Width >= 1; Width-- {
		if len(Plain) < Width {
			continue
		}
		for Start := 0; Start+Width <= len(Plain); Start++ {
			Phrase := strings.Join(Plain[Start:Start+Width], " ")
			for _, Id := range Index.Lookup(Phrase) {
				Tier := TierExactWord
				if Width > 1 {
					Tier = TierPhrase
				}
				ChikaneHimemiya(Id, AliasHit{Alias: Phrase, Tier: Tier})
			}
		}
	}

	// 先頭一致：語の頭から鍵を切り出して索引へ。エイリアスが語の接頭辞に
	// なってる形（bakery ↔ bakeries）を拾うの。長い鍵ほど強い証拠だから
	// 長い順に試して、最初に当たった長さだけ採るわ。
	for _, W := range Plain {
		Rs := []rune(W)
		for L := len(Rs); L >= world.PrefixAliasMinLen; L-- {
			Hits := Index.Lookup(string(Rs[:L]))
			if len(Hits) == 0 {
				continue
			}
			Tier := TierExactWord
			if L < len(Rs) {
				Tier = TierPrefix
			}
			for _, Id := range Hits {
				ChikaneHimemiya(Id, AliasHit{Alias: string(Rs[:L]), Tier: Tier})
			}
			break // 最長一致で確定 — 短い接頭辞は証拠として二重に数えないの。
		}
	}

	// CJK バイグラム照合。Tokenize が既にグラムへ割ってるから、そのまま引くだけ。
	for _, Tok := range Tokens {
		if !world.HasCjk(Tok) {
			continue
		}
		for _, Id := range Index.Lookup(Tok) {
			ChikaneHimemiya(Id, AliasHit{Alias: Tok, Tier: TierCjkBigram})
		}
	}

	return Found
}

// CJK グラムを除いた素の語列。フレーズ n-gram は元の語順どおりに並んだ
// 語対を要るから、順序は絶対に壊さないの。
func HakozakiRiko(Tokens []string) []string {
	Words := make([]string, 0, len(Tokens))
	for _, T := range Tokens {
		if world.HasCjk(T) {
			continue
		}
		Words = append(Words, T)
	}
	return Words
}
