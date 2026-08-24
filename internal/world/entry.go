/**
 * Entry Schema (Suiko World Format) — カノンの素になるレコード型ね。
 *
 * エントリ一つがそのまま世界の正史のひとかけら：NPC、場所、アイテム、勢力、
 * 自由形式の話題。ワールドディレクトリの下に個別JSONとして置かれて、
 * 検索の原子単位になるの。キーワードインデックスがエイリアスをidに写像して、
 * 選ばれたエントリは丸ごと文脈に注入されるわ。
 *
 * DESIGN PHILOSOPHY:
 * データベースじゃなくてファイル。ワールドは git に優しい手編集JSONのフォルダで、
 * テキストの所有者は作者、エンジンが持つのはキャッシュだけ。
 * 厳格デコード（未知フィールドの拒否）のおかげで、作者の書き間違いは静かに消えずに
 * ロードエラーになる — つまりバリデーションはパースの時点から始まってるの。
 *
 * DATA LAYOUT:
 * ┌─────────────┬───────────────────────────────────────────────────────┐
 * │ Field       │ Meaning                                               │
 * ├─────────────┼───────────────────────────────────────────────────────┤
 * │ Id          │ "<prefix>/<slug>"、一意で安定したグラフノード          │
 * │ Type        │ player|character|location|item|faction|lore           │
 * │ Aliases     │ プッシュ注入を引き起こすキーワード                     │
 * │ Summary     │ コンパクト文脈用の一行要約（検索ヒット、関連一覧）     │
 * │ Body        │ 選ばれたときに注入される完全な設定                     │
 * │ Links       │ 関連エントリのid — 走査のためのグラフ辺                │
 * │ Tags        │ 自由形式のグルーピング／フィルタ                       │
 * │ AliasWeight │ エイリアスごとのスコア倍率（既定 1.0）                 │
 * │ Sovereign   │ プレイヤー専用の主権マーカー。player.json のみ設定可   │
 * │ Updated     │ RFC3339 タイムスタンプ。衝突解決は新しい方勝ち         │
 * │ Source      │ 読み込み元ファイル（ワイヤ形式には出ない）              │
 * └─────────────┴───────────────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

// ロアの種類。ローダーが受け付けて、ディレクトリ名がこれに対応するの。
const (
	TypePlayer    = "player"
	TypeCharacter = "character"
	TypeLocation  = "location"
	TypeItem      = "item"
	TypeFaction   = "faction"
	TypeLore      = "lore"
)

// インジェクタはこの幅まで n-gram を走査するだけ。それより長いエイリアスは
// 絶対に発火しないから、著者時に受け付けても静かな死重になるだけなの。
// NOTE(KleaSCM): MaxAliasWords とインジェクタの走査幅は必ず一致させること
const MaxAliasWords = 4

// プレフィックスはグラフ同一性の一部。リンクがプレフィックスを埋め込んでるから、
// 対応表を変えると古い形を参照する辺が全部迷子になるの。
var IdPrefixByType = map[string]string{
	TypePlayer:    "player",
	TypeCharacter: "char",
	TypeLocation:  "loc",
	TypeItem:      "item",
	TypeFaction:   "faction",
	TypeLore:      "lore",
}

// ロード時に走査される型付きロアのディレクトリ。追記型のログは、書き込み側が
// 生まれる書き戻しマイルストーンまで外しておくわ。
// TODO(KleaSCM): LogEvent 実装時に events/ をロード対象へ折り込む（M3）
var EntryDirs = []string{"characters", "locations", "items", "factions", "lore"}

type Entry struct {
	Id          string             `json:"id"`
	Type        string             `json:"type"`
	Name        string             `json:"name"`
	Aliases     []string           `json:"aliases"`
	Summary     string             `json:"summary"`
	Body        string             `json:"body"`
	Links       []string           `json:"links"`
	Tags        []string           `json:"tags"`
	AliasWeight map[string]float64 `json:"alias_weight"`
	Sovereign   bool               `json:"sovereign"`
	Updated     string             `json:"updated"`

	// 読み込み元のファイル。バリデータの診断がここを指すから、修正は
	// 一編集で済むの。ワイヤ形式には絶対出ないわ。
	Source string `json:"-"`
}

// 未知idのルックアップはこの共有スタブに着地する。常に有効、nilは無し。
// NOTE(KleaSCM): zero-is-initialization — 呼び出し側は分岐なしで結果を
// 使えるので、nilチェック由来のバグ群が検索経路から丸ごと消えるの。
var ZeroEntry = Entry{}

func (E Entry) IsZero() bool {
	return E.Id == ""
}

func TypeForDir(Dir string) (string, bool) {
	switch Dir {
	case "characters":
		return TypeCharacter, true
	case "locations":
		return TypeLocation, true
	case "items":
		return TypeItem, true
	case "factions":
		return TypeFaction, true
	case "lore":
		return TypeLore, true
	default:
		return "", false
	}
}
