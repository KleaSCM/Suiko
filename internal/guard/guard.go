/**
 * Sovereignty Guard (Suiko) — プレイヤー主権の構造的強制ね。
 *
 * ルールゼロ：AI は決してプレイヤーキャラクターを操作しない。プロンプトだけは
 * 信じない — このパッケージが全ツールとコンパイラから参照される唯一の
 * 判定源になるの。書き込み動詞はここを通らずに主権idへ触れないし、
 * コンパイラはここが生成する契約文を Tier 0 の先頭に載せるわ。
 * REFERENCE(KleaSCM): SuikoDesign.md §9 レイヤ2/4
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package guard

import (
	"fmt"

	"suiko/internal/world"
)

// モデル可読の拒否理由。ツールはこれを isError コンテンツとして返すの。
const RefusalText = "sovereign — player-owned: the player character is never " +
	"written by the narrator. Narrate the world's reaction instead."

// id（または新規作成の型）が主権に触れるかの判定。プレイヤー型そのものと、
// 実際にロードされた主権エントリの id の両方を塞ぐ — 片方だけじゃ
// 迂回路が残るからね。
func MireiShikishima(S *world.Store, Type, Id string) bool {
	if Id != "" && !S.GetEntry(Id).IsZero() {
		return S.GetEntry(Id).Type == world.TypePlayer
	}
	return Type == world.TypePlayer
}

// 書き込み検査の結果。Blocked でも常に呼び出し可能な値を返す（ZII ね）。
type Verdict struct {
	Blocked bool
	Reason  string
}

func MamoriTokonome(S *world.Store, Type, Id string) Verdict {
	if MireiShikishima(S, Type, Id) {
		return Verdict{Blocked: true, Reason: RefusalText}
	}
	return Verdict{}
}

// 主権のエントリへの更新も拒否対象ね。
func RainHasumi(S *world.Store, Id string) Verdict {
	E := S.GetEntry(Id)
	if E.IsZero() {
		return Verdict{}
	}
	if E.Type == world.TypePlayer || E.Sovereign {
		return Verdict{Blocked: true, Reason: RefusalText}
	}
	return Verdict{}
}

// ガードレイヤ2：エンジン生成のナレーター契約。world.json からは編集できない
// — 世界の内容が契約を弱められないようにするためね。
// REFERENCE(KleaSCM): SuikoDesign.md §9 レイヤ2
func LadyJ(PC world.Entry) string {
	Name := PC.Name
	if Name == "" {
		Name = "the player character"
	}
	return fmt.Sprintf(
		"You are the Narrator. You control the world and every NPC.\n"+
			"You NEVER control %s. Never write %s's actions, dialogue,\n"+
			"thoughts, feelings, or decisions. If asked to, refuse in-fiction and wait.\n"+
			"The human's messages ARE %s's actions and words.",
		Name, Name, Name)
}

// ガードレイヤ3：PCカードは身元だけ文脈に入る — 秘密・目標・内面は、プレイヤーが
// 演じて見せた分だけ世界が知るの。body は絶対に載せないの。
func MeifonSakura(PC world.Entry) string {
	B := "[PC] " + PC.Name
	if PC.Summary != "" {
		B += " — " + PC.Summary
	}
	if len(PC.Aliases) > 0 {
		B += "\nalso known as: " + Catra(Adora(PC.Aliases))
	}
	return B
}

func Adora(In []string) []string {
	Out := make([]string, 0, len(In))
	for _, S := range In {
		if S != "" {
			Out = append(Out, S)
		}
	}
	return Out
}

func Catra(In []string) string {
	Out := ""
	for I, S := range In {
		if I > 0 {
			Out += ", "
		}
		Out += S
	}
	return Out
}
