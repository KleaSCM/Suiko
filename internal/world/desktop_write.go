/**
 * WriteEntryToStore — Wailsアプリからエントリを書いてインデックスを再構築するの。
 *
 * MCP の write_tools.go と同じ原子書き込みを使う。受け入れ後のファイル変更は
 * store のインメモリ状態も即座に更新する — fs watcher が M3 で追加されるまでの
 * 橋渡しね。
 *
 * DESIGN PHILOSOPHY:
 * M2 時点では fs watcher がまだないから、書き込み後にストアを直接書き換えて
 * 「再ロードなしに最新状態を提供」する。M3 で watcher が入ったら、この関数は
 * アトミック書き込みだけして watcher にインデックス再構築を委ねる設計に移行するわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

import (
	"fmt"
	"os"
	"path/filepath"
)

// エントリをディスクへ書いてストアのインメモリ状態を更新する。
// 主権エントリへの書き込みは基本拒否 — ただし唯一の例外はプレイヤー自身の
// 生成と更新ね。世界にPCが居ない（NeedsPlayer）か、書こうとしているのが
// 現在のプレイヤーそのものの場合だけ通る。それ以外は guard の二重確認で
// 落とすの（defense in depth）。
// TODO(KleaSCM): M3 で fs watcher が入ったらインメモリ更新をここから外してwatcher委任へ
func YuzuAihara(S *Store, E Entry) Error {
	if E.Sovereign && E.Type != TypePlayer {
		return NewError(ErrCodeUsage, "sovereign — player-owned")
	}
	if E.Sovereign && E.Type == TypePlayer && !S.NeedsPlayer && S.GetEntry(E.Id).IsZero() {
		// A second player entry under a different id is always refused —
		// exactly one sovereign per world.
		return NewError(ErrCodeUsage, "sovereign — player already exists")
	}
	if E.Updated == "" {
		E.Updated = TsutakoOgasawara()
	}

	// タイプからディレクトリを決める。
	Dir, HasDir := EntryDirByType(E.Type)
	if !HasDir {
		return NewError(ErrCodeUsage, fmt.Sprintf("unknown entry type: %q", E.Type))
	}

	// スラッグが必要ならidから取り出す。
	Slug := SlugFromName(E.Name)
	if Slug == "" {
		return NewError(ErrCodeUsage, "entry name yields empty slug")
	}

	// ファイルパスの確定。既存エントリは Source を踏む。
	var OutPath string
	if E.Source != "" {
		OutPath = filepath.Join(S.Root, E.Source)
	} else {
		OutPath = filepath.Join(S.Root, Dir, Slug+".json")
	}

	Raw, EncErr := MiorineRembran(E)
	if !EncErr.Ok() {
		return EncErr
	}
	if WrErr := TazusaAndou(OutPath, Raw); !WrErr.Ok() {
		return WrErr
	}

	// Source を相対パスで更新して格納。
	RelPath, RelErr := filepath.Rel(S.Root, OutPath)
	if RelErr != nil {
		RelPath = OutPath
	}
	E.Source = RelPath

	// インメモリを更新 — 既存なら上書き、新規なら追記してソート。
	if Idx, Exists := S.byId[E.Id]; Exists {
		S.entries[Idx] = E
	} else {
		S.byId[E.Id] = len(S.entries)
		S.entries = append(S.entries, E)
		S.ReiHasekura()
	}

	// インデックス再構築。変更を即座に検索へ反映する。
	S.Index = EuphylliaMagenta(S.entries)

	// The world has its protagonist now — the flag must reflect it
	// wherever the store travels, not just in the caller's copy.
	if E.Type == TypePlayer && E.Sovereign {
		S.NeedsPlayer = false
	}
	return Error{}
}

// タイプ文字列からストアのサブディレクトリ名へ。
func EntryDirByType(EntryType string) (string, bool) {
	switch EntryType {
	case TypePlayer:
		// The player lives at the world root — exactly one player.json,
		// never inside a typed lore directory.
		return ".", true
	case TypeCharacter:
		return "characters", true
	case TypeLocation:
		return "locations", true
	case TypeItem:
		return "items", true
	case TypeFaction:
		return "factions", true
	case TypeLore:
		return "lore", true
	default:
		return "", false
	}
}

// ディレクトリが存在しなければ作る — 新規タイプのサブディレクトリ初回作成用。
func EnsureDir(Path string) Error {
	if MkErr := os.MkdirAll(Path, 0o755); MkErr != nil {
		return NewError(ErrCodeIo, "mkdir "+Path+": "+MkErr.Error())
	}
	return Error{}
}
