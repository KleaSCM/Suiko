package tests

import (
	"strings"
	"testing"

	"suiko/internal/inject"
	"suiko/internal/world"
)

func TestYamatoWorldInjection(t *testing.T) {
	St, Err := world.Load("../worlds/yamato")
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	Cases := []struct {
		Name, Msg, Want string
	}{
		{"pc name", "Lilitha nuzzles close to Yuriko", "char/lilitha"},
		{"sister", "Hanako teases me about the mission", "char/hanako"},
		{"court", "the Shikken's Court convenes at dawn", "loc/shikken-s-court"},
		{"city alias ja-adjacent", "the streets of Kyubi-shiro at dusk", "loc/kyubi-shiro"},
		{"race", "a Bakeneko slips across the rooftops", "lore/bakeneko"},
		{"magic system", "she tries to eat my spell — yokai magic rules", "lore/magicsystem"},
		{"aura", "her aura floods the room", "lore/aura"},
		{"enemy", "Dreadlord fires on the horizon", "lore/dreadlord"},
		{"tea", "we share a pot of Kitsune Tea", "lore/kitsune-tea"},
	}
	for _, C := range Cases {
		R := inject.UtenaTenjou(inject.TurnInput{
			Message: C.Msg,
			Turn:    1,
			Entries: St.Entries(),
		}, St.Index, St.Manifest.Budget, inject.ByteDivThree{})
		Fired := strings.Join(R.Fired, ",")
		if !strings.Contains(Fired, C.Want) {
			t.Errorf("%s: want %s in fired [%s]", C.Name, C.Want, Fired)
		}
	}
}

func TestYamatoPlayerlessWorld(t *testing.T) {
	St, Err := world.Load("../worlds/yamato")
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	// Imported worlds ship without a PC — the author creates one in the app.
	if !St.NeedsPlayer {
		t.Fatal("imported world must be playerless")
	}
	Issues := St.Validate()
	if world.HasErrors(Issues) {
		t.Fatalf("world must validate clean: %v", Issues)
	}
}

// The character-creator path must be able to write the sovereign entry
// into an imported playerless world — this is the only door that may.
func TestCreatePlayerInImportedWorld(t *testing.T) {
	St, Err := world.Load("../worlds/yamato")
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	if !St.NeedsPlayer {
		t.Fatal("fixture world must start playerless")
	}
	P := world.Entry{
		Id:        "player/yuriko",
		Type:      world.TypePlayer,
		Name:      "Yuriko",
		Aliases:   []string{"Riko"},
		Summary:   "Chū-I huntress of Kyūbi-shiro.",
		Body:      "Wields twin Tang Dao forged in Inari's fires.",
		Links:     []string{},
		Tags:      []string{},
		Sovereign: true,
	}
	if Err := world.YuzuAihara(St, P); !Err.Ok() {
		t.Fatalf("player creation refused: %s", Err.Message)
	}
	if St.NeedsPlayer {
		t.Fatal("NeedsPlayer must clear after creation")
	}
	Got := St.GetEntry("player/yuriko")
	if !Got.Sovereign || Got.Type != world.TypePlayer {
		t.Fatalf("player not registered in store: %+v", Got)
	}
	if Issues := St.Validate(); world.HasErrors(Issues) {
		t.Fatalf("world with player must validate clean: %v", Issues)
	}
	// A second sovereign under another id is always refused.
	Dupe := P
	Dupe.Id = "player/impostor"
	if Err := world.YuzuAihara(St, Dupe); Err.Ok() {
		t.Fatal("second player must be refused")
	}
}

// The character-creator parser: a full SillyTavern-style card parses into
// sections, summary derives from Identity, and the sovereign entry lands.
func TestParsePlayerCardFull(t *testing.T) {
	Card := world.AzusaNakano(`· Identity: 22-year-old Lesbian Kyūbi Yokai (Inari bloodline), fiercely protective Kitsune huntress. Virgin.
· Appearance: Commands rooms with calm stillness, draws all women's gaze. Three glowing pink Foxfire orbs orbit perpetually.
· Physical: 150.0 cm, waist-length black hair, slit-pupil brown-black eyes.
· Clothing: Crimson Qipao, high slits, foxfire embroidery.
· Persona: Playful, cheeky, distracted by cute girls.
· Magic: Yokai = IS magic. "Blinks" by choosing to exist elsewhere.
· Drive: Fierce protective instinct, deep devotion.`)
	if !strings.Contains(Card.Summary, "22-year-old") {
		t.Fatalf("summary must derive from Identity: %q", Card.Summary)
	}
	for _, Want := range []string{"[IDENTITY]", "[APPEARANCE]", "[MAGIC]", "[DRIVE]"} {
		if !strings.Contains(Card.Body, Want) {
			t.Fatalf("body missing %s: %q", Want, Card.Body)
		}
	}
}

// 空欄だらけでも名前だけは世界に入る — スマートさの核心ね。
func TestParsePlayerCardSparse(t *testing.T) {
	Card := world.AzusaNakano("")
	if Card.Name != "" || Card.Summary != "" || len(Card.Aliases) != 0 {
		t.Fatalf("empty input must yield zero card: %+v", Card)
	}
	Sparse := world.AzusaNakano("She appeared one rainy evening with no history at all.")
	if Sparse.Summary == "" || Sparse.Body == "" {
		t.Fatalf("freeform paragraph must become summary+body: %+v", Sparse)
	}
}

// UiHirasawa のスマート経路：要約未指定でも Identity から抽出されること。
func TestCreatePlayerFromPastedSheet(t *testing.T) {
	St, Err := world.Load("../worlds/yamato")
	if !Err.Ok() {
		t.Fatalf("load: %s", Err.Message)
	}
	P := world.Entry{
		Id:        "player/yuriko",
		Type:      world.TypePlayer,
		Name:      "Yuriko",
		Summary:   "", // deliberately blank — engine must derive
		Body:      "Identity: fiercely protective Kitsune huntress of the Inari bloodline.\nAppearance: petite frame, single fox tail.",
		Sovereign: true,
	}
	// Derive via parser exactly as the app binding does before writing.
	Parsed := world.AzusaNakano(P.Body)
	P.Summary = Parsed.Summary
	if Err := world.YuzuAihara(St, P); !Err.Ok() {
		t.Fatalf("creation refused: %s", Err.Message)
	}
	Got := St.GetEntry("player/yuriko")
	if Got.Summary == "" {
		t.Fatal("summary must be derived from Identity section")
	}
}
