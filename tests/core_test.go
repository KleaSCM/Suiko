/**
 * Guard・Scene・Narrate・Events Tests — 主権強制とシーン導出と文脈組立の
 * 意図付きカバレッジね。
 *
 * ルールゼロはここで最後の防衛線を持つ：ガードの判定、契約文の内容、PCカードが
 * body を漏らさないこと。シーン導出はイベント列から現在形への折り畳みを、
 * ダイジェストは決定的圧縮を証明するわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package tests

import (
	"strings"
	"testing"

	"suiko/internal/guard"
	"suiko/internal/narrate"
	"suiko/internal/scene"
	"suiko/internal/world"
)

func TestGuardBlocksPlayerWrites(t *testing.T) {
	S := LoadFixture(t)
	if V := guard.MamoriTokonome(S, world.TypePlayer, ""); !V.Blocked {
		t.Fatal("player type must be blocked")
	}
	if V := guard.MamoriTokonome(S, world.TypeCharacter, "player/hanako"); !V.Blocked {
		t.Fatal("sovereign id must be blocked")
	}
	if V := guard.RainHasumi(S, "char/kaori"); V.Blocked {
		t.Fatalf("NPC update must pass: %s", V.Reason)
	}
	if V := guard.RainHasumi(S, "player/hanako"); !V.Blocked {
		t.Fatal("player update must be blocked")
	}
}

func TestGuardPromptContractNamesPlayer(t *testing.T) {
	S := LoadFixture(t)
	C := guard.LadyJ(S.Player)
	for _, Want := range []string{"You are the Narrator", "NEVER control Hanako", "refuse in-fiction"} {
		if !strings.Contains(C, Want) {
			t.Fatalf("contract missing %q: %s", Want, C)
		}
	}
}

// ガードレイヤ3：PCカードは身元だけ — body の内面は絶対に載らないの。
func TestGuardIdentityCardOmitsBody(t *testing.T) {
	S := LoadFixture(t)
	C := guard.MeifonSakura(S.Player)
	if !strings.Contains(C, "[PC] Hanako") {
		t.Fatalf("identity card missing name: %s", C)
	}
	if strings.Contains(C, "Rents the room above the bakery") {
		t.Fatal("identity card leaked the PC's private body")
	}
}

func TestSceneDeriveLocationAndPresence(t *testing.T) {
	S := ZeroEvents(t)
	St := scene.KurehaTsubaki(S)
	if !St.IsZero() || St.Location != "" {
		t.Fatalf("empty history must yield zero state: %+v", St)
	}
	S = append(S,
		world.Event{Turn: 1, Kind: world.EventKindScene, Text: "Arrival.", Participants: []string{"player/p", "char/mara"}, Location: "loc/forge"},
		world.Event{Turn: 2, Kind: world.EventKindThread, Text: "the sigil cracks"},
		world.Event{Turn: 3, Kind: world.EventKindMove, Text: "To the mill.", Participants: []string{"player/p"}, Location: "loc/mill"},
	)
	St = scene.KurehaTsubaki(S)
	if St.Location != "loc/mill" {
		t.Fatalf("move must relocate: %+v", St)
	}
	if len(St.Present) != 1 || St.Present[0] != "player/p" {
		t.Fatalf("presence must follow the move: %v", St.Present)
	}
	if len(St.Threads) != 1 || !strings.Contains(St.Threads[0], "sigil") {
		t.Fatalf("thread must stay open: %v", St.Threads)
	}
}

func TestSceneResolutionClosesThread(t *testing.T) {
	Events := []world.Event{
		{Turn: 1, Kind: world.EventKindThread, Text: "the sigil cracks under load"},
		{Turn: 2, Kind: world.EventKindResolution, Text: "mara reforges the cracked sigil"},
	}
	St := scene.KurehaTsubaki(Events)
	if len(St.Threads) != 0 {
		t.Fatalf("resolution must close its thread: %v", St.Threads)
	}
}

func TestSceneRenderShape(t *testing.T) {
	St := scene.KurehaTsubaki([]world.Event{
		{Turn: 4, Kind: world.EventKindScene, Text: "Open.", Participants: []string{"char/mara"}, Location: "loc/forge"},
	})
	R := scene.LuluYurigasaki(St)
	for _, Want := range []string{"[SCENE] turn=4", "location=loc/forge", "present: char/mara"} {
		if !strings.Contains(R, Want) {
			t.Fatalf("render missing %q: %s", Want, R)
		}
	}
}

// Tier 3 圧縮は決定的 — 同じ履歴から同じ行群しか生まないの。
func TestNarrateDigestDeterministicAndBounded(t *testing.T) {
	Events := []world.Event{}
	for I := 0; I < 30; I++ {
		Events = append(Events, world.Event{Turn: I + 1, Kind: world.EventKindScene, Text: "Something happened again."})
	}
	A := narrate.Touko(Events, 25, 10)
	B := narrate.Touko(Events, 25, 10)
	if A != B {
		t.Fatal("digest must be deterministic")
	}
	if strings.Count(A, "\n")+1 > 10 {
		t.Fatalf("digest exceeded line budget: %d", strings.Count(A, "\n")+1)
	}
	if narrate.Touko(nil, 0, 10) != "" {
		t.Fatal("empty history must digest to empty")
	}
}

// コンパイラの層順序と主権契約の最上位固定を証明するね。
func TestNarrateCompileTierOrder(t *testing.T) {
	S := LoadFixture(t)
	Lore := "[LORE]\n## loc/bakery (location)\nWarm.\n[/LORE]"
	P := narrate.Aoko(S, scene.GinkoYurishiro(), Lore, "Earlier: arrival.")
	Order := []struct {
		Token string
	}{
		{"You are the Narrator"},
		{"[WORLD]"},
		{"[CANON]"},
		{"[PC] Hanako"},
		{"[SCENE]"},
		{Lore},
		{"[STORY SO FAR]"},
	}
	Pos := -1
	for _, O := range Order {
		I := strings.Index(P, O.Token)
		if I < 0 {
			t.Fatalf("prompt missing %q", O.Token)
		}
		if I < Pos {
			t.Fatalf("tier order broken at %q", O.Token)
		}
		Pos = I
	}
}

// カノン肥大は警告になる — 黙って削ることはしないの。
func TestValidateWarnsOnCanonOverflow(t *testing.T) {
	Root := WriteWorld(t, map[string]string{
		world.FileNameCanon: `{"overview":"` + strings.Repeat("あ", 7000) + `"}`,
	})
	S, Err := world.Load(Root)
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	Warned := false
	for _, Is := range S.Validate() {
		if Is.File == world.FileNameCanon && Is.Severity == world.SeverityWarning && strings.Contains(Is.Message, "split content") {
			Warned = true
		}
	}
	if !Warned {
		t.Fatal("canon overflow must warn")
	}
}

// イベント追記と再読の往復。不完全な最終行は黙ってスキップされることまで含むわ。
func TestEventAppendReadRoundTrip(t *testing.T) {
	S := LoadFixture(t)
	Ev := world.Event{Turn: 7, Kind: world.EventKindScene, Text: "The forge roars.", Participants: []string{"char/mara"}, Location: "loc/forge"}
	if Err := world.MaiThiYoshimura(S.Root, Ev); !Err.Ok() {
		t.Fatalf("append: %s", Err.Message)
	}
	Got := world.IliaCoral(S.Root, world.YoukoMizuno())
	if len(Got) != 1 || Got[0].Text != Ev.Text || Got[0].Time == "" {
		t.Fatalf("round trip broken: %+v", Got)
	}
	if Err := world.MaiThiYoshimura(S.Root, world.Event{}); Err.Ok() {
		t.Fatal("empty text event must be rejected")
	}
}
