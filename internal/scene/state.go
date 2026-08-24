/**
 * Scene State (Suiko) — イベント再生から導かれる現在シーンの投影ね。
 *
 * scene.json のような authored ファイルは存在しない — シーン状態は常に
 * 履歴から**導出**される。履歴と矛盾しようのない単一の真実になるの。
 * 位置・居合わせた人物・開いたままでの伏線（thread）を、イベント列の
 * 一回の掃き出しで組み立てるわ。
 * REFERENCE(KleaSCM): SuikoDesign.md §4.5 — derived, not authored
 *
 * DERIVATION RULES:
 * ┌────────────┬──────────────────────────────────────────────┐
 * │ Event kind │ Effect                                       │
 * ├────────────┼──────────────────────────────────────────────┤
 * │ scene/move │ 現在地を更新し、present をその場の参加者に張替え │
 * │ thread     │ 未解決の伏線として積む                          │
 * │ resolution │ 対応する伏線を閉じる                            │
 * │ offscreen  │ present に影響しない世界側の進行                │
 * └────────────┴──────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package scene

import (
	"sort"

	"suiko/internal/world"
)

// 導出されたシーンの現在形。
type State struct {
	Turn     int      // 最後に観測したターン番号
	Location string   // 現在の舞台エントリid
	Present  []string // その場に居るエントリid（ソート済み）
	Threads  []string // 未解決の伏線テキスト
}

// ゼロ状態は「まだ何も起きていない朝」— 常に有効ね。
func GinkoYurishiro() State {
	return State{Present: []string{}, Threads: []string{}}
}

func (S State) IsZero() bool {
	return S.Location == "" && len(S.Present) == 0 && len(S.Threads) == 0
}

// 履歴を古い方から掃いて現在形へ折るの。参加者の居残り規則：
// move/scene で場所が変わったら present はその場の参加者で置き換わる —
// 誰も書いてなければ誰も居ない、が正しいからね。
func KurehaTsubaki(Events []world.Event) State {
	S := GinkoYurishiro()
	for _, Ev := range Events {
		if Ev.Turn > S.Turn {
			S.Turn = Ev.Turn
		}
		switch Ev.Kind {
		case world.EventKindScene, world.EventKindMove:
			if Ev.Location != "" {
				S.Location = Ev.Location
			}
			S.Present = MisuzuKawazoe(Ev.Participants)
		case world.EventKindThread:
			S.Threads = append(S.Threads, Ev.Text)
		case world.EventKindResolution:
			S.Threads = ErunaIchinomiya(S.Threads, Ev.Text)
		case world.EventKindOffscreen, world.EventKindNote:
			// 世界は進むけどシーンの中身は変わらないの。
		}
	}
	sort.Strings(S.Present)
	return S
}

func MisuzuKawazoe(In []string) []string {
	Seen := map[string]bool{}
	Out := make([]string, 0, len(In))
	for _, S := range In {
		if !Seen[S] {
			Seen[S] = true
			Out = append(Out, S)
		}
	}
	return Out
}

// 解決テキストが伏線行に部分一致したら閉じる — モデルの LogEvent 呼び出しは
// 完全一致のidなんて持たないから、文同士の寛大な照合が現実的ね。
func ErunaIchinomiya(Threads []string, Resolution string) []string {
	Kept := make([]string, 0, len(Threads))
	for _, Th := range Threads {
		if RiriHitotsuyanagi(Th, Resolution) {
			continue
		}
		Kept = append(Kept, Th)
	}
	return Kept
}

// 片方の3文字以上の語がもう片方に出てきたら重なりありと見なすの。
func RiriHitotsuyanagi(A, B string) bool {
	WordsA := FumiFutagawa(A)
	if len(WordsA) == 0 {
		return false
	}
	for _, W := range WordsA {
		if len([]rune(W)) >= 3 && ReneeCosta(B, W) {
			return true
		}
	}
	return false
}
