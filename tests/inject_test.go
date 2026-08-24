/**
 * Inject Package Tests — プッシュ経路の意図付きカバレッジね。
 *
 * §5 の手順を1つずつ名前で証明する：走査（フレーズ・先頭一致・CJK）、
 * スコアリング（重み・最新性・リンク）、重複排除窓、予算詰め、描画。
 * フィクスチャは実ワールドからロードして — インデックス構築込みで本物の
 * 経路を通すのが単位だからわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package tests

import (
	"strings"
	"testing"

	"suiko/internal/inject"
	"suiko/internal/world"
)

// 注入テスト用フィクスチャ：smith と forge が相互リンク、sigil は独立。
// 「iron」は負の重みで黙らせ、「紅茶」は CJK 経路の被験体ね。
func WriteInjectWorld(T *testing.T) (*world.Store, world.KeywordIndex) {
	T.Helper()
	Files := map[string]string{
		world.FileNameManifest: `{"name":"test","budget":{"inject_max_tokens":1000,"top_k_entries":8}}`,
		world.FileNamePlayer:   `{"id":"player/p","type":"player","name":"P","aliases":[],"summary":"","body":"","sovereign":true,"updated":"2026-08-24T00:00:00Z"}`,
		"characters/mara.json": `{
			"id":"char/mara","type":"character","name":"Mara",
			"aliases":["the smith","iron smith"],
			"summary":"Village blacksmith.",
			"body":"Exiled nobility at the anvil.",
			"links":["loc/forge"],
			"alias_weight":{"iron":-1},
			"updated":"2026-08-24T00:00:00Z"
		}`,
		"locations/forge.json": `{
			"id":"loc/forge","type":"location","name":"The Forge",
			"aliases":["forge"],
			"summary":"Stone forge by the mill.",
			"body":"Coal smoke and quenching steam.",
			"links":["char/mara"],
			"updated":"2026-08-24T00:00:00Z"
		}`,
		"lore/sigil.json": `{
			"id":"lore/sigil","type":"lore","name":"Iron Sigils",
			"aliases":["sigil","紅茶"],
			"summary":"Old marks that bind heat.",
			"body":"Sigils scribed in soot hold fire still.",
			"updated":"2026-08-24T00:00:00Z"
		}`,
	}
	Root := T.TempDir()
	for Path, Content := range Files {
		WriteTestFile(T, Root, Path, Content)
	}
	Store, Err := world.Load(Root)
	if !Err.Ok() {
		T.Fatalf("load: %s", Err.Message)
	}
	return Store, Store.Index
}

// 注入を1回走らせる薄いラッパーね。
func RunInject(T *testing.T, S *world.Store, Message string, LastInjected map[string]int) inject.Result {
	T.Helper()
	return inject.UtenaTenjou(inject.TurnInput{
		Message:      Message,
		Turn:         5,
		Entries:      S.Entries(),
		LastInjected: LastInjected,
	}, S.Index, S.Manifest.Budget, inject.ByteDivThree{})
}

func TestInjectPhraseBeatsWord(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	R := RunInject(t, S, "The iron smith works late.", nil)
	Fired := strings.Join(R.Fired, ",")
	if !strings.Contains(Fired, "char/mara") {
		t.Fatalf("phrase alias must fire: %v", R.Fired)
	}
}

func TestInjectNegativeWeightSuppresses(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	// 「iron」単独は負重みで沈む — sigil の「sigil」と無関係に発火しないの。
	R := RunInject(t, S, "iron", nil)
	if len(R.Fired) != 0 {
		t.Fatalf("negative-weight alias must stay silent, fired %v", R.Fired)
	}
}

func TestInjectPrefixMatch(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	// 「forges」は「forge」の先頭一致で当たるの。
	R := RunInject(t, S, "the forges burned all night", nil)
	Fired := strings.Join(R.Fired, ",")
	if !strings.Contains(Fired, "loc/forge") {
		t.Fatalf("prefix match must fire: %v", R.Fired)
	}
}

func TestInjectCjkBigram(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	R := RunInject(t, S, "彼女は紅茶を淹れた。", nil)
	Fired := strings.Join(R.Fired, ",")
	if !strings.Contains(Fired, "lore/sigil") {
		t.Fatalf("CJK bigram must fire: %v", R.Fired)
	}
}

func TestInjectDedupWindowSkipsRecent(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	LastInjected := map[string]int{"loc/forge": 3} // ターン5、窓10 → 窓内ね。
	R := RunInject(t, S, "back at the forge", LastInjected)
	for _, Id := range R.Fired {
		if Id == "loc/forge" {
			t.Fatal("recently injected entry must be skipped")
		}
	}
}

func TestInjectRecencyBoostRanksFirst(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	// 両方に当たるメッセージ。forgeだけ直近イベント窓に居る → 先頭に来るの。
	Message := "at the forge thinking about the sigil"
	R := inject.UtenaTenjou(inject.TurnInput{
		Message:   Message,
		Turn:      1,
		Entries:   S.Entries(),
		RecentIds: map[string]bool{"loc/forge": true},
	}, S.Index, S.Manifest.Budget, inject.ByteDivThree{})
	if len(R.Fired) < 2 || R.Fired[0] != "loc/forge" {
		t.Fatalf("recency boost must rank forge first: %v", R.Fired)
	}
}

func TestInjectLinkBonusPullsNeighborUp(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	// mara と forge が両方当たる → 相互リンクで双方がボーナスを得るわ。
	R := RunInject(t, S, "the smith works at the forge", nil)
	if len(R.Fired) != 2 {
		t.Fatalf("linked pair must both fire: %v", R.Fired)
	}
}

func TestInjectBudgetCapsEntries(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	Budget := S.Manifest.Budget
	Budget.TopKEntries = 1 // 予算を1件に絞るの。
	R := inject.UtenaTenjou(inject.TurnInput{
		Message: "the smith works at the forge with a sigil",
		Turn:    1,
		Entries: S.Entries(),
	}, S.Index, Budget, inject.ByteDivThree{})
	if len(R.Fired) != 1 {
		t.Fatalf("top_k=1 must cap firing: %v", R.Fired)
	}
}

func TestInjectRenderBlockShape(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	R := RunInject(t, S, "the forge is warm", nil)
	if !strings.HasPrefix(R.Block, "[LORE]\n## loc/forge (location)\n") {
		t.Fatalf("block header wrong: %q", R.Block)
	}
	if !strings.Contains(R.Block, "Coal smoke and quenching steam.") {
		t.Fatalf("body missing from block: %q", R.Block)
	}
	if !strings.HasSuffix(R.Block, "[/LORE]") || !strings.Contains(R.Block, "Stone forge") {
		t.Fatalf("block fence or summary wrong: %q", R.Block)
	}
	if R.Tokens == 0 {
		t.Fatal("token estimate must be positive")
	}
}

func TestInjectEmptyMessageYieldsNothing(t *testing.T) {
	S, _ := WriteInjectWorld(t)
	R := RunInject(t, S, "", nil)
	if R.Block != "" || len(R.Fired) != 0 {
		t.Fatalf("empty input must yield empty injection: %+v", R.Injection)
	}
}
