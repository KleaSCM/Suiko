/**
 * World Validator — ロード済み Store への意味的規則チェックね。
 *
 * パースエラーはロード時に捕捉済み。このパスはパースから見えないものを
 * 全部捉える：主権違反、不正なid、宙ぶらりんのリンク、絶対に一致しない
 * エイリアス。問題は「エラー（世界は壊れている、終了コード1）」と「警告
 * （遊べるが劣化、終了コード0）」に分かれる — 化粧的な問題で作者の反復を
 * 止めないためなの。
 *
 * DESIGN PHILOSOPHY:
 * ルールゼロはまずここに住む。主権を持てるエントリはちょうど一つで、それは
 * プレイヤーでなければならない — NPCが主権を名乗るのはハードエラー。
 * 下流の全ガードレイヤ（ツールロックアウト、プロンプト契約）がこのフラグに
 * キーしているからね。診断は必ずソースファイル名を挙げて、修正が一編集で
 * 済むようにしてあるわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

// id の形状："<prefix>/<slug>" — 2パート、1セパレータね。
const (
	IdSeparator = "/"
	IdMaxParts  = 2
)

type Issue struct {
	File     string
	Severity Severity
	Message  string
}

// セパレータ後半のスラッグの形状：小文字英数字を単一ハイフンでつないだもの。
// 静的パターンだから一度だけコンパイルしていいの。
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// 結果はファイル→メッセージ順にソートして返す。CLI出力を実行ごとに
// バイト単位で安定させるためね。
func (S *Store) Validate() []Issue {
	Issues := []Issue{}

	if S.Manifest.Name == "" {
		Issues = append(Issues, Issue{
			File:     FileNameManifest,
			Severity: SeverityError,
			Message:  "name is empty",
		})
	}

	Known := map[string]bool{}
	for _, E := range S.entries {
		Known[E.Id] = true
	}

	for _, E := range S.entries {
		Issues = append(Issues, validateEntry(E, Known)...)
	}

	sort.SliceStable(Issues, func(I, J int) bool {
		if Issues[I].File != Issues[J].File {
			return Issues[I].File < Issues[J].File
		}
		return Issues[I].Message < Issues[J].Message
	})
	return Issues
}

func HasErrors(Issues []Issue) bool {
	for _, Is := range Issues {
		if Is.Severity == SeverityError {
			return true
		}
	}
	return false
}

func validateEntry(E Entry, Known map[string]bool) []Issue {
	Issues := []Issue{}

	//NOTE(KleaSCM): ルールゼロ・ガードレイヤ1 — 後続の全レイヤ（プロンプト
	// 契約、ツールロックアウト）が同じフラグにキーしているから、ここでの
	// 違反は容赦なくプレイを止めるの。
	if E.Sovereign && E.Type != TypePlayer {
		Issues = append(Issues, Issue{
			File:     E.Source,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"%q claims sovereign but is not the player — only player.json may be sovereign", E.Id),
		})
	}
	if E.Type == TypePlayer && !E.Sovereign {
		Issues = append(Issues, Issue{
			File:     E.Source,
			Severity: SeverityError,
			Message:  "player must set sovereign:true — the PC is never AI-controlled",
		})
	}

	Issues = append(Issues, validateId(E)...)

	for _, L := range E.Links {
		if L == E.Id {
			Issues = append(Issues, Issue{
				File:     E.Source,
				Severity: SeverityWarning,
				Message:  "entry links to itself",
			})
			continue
		}
		if !Known[L] {
			Issues = append(Issues, Issue{
				File:     E.Source,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("links to unknown id %q", L),
			})
		}
	}

	for _, A := range E.Aliases {
		N := NormalizeAlias(A)
		if N == "" {
			Issues = append(Issues, Issue{
				File:     E.Source,
				Severity: SeverityWarning,
				Message:  "empty alias after normalization",
			})
			continue
		}
		if len(strings.Fields(N)) > MaxAliasWords {
			Issues = append(Issues, Issue{
				File:     E.Source,
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"alias %q exceeds %d words — the injector scan will never match it", A, MaxAliasWords),
			})
		}
	}

	if strings.TrimSpace(E.Summary) == "" {
		Issues = append(Issues, Issue{
			File:     E.Source,
			Severity: SeverityWarning,
			Message:  "summary is empty — compact contexts (search hits, related lists) will show nothing useful",
		})
	}

	//TODO(KleaSCM): インジェクタのスコアラーが重みで曖昧性を解消できるように
	// なったら、エントリ間エイリアス衝突の警告を足すこと
	return Issues
}

func validateId(E Entry) []Issue {
	Issues := []Issue{}
	Prefix, KnownType := IdPrefixByType[E.Type]
	if !KnownType {
		Issues = append(Issues, Issue{
			File:     E.Source,
			Severity: SeverityError,
			Message:  fmt.Sprintf("unknown entry type %q", E.Type),
		})
		return Issues
	}

	Parts := strings.SplitN(E.Id, IdSeparator, IdMaxParts)
	if len(Parts) != IdMaxParts {
		Issues = append(Issues, Issue{
			File:     E.Source,
			Severity: SeverityError,
			Message:  fmt.Sprintf("id %q must be <prefix>/<slug>", E.Id),
		})
		return Issues
	}
	if Parts[0] != Prefix {
		Issues = append(Issues, Issue{
			File:     E.Source,
			Severity: SeverityError,
			Message:  fmt.Sprintf("id %q should use prefix %q for type %q", E.Id, Prefix, E.Type),
		})
	}
	if !slugPattern.MatchString(Parts[1]) {
		Issues = append(Issues, Issue{
			File:     E.Source,
			Severity: SeverityError,
			Message:  fmt.Sprintf("id slug %q must be lowercase alphanumerics with single hyphens", Parts[1]),
		})
	}
	return Issues
}
