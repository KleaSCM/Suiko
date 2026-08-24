/**
 * World Package Tests — ロード＋バリデーションの意図付きカバレッジね。
 *
 * フィクスチャは全部インラインJSONから t.TempDir へ組まれて、各テストが
 * どの規則を試してるかを名前で語るの。モックじゃなく実ファイルを使うのは、
 * ファイルシステム境界こそがここでの単位（厳格デコードの挙動）だからわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"suiko/internal/world"
)

// フィクスチャのファイル権限。テストはrootで走らないから固定でいいの。
const (
	DirPerm  = 0o755
	FilePerm = 0o644
)

// 最小の有効なワールドを組み立てて、その上に Overrides を重ねるの
// （空文字はファイル削除 — 必須ファイル強制を証明するテストで使うわ）。
func WriteWorld(T *testing.T, Overrides map[string]string) string {
	T.Helper()
	Files := map[string]string{
		world.FileNameManifest: `{
			"name": "Yurikawa",
			"description": "Rainy port town.",
			"starting_scene": "Evening. Rain."
		}`,
		world.FileNameCanon: `{
			"overview": "A small port town.",
			"laws": ["Ley lines grant small magics."],
			"tone": "Cozy melancholy.",
			"hard_facts": []
		}`,
		world.FileNamePlayer: `{
			"id": "player/hanako",
			"type": "player",
			"name": "Hanako",
			"aliases": ["Hana"],
			"summary": "Pastel-skirted engineer new to town.",
			"body": "Rents the room above the bakery.",
			"sovereign": true,
			"updated": "2026-08-24T00:00:00Z"
		}`,
		"characters/kaori.json": `{
			"id": "char/kaori",
			"type": "character",
			"name": "Kaori",
			"aliases": ["the baker"],
			"summary": "Baker and secret hedge-witch.",
			"body": "Ninety-year-old family bakery.",
			"links": ["loc/bakery"],
			"tags": ["household"],
			"updated": "2026-08-24T00:00:00Z"
		}`,
		"locations/bakery.json": `{
			"id": "loc/bakery",
			"type": "location",
			"name": "The Bakery",
			"aliases": ["bakery", "Bakery Street"],
			"summary": "Warm brick shop over a ley crossing.",
			"body": "Smells of sourdough at 4am.",
			"links": [],
			"tags": [],
			"updated": "2026-08-24T00:00:00Z"
		}`,
	}
	for Path, Blank := range Overrides {
		if Blank == "" {
			delete(Files, Path)
			continue
		}
		Files[Path] = Blank
	}
	Root := T.TempDir()
	for Path, Content := range Files {
		Full := filepath.Join(Root, Path)
		if MkdirErr := os.MkdirAll(filepath.Dir(Full), DirPerm); MkdirErr != nil {
			T.Fatalf("mkdir %s: %v", Full, MkdirErr)
		}
		if WriteErr := os.WriteFile(Full, []byte(Content), FilePerm); WriteErr != nil {
			T.Fatalf("write %s: %v", Full, WriteErr)
		}
	}
	return Root
}

// ハッピーパスの証明：manifest/canon/player/entries が全部ロードされて、
// ルックアップが当たって、未知idは失敗ではなく ZeroEntry を返すの。
func TestLoadValidWorld(t *testing.T) {
	Store, Err := world.Load(WriteWorld(t, nil))
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	if Store.Count() != 3 {
		t.Fatalf("expected 3 entries, got %d", Store.Count())
	}
	Kaori := Store.GetEntry("char/kaori")
	if Kaori.IsZero() || Kaori.Name != "Kaori" {
		t.Fatalf("expected Kaori entry, got %+v", Kaori)
	}
	if !Store.GetEntry("char/nobody").IsZero() {
		t.Fatal("unknown id must return zero entry")
	}
	// プレイヤーはid順に関係なく常に先頭に居座るの。
	if Store.Entries()[0].Type != world.TypePlayer {
		t.Fatal("player must sort first")
	}
}

// 厳格デコードの証明：作者の書き間違いは静かに消えたフィールドじゃなく、
// 大きなロードエラーになるの。
func TestLoadRejectsUnknownField(t *testing.T) {
	_, Err := world.Load(WriteWorld(t, map[string]string{
		world.FileNameManifest: `{"name": "X", "totally_not_a_typo": true}`,
	}))
	if Err.Ok() || Err.Code != world.ErrCodeSchema {
		t.Fatalf("expected schema error, got %+v", Err)
	}
}

// ファイル間の同一性重複は拒否されるという証明 — ふたつのエントリがひとつの
// idを名乗るとグラフのリンクが曖昧になるからね。
func TestLoadRejectsDuplicateId(t *testing.T) {
	_, Err := world.Load(WriteWorld(t, map[string]string{
		"characters/twin.json": `{
			"id": "char/kaori",
			"type": "character",
			"name": "Twin",
			"aliases": [],
			"summary": "",
			"body": "",
			"links": [],
			"tags": [],
			"updated": "2026-08-24T00:00:00Z"
		}`,
	}))
	if Err.Ok() || Err.Code != world.ErrCodeData {
		t.Fatalf("expected data error for duplicate id, got %+v", Err)
	}
}

// ルールゼロ・レイヤ1：主権を名乗るNPCはハードエラーになるの。
func TestValidateFlagsSovereignNpc(t *testing.T) {
	Store, Err := world.Load(WriteWorld(t, map[string]string{
		"characters/kaori.json": strings.Replace(
			FixtureKaoriSovereign(), `"sovereign": false`, `"sovereign": true`, 1),
	}))
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	Issues := Store.Validate()
	if !world.HasErrors(Issues) {
		t.Fatal("sovereign NPC must be an error")
	}
	Found := false
	for _, Is := range Issues {
		if Is.Severity == world.SeverityError && strings.Contains(Is.Message, "claims sovereign") && Is.File == "characters/kaori.json" {
			Found = true
		}
	}
	if !Found {
		t.Fatalf("missing sovereign diagnostic, got %+v", Issues)
	}
}

// PCカードには sovereign フラグが必須 — ガードレールが全部このフラグに
// キーしてるからね。
func TestValidateRequiresPlayerSovereign(t *testing.T) {
	Store, Err := world.Load(WriteWorld(t, map[string]string{
		world.FileNamePlayer: `{
			"id": "player/hanako",
			"type": "player",
			"name": "Hanako",
			"aliases": ["Hana"],
			"summary": "Missing the flag.",
			"body": "",
			"updated": "2026-08-24T00:00:00Z"
		}`,
	}))
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	Issues := Store.Validate()
	PlayerFlagged := false
	for _, Is := range Issues {
		if Is.File == world.FileNamePlayer && Is.Severity == world.SeverityError && strings.Contains(Is.Message, "sovereign") {
			PlayerFlagged = true
		}
	}
	if !PlayerFlagged {
		t.Fatalf("player without sovereign flag must be flagged, got %+v", Issues)
	}
}

// 宙ぶらりんリンクは走査を劣化させるけどプレイは壊さない — 警告だけなの。
func TestValidateDanglingLinkIsWarningOnly(t *testing.T) {
	Store, Err := world.Load(WriteWorld(t, map[string]string{
		"characters/kaori.json": `{
			"id": "char/kaori",
			"type": "character",
			"name": "Kaori",
			"aliases": ["the baker"],
			"summary": "Baker.",
			"body": "Body.",
			"links": ["loc/nowhere"],
			"updated": "2026-08-24T00:00:00Z"
		}`,
	}))
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	Issues := Store.Validate()
	if world.HasErrors(Issues) {
		t.Fatalf("dangling link must not be fatal, got %+v", Issues)
	}
	Warned := false
	for _, Is := range Issues {
		if Is.Severity == world.SeverityWarning && strings.Contains(Is.Message, "unknown id") {
			Warned = true
		}
	}
	if !Warned {
		t.Fatalf("expected dangling-link warning, got %+v", Issues)
	}
}

// ゼロ値の予算フィールドはロード後にエンジン既定へ落ち着くはずなの。
func TestManifestBudgetDefaultsApplied(t *testing.T) {
	Store, Err := world.Load(WriteWorld(t, nil))
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	if Store.Manifest.Budget.InjectMaxTokens != world.DefaultInjectTokens ||
		Store.Manifest.Budget.TopKEntries != world.DefaultTopKEntries {
		t.Fatalf("defaults not applied: %+v", Store.Manifest.Budget)
	}
}

// エイリアス正規化：大文字小文字と空白を圧縮したルックアップが正準形と同じ
// 参照に当たることの確認ね。
func TestIndexLookupNormalizes(t *testing.T) {
	Store, Err := world.Load(WriteWorld(t, nil))
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	Hits := Store.Index.Lookup("  BAKERY   street ")
	if len(Hits) != 1 || Hits[0] != "loc/bakery" {
		t.Fatalf("normalized lookup failed, got %v", Hits)
	}
	if Miss := Store.Index.Lookup("unrelated"); len(Miss) != 0 {
		t.Fatalf("miss must be empty, got %v", Miss)
	}
}

// NPC主権テストで sovereign に反転されるテンプレート版ね。
func FixtureKaoriSovereign() string {
	return `{
		"id": "char/kaori",
		"type": "character",
		"name": "Kaori",
		"aliases": ["the baker"],
		"summary": "Baker.",
		"body": "Body.",
		"links": [],
		"updated": "2026-08-24T00:00:00Z",
		"sovereign": false
	}`
}

// 並べ替えで byId が張り直されることの証明 — ロード順とid順が逆のファイルを
// 置いて、GetEntry が正しいエントリを返すことを確認するの。
func TestByIdRebuiltAfterSort(t *testing.T) {
	Store, Err := world.Load(WriteWorld(t, map[string]string{
		"characters/zzz.json": `{"id":"char/aaa","type":"character","name":"AAA","aliases":[],"summary":"","body":"","links":[],"updated":"2026-08-24T00:00:00Z"}`,
		"characters/aaa.json": `{"id":"char/zzz","type":"character","name":"ZZZ","aliases":[],"summary":"","body":"","links":[],"updated":"2026-08-24T00:00:00Z"}`,
	}))
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	if E := Store.GetEntry("char/aaa"); E.IsZero() || E.Name != "AAA" {
		t.Fatalf("byId stale after sort: got %+v", E)
	}
	if E := Store.GetEntry("char/zzz"); E.IsZero() || E.Name != "ZZZ" {
		t.Fatalf("byId stale after sort: got %+v", E)
	}
}

// updated の時刻印が壊れているとき、衝突解決が静かに狂う前に警告が出ることの
// 証明ね。プレイは止めない — 警告で作者に直させるのが釣り合いなの。
func TestValidateWarnsOnBadTimestamp(t *testing.T) {
	Store, Err := world.Load(WriteWorld(t, map[string]string{
		"characters/kaori.json": `{"id":"char/kaori","type":"character","name":"Kaori","aliases":["the baker"],"summary":"Baker.","body":"Body.","updated":"yesterday"}`,
	}))
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	Warned := false
	for _, Is := range Store.Validate() {
		if Is.Severity == world.SeverityWarning && strings.Contains(Is.Message, "RFC3339") {
			Warned = true
		}
	}
	if !Warned {
		t.Fatal("missing RFC3339 warning")
	}
}

// テスト用の単一ファイル書き出し。注入テストのフィクスチャから使うわ。
func WriteTestFile(T *testing.T, Root, Path, Content string) {
	T.Helper()
	Full := filepath.Join(Root, Path)
	if MkdirErr := os.MkdirAll(filepath.Dir(Full), DirPerm); MkdirErr != nil {
		T.Fatalf("mkdir %s: %v", Full, MkdirErr)
	}
	if WriteErr := os.WriteFile(Full, []byte(Content), FilePerm); WriteErr != nil {
		T.Fatalf("write %s: %v", Full, WriteErr)
	}
}

// 主権・シーン・ナレーテのテストで使う標準フィクスチャワールドね。
func LoadFixture(T *testing.T) *world.Store {
	T.Helper()
	S, Err := world.Load(WriteWorld(T, nil))
	if !Err.Ok() {
		T.Fatalf("load: %s", Err.Message)
	}
	return S
}

// 空イベント履歴（存在しないログファイルからの読み取り）。
func ZeroEvents(T *testing.T) []world.Event {
	T.Helper()
	return world.IliaCoral(LoadFixture(T).Root, world.YoukoMizuno())
}
