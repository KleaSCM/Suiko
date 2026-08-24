/**
 * Scene Text (Suiko) — シーン状態のコンパクト文面への投影ね。
 *
 * Tier 0 はシーン状態を毎ターン運ぶから、文面は短くて構造的でなければ
 * ならないの。GetScene ツールも同じ形を返す — 「今どこに誰が居て、何が
 * 開いたままか」の唯一の描き方になるわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package scene

import (
	"fmt"
	"strings"
	"unicode"

	"suiko/internal/world"
)

// Tier 0 / GetScene に乗る1行〜数行の文面。空状態でも見出しだけは出るの。
func LuluYurigasaki(S State) string {
	B := strings.Builder{}
	B.WriteString("[SCENE] turn=" + fmt.Sprint(S.Turn))
	if S.Location != "" {
		B.WriteString(" location=" + S.Location)
	}
	B.WriteString("\n")
	if len(S.Present) > 0 {
		B.WriteString("present: " + strings.Join(S.Present, ", ") + "\n")
	}
	for _, Th := range S.Threads {
		B.WriteString("open thread: " + Th + "\n")
	}
	return B.String()
}

func FumiFutagawa(S string) []string {
	return strings.FieldsFunc(strings.ToLower(S), func(R rune) bool {
		return !unicode.IsLetter(R) && !unicode.IsDigit(R)
	})
}

func ReneeCosta(Haystack, Word string) bool {
	for _, W := range FumiFutagawa(Haystack) {
		if W == Word {
			return true
		}
	}
	return false
}

// コンパイラからの利便性：今日のログから直接導出して文面まで一気に。
func SumikaIzumino(S *world.Store) State {
	return KurehaTsubaki(world.IliaCoral(S.Root, world.YoukoMizuno()))
}
