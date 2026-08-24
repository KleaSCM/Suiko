/**
 * WorldStore (Suiko) — ワールドディレクトリ一つのメモリ内イメージね。
 *
 * ロードは world.json・canon.json・player.json と型付きサブディレクトリ下の
 * 全エントリファイルを読み、キーワードインデックスを組み上げて、不変の
 * スナップショットを返すの。ストアは今後のMCPサーバ、プロンプトコンパイラ、
 * Wailsアプリが共有する唯一の真実の源。ディスクはあくまで本当の権威で、
 * fs watcher が M3 でスナップショット更新を担うわ。
 *
 * DESIGN PHILOSOPHY:
 * 厳格パースがバリデーションの第一線：未知のJSONフィールドはロードエラー。
 * 「alias_weight」の書き間違いが静かにロアを落とすより、世界を拒否する方が
 * ずっとましなの。ロード成功後の実行時アクセスは zero-is-valid — 存在しないidでの
 * GetEntry は ZeroEntry を返すから、呼び出し側は決して分岐しない。
 * エントリはid順にソートして格納。決定的な反復順がMCPの一覧出力とテストを安定させるわ。
 *
 * DATA LAYOUT:
 * ┌──────────┬────────────────────────────────────────────────────────┐
 * │ Field    │ Purpose                                                │
 * ├──────────┼────────────────────────────────────────────────────────┤
 * │ Manifest │ world.json の調整値。既定適用済み                       │
 * │ Canon    │ Tier-0 ドキュメント。常に文脈に在席                     │
 * │ Player   │ 主権を持つPCエントリ（player.json 由来）                │
 * │ entries  │ player ＋ 全型付きエントリ。Id 順にソート               │
 * │ byId     │ Id → entries の添字                                    │
 * │ Index    │ 正規化エイリアス → id 群                                │
 * └──────────┴────────────────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ワールドディレクトリ内の正規ファイル名。
const (
	FileNameManifest = "world.json"
	FileNameCanon    = "canon.json"
	FileNamePlayer   = "player.json"
)

// 手書きワールド向けの事前確保のヘッドルーム。超えても再スライスされるだけ。
const InitialEntryCapacity = 16

type Store struct {
	Root     string
	Manifest WorldManifest
	Canon    Canon
	Player   Entry

	entries []Entry
	byId    map[string]int
	Index   KeywordIndex
}

// 起動経路 — 読めない・壊れているファイルで即座に失敗するの。
// 成功したあとの実行時読み取りは分岐フリーになるわ。
func Load(Root string) (*Store, Error) {
	if Root == "" {
		return nil, NewError(ErrCodeUsage, "world directory path is required")
	}
	if Info, StatErr := os.Stat(Root); StatErr != nil || !Info.IsDir() {
		return nil, NewError(ErrCodeIo, "world directory not found: "+Root)
	}

	M := WorldManifest{}
	if Err := readJson(filepath.Join(Root, FileNameManifest), &M, true); !Err.Ok() {
		return nil, Err
	}
	M = M.WithDefaults()

	C := Canon{}
	if Err := readJson(filepath.Join(Root, FileNameCanon), &C, false); !Err.Ok() {
		return nil, Err
	}

	P := Entry{}
	if Err := readJson(filepath.Join(Root, FileNamePlayer), &P, true); !Err.Ok() {
		return nil, Err
	}
	P.Type = TypePlayer
	P.Source = FileNamePlayer

	S := &Store{
		Root:     Root,
		Manifest: M,
		Canon:    C,
		Player:   P,
		entries:  make([]Entry, 0, InitialEntryCapacity),
		byId:     make(map[string]int),
	}
	S.entries = append(S.entries, P)
	S.byId[P.Id] = 0

	for _, Dir := range EntryDirs {
		Kind, _ := TypeForDir(Dir)
		Paths, _ := filepath.Glob(filepath.Join(Root, Dir, "*.json"))
		for _, Path := range Paths {
			E := Entry{}
			if Err := readJson(Path, &E, true); !Err.Ok() {
				return nil, Err
			}
			E.Source = relSource(Root, Path)
			// 型の省略はディレクトリから推論。衝突する記述は著者がファイルを
			// 間の場所に置いた証拠 — どちらの意図か推測せず拒否するの。
			if E.Type == "" {
				E.Type = Kind
			} else if E.Type != Kind {
				return nil, NewError(ErrCodeSchema, fmt.Sprintf(
					"%s: type %q does not match directory %q", E.Source, E.Type, Dir))
			}
			//NOTE(KleaSCM): 重複idは警告じゃなくロード拒否 — ふたつのエントリが
			// 同じidを名乗ると、下流のグラフ辺が全部曖昧になってしまうのね。
			if _, Exists := S.byId[E.Id]; Exists {
				return nil, NewError(ErrCodeData, fmt.Sprintf(
					"%s: duplicate entry id %q", E.Source, E.Id))
			}
			S.byId[E.Id] = len(S.entries)
			S.entries = append(S.entries, E)
		}
	}

	S.sortEntries()
	S.Index = BuildIndex(S.entries)
	return S, Error{}
}

// NOTE(KleaSCM): 未知idは共有ゼロスタブに着地 — 呼び出し側は結果を分岐なしで
// 使うのが zero-is-initialization の流儀ね。
func (S *Store) GetEntry(Id string) Entry {
	if I, Ok := S.byId[Id]; Ok {
		return S.entries[I]
	}
	return ZeroEntry
}

func (S *Store) Entries() []Entry {
	return S.entries
}

func (S *Store) Count() int {
	return len(S.entries)
}

// プレイヤーは常にスロット0に固定。決定的な順序のおかげでCLI出力・MCP一覧・
// テストが実行ごとにバイト単位で安定するの。
func (S *Store) sortEntries() {
	Rest := S.entries[1:]
	sort.Slice(Rest, func(I, J int) bool {
		return Rest[I].Id < Rest[J].Id
	})
}

// 厳格デコード — 未知フィールドは黙って落とさずロードエラー。
// 任意ファイルの欠如は「そのまま成功」扱いなの。
// REFERENCE(KleaSCM): encoding/json DisallowUnknownFields — 作者の書き間違い
// （"alias_weigth" 等）をロード時の大きな失敗に変えるためね。
func readJson(Path string, Dst any, Required bool) Error {
	Data, Err := os.ReadFile(Path)
	if Err != nil {
		if os.IsNotExist(Err) && !Required {
			return Error{}
		}
		if os.IsNotExist(Err) {
			return NewError(ErrCodeIo, Path+": required file not found")
		}
		return NewError(ErrCodeIo, fmt.Sprintf("%s: %v", Path, Err))
	}
	Dec := json.NewDecoder(bytes.NewReader(Data))
	Dec.DisallowUnknownFields()
	if DecErr := Dec.Decode(Dst); DecErr != nil {
		return NewError(ErrCodeSchema, fmt.Sprintf("%s: %v", Path, DecErr))
	}
	return Error{}
}

func relSource(Root, Path string) string {
	Rel, Err := filepath.Rel(Root, Path)
	if Err != nil {
		return Path
	}
	return Rel
}
