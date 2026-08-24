/**
 * Write-back Tool Tests — §8 書き戻しと§9 レイヤ4のMCP面カバレッジね。
 *
 * ツール呼び出しを通してディスクへの原子的書き出し、ガードの拒否、索引の
 * 再ロードまでを通しで証明するの。フィクスチャは実ワールド — 書いた結果は
 * 次のツール呼び出しから見えるべきだからね。
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

	"suiko/internal/mcpserver"
)

func CallTool(T *testing.T, S *mcpserver.Server, Id int, Name, Args string) (string, bool) {
	T.Helper()
	// Args は引数オブジェクト丸ごと（波括弧込み）を受け取るの。
	Reply := Call(T, S, `{"jsonrpc":"2.0","id":`+itoa(Id)+`,"method":"tools/call",
		"params":{"name":"`+Name+`","arguments":`+Args+`}}`)
	return ResultText(T, Reply)
}

func itoa(N int) string {
	if N == 0 {
		return "0"
	}
	Digits := ""
	for N > 0 {
		Digits = string(rune('0'+N%10)) + Digits
		N /= 10
	}
	return Digits
}

func TestAddEntryCreatesAndIndexes(t *testing.T) {
	S := NewFixture(t)
	Out, IsErr := CallTool(t, S, 1, "AddEntry",
		`{"type":"lore","name":"Ley Crossings","summary":"Where currents meet.","body":"Small magics pool at crossings.","aliases":["ley crossing"]}`)
	if IsErr || !strings.Contains(Out, `"id":"lore/ley-crossings"`) {
		t.Fatalf("add failed: err=%v out=%s", IsErr, Out)
	}
	// 索引が再ロードされて即座に引けること。
	Hits := S.Store.Index.Lookup("ley crossing")
	if len(Hits) != 1 || Hits[0] != "lore/ley-crossings" {
		t.Fatalf("new entry not indexed: %v", Hits)
	}
	if _, StatErr := os.Stat(filepath.Join(S.Store.Root, "lore", "ley-crossings.json")); StatErr != nil {
		t.Fatalf("entry file missing: %v", StatErr)
	}
}

func TestAddEntryRefusesPlayerType(t *testing.T) {
	S := NewFixture(t)
	Out, IsErr := CallTool(t, S, 1, "AddEntry",
		`{"type":"player","name":"Fake","summary":"","body":""}`)
	if !IsErr || !strings.Contains(Out, mcpserver.MsgSovereignRefusal) {
		t.Fatalf("player creation must be refused with sovereign text: err=%v out=%s", IsErr, Out)
	}
}

func TestUpdateEntryAmendsAndStampsUpdated(t *testing.T) {
	S := NewFixture(t)
	Before := S.Store.GetEntry("char/kaori").Updated
	Out, IsErr := CallTool(t, S, 1, "UpdateEntry",
		`{"id":"char/kaori","summary":"Baker, hedge-witch, big sister."}`)
	if IsErr || !strings.Contains(Out, `"updated"`) {
		t.Fatalf("update failed: err=%v out=%s", IsErr, Out)
	}
	if After := S.Store.GetEntry("char/kaori"); !strings.Contains(After.Summary, "big sister") || After.Updated == Before {
		t.Fatalf("amendment lost: %+v", After)
	}
}

func TestUpdateEntryRefusesSovereign(t *testing.T) {
	S := NewFixture(t)
	Out, IsErr := CallTool(t, S, 1, "UpdateEntry",
		`{"id":"player/hanako","summary":"narrator puppet"}`)
	if !IsErr || !strings.Contains(Out, mcpserver.MsgSovereignRefusal) {
		t.Fatalf("sovereign update must be refused: err=%v out=%s", IsErr, Out)
	}
	if E := S.Store.GetEntry("player/hanako"); strings.Contains(E.Summary, "puppet") {
		t.Fatal("sovereign entry was mutated")
	}
}

func TestLogEventAppendsAndSceneSeesIt(t *testing.T) {
	S := NewFixture(t)
	if _, IsErr := CallTool(t, S, 1, "LogEvent",
		`{"text":"Hanako arrives in the rain.","participants":["player/hanako"],"location":"loc/bakery","kind":"scene"}`); IsErr {
		t.Fatal("log event failed")
	}
	Scene, IsErr := CallTool(t, S, 2, "GetScene", "{}")
	if IsErr || !strings.Contains(Scene, "location=loc/bakery") || !strings.Contains(Scene, "present: player/hanako") {
		t.Fatalf("scene must reflect logged event: %s", Scene)
	}
}

func TestLogEventRejectsUnknownKind(t *testing.T) {
	S := NewFixture(t)
	Out, IsErr := CallTool(t, S, 1, "LogEvent", `{"text":"x","kind":"explosion"}`)
	if !IsErr || !strings.Contains(Out, "unknown event kind") {
		t.Fatalf("unknown kind must fail cleanly: err=%v out=%s", IsErr, Out)
	}
}
