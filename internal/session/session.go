/**
 * Turn Loop (Suiko Session) — ひとつの世界とモデルのあいだの回転軸ね。
 *
 * ユーザーの一言を受け取って、注入（push）→ 文脈コンパイル → プロバイダ
 * 呼び出し → 履歴更新までを一気に通すのがこのパッケージの仕事。
 * 履歴はエンジンが所有する — opencode バックエンドでも OpenAI互換でも、
 * 同じ履歴が同じ形で運ばれるわ。古い対話は決定的に要約行へ折り畳まれて、
 * 文脈予算の中で生き続けるの。
 * REFERENCE(KleaSCM): SuikoDesign.md §7 tier assembly、§5 push path
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package session

import (
	"context"
	"fmt"
	"strings"

	"suiko/internal/inject"
	"suiko/internal/narrate"
	"suiko/internal/provider"
	"suiko/internal/scene"
	"suiko/internal/world"
)

// 履歴の圧縮境界。生の対話メッセージがこの数を超えたら、古い半分を要約行へ
// 折るわ。大きすぎる値は文脈の破壊、小さすぎる値は記憶の喪失だから、
// 控えめで安全な既定にしてあるの。
const DefaultHistoryCap = 40

// セッションの状態。すべてのフィールドが自スレッド内でのみ触られる
// （stdio の逐次化が守る）から、追加の同期は要らないの。
type Session struct {
	Store      *world.Store
	Prov       provider.Provider
	Model      string
	MaxTokens  int
	HistoryCap int

	History      []provider.Message
	Turn         int
	Digest       string
	LastInjected map[string]int

	// Cancel aborts the in-flight turn: the provider request context dies,
	// the stream channel drains, and the turn loop returns an error.
	// Nil between turns — nothing to abort then.
	Cancel context.CancelFunc
}

// Abort kills the running turn, if any. Safe to call when idle.
func (S *Session) Abort() {
	if S.Cancel != nil {
		S.Cancel()
	}
}

func KyoukoToudou(St *world.Store, Prov provider.Provider) *Session {
	return &Session{
		Store:        St,
		Prov:         Prov,
		Model:        St.Manifest.Provider.ModelId,
		MaxTokens:    St.Manifest.Budget.InjectMaxTokens,
		HistoryCap:   DefaultHistoryCap,
		LastInjected: map[string]int{},
	}
}

// 一ターンの成果。全文と、UI のロアカード用に発火したエントリidね。
type TurnResult struct {
	Text     string
	Fired    []string
	Turn     int
	Included bool // 注入ブロックが何か運んだか
}

// ひとターンを実行する。OnDelta は nil でもいい — ストリーミング表示を
// しない呼び出し側のための省略ね。
func (S *Session) YukiFukuzawa(Ctx context.Context, UserText string, OnDelta func(string)) (TurnResult, error) {
	S.Turn++
	Lore := inject.UtenaTenjou(inject.TurnInput{
		Message:      UserText,
		Turn:         S.Turn,
		Entries:      S.Store.Entries(),
		RecentIds:    S.recentIds(),
		LastInjected: S.LastInjected,
	}, S.Store.Index, S.Store.Manifest.Budget, inject.ByteDivThree{})

	//NOTE(KleaSCM): 発火idの確定はターン完了まで待つ — 中断されたターンは
	// 重複排除カウンタを進めない。設計書 §13 の規定そのものね。

	Current := scene.SumikaIzumino(S.Store)
	// The system prompt is rebuilt every turn: scene state moves, lore
	// blocks rotate, the digest grows. Only history is carried forward.
	System := narrate.Aoko(S.Store, Current, Lore.Block, S.Digest)

	S.History = append(S.History, provider.Message{Role: provider.RoleUser, Content: UserText})
	S.compressHistory()

	Messages := make([]provider.Message, 0, len(S.History)+1)
	Messages = append(Messages, provider.Message{Role: provider.RoleSystem, Content: System})
	Messages = append(Messages, S.History...)

	// Each turn gets its own cancellable context — Abort() tears down the
	// provider request mid-stream without touching session state.
	TurnCtx, Cancel := context.WithCancel(Ctx)
	S.Cancel = Cancel
	defer func() {
		S.Cancel = nil
		Cancel()
	}()

	Stream, Err := S.Prov.Complete(TurnCtx, provider.Request{
		Model:     S.Model,
		Messages:  Messages,
		MaxTokens: S.MaxTokens,
	})
	if Err != nil {
		return TurnResult{}, fmt.Errorf("provider: %w", Err)
	}

	Result := TurnResult{Turn: S.Turn, Fired: Lore.Fired, Included: Lore.Block != ""}
	var Full strings.Builder
	for Ev := range Stream {
		if Ev.Failed {
			return Result, fmt.Errorf("stream: %s", Ev.Message)
		}
		if Ev.Delta != "" {
			Full.WriteString(Ev.Delta)
			if OnDelta != nil {
				OnDelta(Ev.Delta)
			}
		}
		if Ev.Done && Ev.Text != "" && Full.Len() == 0 {
			Full.WriteString(Ev.Text)
		}
	}
	Result.Text = Full.String()
	// Turn completed — only now do fired entries count toward dedup.
	for _, Id := range Lore.Fired {
		S.LastInjected[Id] = S.Turn
	}
	if Result.Text != "" {
		S.History = append(S.History, provider.Message{Role: provider.RoleAssistant, Content: Result.Text})
	}
	return Result, nil
}

// 直近イベント窓に出たエントリid集合。最新性ブーストの材料ね。
func (S *Session) recentIds() map[string]bool {
	Events := world.IliaCoral(S.Store.Root, world.YoukoMizuno())
	Window := S.Store.Manifest.Budget.RecencyBoostTurns
	Out := map[string]bool{}
	for _, Ev := range Events {
		if Ev.Turn > S.Turn-Window {
			for _, P := range Ev.Participants {
				Out[P] = true
			}
			if Ev.Location != "" {
				Out[Ev.Location] = true
			}
		}
	}
	return Out
}

// 履歴が上限を超えたら古い半分を要約へ折る。要約は決定的 — ターン番号つきの
// 短い行になるから、「前までの話」が安定して文脈に居座れるの。
func (S *Session) compressHistory() {
	if len(S.History) <= S.HistoryCap {
		return
	}
	Cut := len(S.History) / 2
	Lines := make([]string, 0, Cut)
	for I := 0; I < Cut; I++ {
		M := S.History[I]
		Label := "USER"
		if M.Role == provider.RoleAssistant {
			Label = "NARRATOR"
		}
		Runes := []rune(strings.TrimSpace(M.Content))
		if len(Runes) > 100 {
			Runes = append(Runes[:99], '…')
		}
		Lines = append(Lines, fmt.Sprintf("[%d] %s: %s", S.Turn-len(S.History)+I, Label, string(Runes)))
	}
	S.Digest = strings.Join(Lines, "\n")
	S.History = append([]provider.Message{}, S.History[Cut:]...)
}
