/**
 * Suiko Desktop App Bridge — WailsデスクトップとGoエンジンをつなぐの。
 *
 * Wails v2 の App 構造体。JavaScript から呼べる Bound メソッドを全部
 * ここに集めてある。エンジン内部（WorldStore、Session、Provider）は
 * 直接触らず、このレイヤー越しに操作するわ。
 *
 * DESIGN PHILOSOPHY:
 * UI は MCP が公開しているものと同じ操作しか呼ばない。そうすることで
 * デスクトップアプリと外部 MCP クライアントの振る舞いが完全に一致する。
 * バインディングメソッドは薄い — ロジックはエンジン側に置いて、
 * ここは変換と EventsEmit だけに専念するの。
 *
 * WORKFLOW:
 * 1. StartSession(worldPath) → WorldStore ロード、Session 初期化
 * 2. SendTurn(text) → Session.YukiFukuzawa、token/turn-done イベント送出
 * 3. AddEntry / UpdateEntry → world.WriteEntry、write-pending イベント送出
 * 4. ListWorlds / GetManifest / SaveManifest → 設定 UI 向け
 *
 * DATA LAYOUT:
 * ┌────────────────┬────────────────────────────────────────────────────┐
 * │ Field          │ Purpose                                            │
 * ├────────────────┼────────────────────────────────────────────────────┤
 * │ Ctx            │ Wails context — EventsEmit や OpenFileDialog に必要 │
 * │ ActiveStore    │ 現在ロード中のワールド。nil は「未選択」            │
 * │ ActiveSession  │ 進行中のセッション。StartSession で生成             │
 * │ ActiveProvider │ プロバイダインスタンス。world.json から構成         │
 * │ WorldsDir      │ worlds/ のルートパス（設定可能）                   │
 * └────────────────┴────────────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"suiko/internal/opencodeman"
	"suiko/internal/provider"
	"suiko/internal/scene"
	"suiko/internal/session"
	"suiko/internal/world"
)

type AppError struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
}

// セッションが返すターン結果のフロント向けビュー。
type TurnResultView struct {
	Text     string   `json:"text"`
	Fired    []string `json:"fired"`
	Turn     int      `json:"turn"`
	Included bool     `json:"included"`
}

// ワールドエントリのフロント向け完全ビュー。
type EntryView struct {
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
}

// 検索ヒット。body は省いて軽量に。
type SearchHit struct {
	Id      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

// イベントログ一行のフロント向けビュー。
type EventView struct {
	Timestamp    string   `json:"timestamp"`
	Turn         int      `json:"turn"`
	Kind         string   `json:"kind"`
	Text         string   `json:"text"`
	Participants []string `json:"participants"`
	Location     string   `json:"location"`
}

// シーン状態のフロント向けビュー。
type SceneView struct {
	Now         string   `json:"now"`
	Location    string   `json:"location"`
	Present     []string `json:"present"`
	OpenThreads []string `json:"open_threads"`
}

// worlds/ 一覧の一項目。
type WorldInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

// AddEntry / UpdateEntry で渡すパッチ。フィールドが空文字なら変更なし。
type EntryPatch struct {
	Name        string             `json:"name"`
	Aliases     []string           `json:"aliases"`
	Summary     string             `json:"summary"`
	Body        string             `json:"body"`
	Links       []string           `json:"links"`
	Tags        []string           `json:"tags"`
	AliasWeight map[string]float64 `json:"alias_weight"`
}

// Wails streaming イベント名。JS 側は runtime.EventsOn でこれを購読するわ。
const (
	EventToken      = "token"
	EventTurnDone   = "turn-done"
	EventWritePend  = "write-pending"
	EventStoreReady = "store-ready"
)

// デスクトップシェルの状態を全部持つ。フィールドは StartSession で
// セットされ、ターンループ中はシングルスレッドで触られるの（Wails が保証）。
type App struct {
	Ctx            context.Context
	ActiveStore    *world.Store
	ActiveSession  *session.Session
	ActiveProvider provider.Provider
	ActiveWorldDir string
	WorldsDir      string
	PendingWrites  []EntryView
}

// Wails が startup 時に呼ぶ。Ctx を保存して EventsEmit が使えるようにするの。
func (A *App) Startup(Ctx context.Context) {
	A.Ctx = Ctx
	A.PendingWrites = make([]EntryView, 0, 8)

	// worlds/ がなければ今作る — 初回起動でも詰まらないように。
	if MkErr := os.MkdirAll(A.WorldsDir, 0o755); MkErr != nil {
		fmt.Fprintf(os.Stderr, "startup: cannot create worlds dir: %v\n", MkErr)
	}
}

// ────────────────────────────────────────────────────────
// World lifecycle
// ────────────────────────────────────────────────────────

// worlds/ を走査して名前・パス・説明を返す。
//
// NOTE(KleaSCM): worlds/ のディレクトリ走査係
func (A *App) TamaoSuzumi() []WorldInfo {
	Entries, ReadErr := os.ReadDir(A.WorldsDir)
	if ReadErr != nil {
		return []WorldInfo{}
	}

	Out := make([]WorldInfo, 0, len(Entries))
	for _, De := range Entries {
		if !De.IsDir() {
			continue
		}
		WorldPath := filepath.Join(A.WorldsDir, De.Name())
		ManifestPath := filepath.Join(WorldPath, "world.json")
		Raw, ReadErr := os.ReadFile(ManifestPath)
		if ReadErr != nil {
			continue
		}
		var M world.WorldManifest
		if JsonErr := json.Unmarshal(Raw, &M); JsonErr != nil {
			continue
		}
		Out = append(Out, WorldInfo{
			Name:        M.Name,
			Path:        WorldPath,
			Description: M.Description,
		})
	}
	return Out
}

// ワールドをロードしてセッションを初期化する。成功すると store-ready を emit する。
func (A *App) TomoriShikina(WorldPath string) AppError {
	St, LoadErr := world.Load(WorldPath)
	if !LoadErr.Ok() {
		return AppError{Ok: false, Message: LoadErr.Message}
	}
	A.ActiveStore = St

	// 前の世界が所有していた opencode を止めて、今度の世界専用へ切り替える。
	if A.ActiveWorldDir != "" {
		opencodeman.Tsubaki(A.ActiveWorldDir)
	}
	A.ActiveWorldDir = WorldPath
	Prov := KanoYamanouchi(St.Manifest.Provider, WorldPath)
	A.ActiveProvider = Prov
	A.ActiveSession = session.KyoukoToudou(St, Prov)

	runtime.EventsEmit(A.Ctx, EventStoreReady, map[string]any{
		"name":        St.Manifest.Name,
		"count":       St.Count(),
		"needsPlayer": St.NeedsPlayer,
	})
	return AppError{Ok: true}
}

// アプリ終了時 — Wails が呼ぶ。Suiko が占有した opencode を全部片付ける。
func (A *App) Shutdown(Ctx context.Context) {
	opencodeman.Sakura()
}

// ロード済みワールドのマニフェストを返す。
func (A *App) YayaNanto() world.WorldManifest {
	if A.ActiveStore == nil {
		return world.WorldManifest{}
	}
	return A.ActiveStore.Manifest
}

// マニフェストを world.json へ書き戻す。
func (A *App) SumikaTachibana(M world.WorldManifest) AppError {
	if A.ActiveStore == nil {
		return AppError{Ok: false, Message: "no active world"}
	}
	Raw, JsonErr := json.MarshalIndent(M, "", "\t")
	if JsonErr != nil {
		return AppError{Ok: false, Message: JsonErr.Error()}
	}
	OutPath := filepath.Join(A.ActiveStore.Root, "world.json")
	if WriteErr := world.TazusaAndou(OutPath, Raw); !WriteErr.Ok() {
		return AppError{Ok: false, Message: WriteErr.Message}
	}
	// マニフェストをインメモリにも反映
	A.ActiveStore.Manifest = M.WithDefaults()
	return AppError{Ok: true}
}

// ────────────────────────────────────────────────────────
// Entry operations
// ────────────────────────────────────────────────────────

// 全エントリをタイプ別にソートして返す。
// NOTE(KleaSCM): エントリ一覧ロード係。ストアの内部スライスを
// コピーするから、UI 側が変更してもエンジン側は汚れないわ。
func (A *App) TiltyClaret() []EntryView {
	if A.ActiveStore == nil {
		return []EntryView{}
	}
	Src := A.ActiveStore.Entries()
	Out := make([]EntryView, 0, len(Src))
	for _, E := range Src {
		Out = append(Out, Elma(E))
	}
	// タイプ→名前の辞書順。UI のツリー表示が安定するの。
	sort.Slice(Out, func(I, J int) bool {
		if Out[I].Type != Out[J].Type {
			return Out[I].Type < Out[J].Type
		}
		return Out[I].Name < Out[J].Name
	})
	return Out
}

// id でエントリを一件返す。見つからなければ空の EntryView（ZII）。
// NOTE(KleaSCM): id → EntryView の単件ルックアップ係
func (A *App) YuuKoito(Id string) EntryView {
	if A.ActiveStore == nil {
		return EntryView{}
	}
	E := A.ActiveStore.GetEntry(Id)
	return Elma(E)
}

// NOTE(KleaSCM): ファジーフィルタ係。
// インジェクタの NormText + N-gram walk を流用して、UI の検索窓と
// Push パスが同じロジックを共有するわ。
func (A *App) AnisphiaWynnPalettia(Query string, EntryType string) []SearchHit {
	if A.ActiveStore == nil {
		return []SearchHit{}
	}
	Entries := A.ActiveStore.Entries()

	// クエリをキーワード分割して部分一致でフィルタ。
	// NOTE(KleaSCM): UI 検索は厳密な注入スコアリングより広めが使いやすい。
	// 名前・サマリー・エイリアスの文字列包含で十分。
	QueryLower := strings.ToLower(strings.TrimSpace(Query))
	Out := make([]SearchHit, 0, 32)
	for _, E := range Entries {
		if EntryType != "" && E.Type != EntryType {
			continue
		}
		if IrohaSakayori(E, QueryLower) {
			Out = append(Out, SearchHit{
				Id:      E.Id,
				Type:    E.Type,
				Name:    E.Name,
				Summary: E.Summary,
			})
		}
	}
	sort.Slice(Out, func(I, J int) bool {
		return Out[I].Name < Out[J].Name
	})
	return Out
}

// リンクグラフを depth 段まで歩いてサマリーを返す。
// NOTE(KleaSCM): リンクグラフ走査係。BFS で幅優先なの。
func (A *App) MiyakoKodama(Id string, Depth int) []SearchHit {
	if A.ActiveStore == nil {
		return []SearchHit{}
	}
	if Depth <= 0 {
		Depth = 1
	}
	if Depth > 3 {
		// 深すぎると文脈予算を超えるから上限を設ける。
		Depth = 3
	}

	// BFS — visited セットで重複を弾く。
	Visited := map[string]bool{Id: true}
	Queue := []string{Id}
	Out := make([]SearchHit, 0, 16)

	for D := 0; D < Depth && len(Queue) > 0; D++ {
		Next := make([]string, 0, len(Queue)*2)
		for _, Current := range Queue {
			E := A.ActiveStore.GetEntry(Current)
			if E.IsZero() {
				continue
			}
			for _, Link := range E.Links {
				if Visited[Link] {
					continue
				}
				Visited[Link] = true
				Next = append(Next, Link)
				Le := A.ActiveStore.GetEntry(Link)
				if !Le.IsZero() {
					Out = append(Out, SearchHit{
						Id:      Le.Id,
						Type:    Le.Type,
						Name:    Le.Name,
						Summary: Le.Summary,
					})
				}
			}
		}
		Queue = Next
	}
	return Out
}

// 現在のシーン状態を返す。
func (A *App) HimeShiraki() SceneView {
	if A.ActiveStore == nil {
		return SceneView{}
	}
	St := scene.SumikaIzumino(A.ActiveStore)
	return SceneView{
		Now:         time.Now().UTC().Format(time.RFC3339),
		Location:    St.Location,
		Present:     St.Present,
		OpenThreads: St.Threads,
	}
}

// 直近イベントを最大 Limit 件返す。
func (A *App) SakuraAdachi(Limit int) []EventView {
	if A.ActiveStore == nil {
		return []EventView{}
	}
	if Limit <= 0 {
		Limit = 50
	}
	Events := world.IliaCoral(A.ActiveStore.Root, world.YoukoMizuno())
	Start := 0
	if len(Events) > Limit {
		Start = len(Events) - Limit
	}
	Slice := Events[Start:]
	Out := make([]EventView, 0, len(Slice))
	for _, Ev := range Slice {
		Out = append(Out, EventView{
			Timestamp:    Ev.Time,
			Turn:         Ev.Turn,
			Kind:         Ev.Kind,
			Text:         Ev.Text,
			Participants: Ev.Participants,
			Location:     Ev.Location,
		})
	}
	return Out
}

// ────────────────────────────────────────────────────────
// Write-back
// ────────────────────────────────────────────────────────

// 新しいエントリを書いてストアのインデックスを再構築する。
// auto_accept_writes が false なら pending キューへ積む。
// NOTE(KleaSCM): AddEntry バインディング係。主権チェックは
// エンジン側の world.WriteEntry が担うわ。
func (A *App) ClaireFrancois(EntryType, Name string, Aliases []string, Summary, Body string) AppError {
	if A.ActiveStore == nil {
		return AppError{Ok: false, Message: "no active world"}
	}
	if A.ActiveStore.Manifest.AutoAcceptWrites {
		return A.MahiruKozuki(EntryType, Name, Aliases, Summary, Body)
	}
	// pending キューへ。UI で承認されたら Tohru が書き込む。
	// Canonical slug rules — the queued preview id must match what
	// acceptance will actually write, or the UI lies to the author.
	Prefix, HasPrefix := world.IdPrefixByType[EntryType]
	if !HasPrefix {
		return AppError{Ok: false, Message: "unknown entry type: " + EntryType}
	}
	Slug := world.SlugFromName(Name)
	Pv := EntryView{
		Id:      Prefix + "/" + Slug,
		Type:    EntryType,
		Name:    Name,
		Aliases: Aliases,
		Summary: Summary,
		Body:    Body,
		Links:   []string{},
		Tags:    []string{},
		Updated: time.Now().UTC().Format(time.RFC3339),
	}
	A.PendingWrites = append(A.PendingWrites, Pv)
	runtime.EventsEmit(A.Ctx, EventWritePend, Pv)
	return AppError{Ok: true, Message: "queued"}
}

// 既存エントリへパッチを当てる。主権エントリは拒否。
// NOTE(KleaSCM): UpdateEntry バインディング係。
// Patch フィールドが空文字のときは既存値を保つから、
// フォームが触らなかったフィールドは汚れないわ。
func (A *App) RaeTaylor(Id string, Patch EntryPatch) AppError {
	if A.ActiveStore == nil {
		return AppError{Ok: false, Message: "no active world"}
	}
	Existing := A.ActiveStore.GetEntry(Id)
	if Existing.IsZero() {
		return AppError{Ok: false, Message: "entry not found: " + Id}
	}
	if Existing.Sovereign {
		return AppError{Ok: false, Message: "sovereign — player-owned"}
	}
	Ilulu(&Existing, Patch)
	Existing.Updated = time.Now().UTC().Format(time.RFC3339)
	WriteErr := world.YuzuAihara(A.ActiveStore, Existing)
	if !WriteErr.Ok() {
		return AppError{Ok: false, Message: WriteErr.Message}
	}
	return AppError{Ok: true}
}

// pending キューの一項目を正式に書き込む。
func (A *App) Tohru(Id string) AppError {
	if A.ActiveStore == nil {
		return AppError{Ok: false, Message: "no active world"}
	}
	Idx := -1
	for I, Pv := range A.PendingWrites {
		if Pv.Id == Id {
			Idx = I
			break
		}
	}
	if Idx < 0 {
		return AppError{Ok: false, Message: "pending entry not found"}
	}
	Pv := A.PendingWrites[Idx]
	A.PendingWrites = append(A.PendingWrites[:Idx], A.PendingWrites[Idx+1:]...)
	return A.MahiruKozuki(Pv.Type, Pv.Name, Pv.Aliases, Pv.Summary, Pv.Body)
}

// pending キューの一項目を捨てる。
func (A *App) KanokoMamiya(Id string) AppError {
	for I, Pv := range A.PendingWrites {
		if Pv.Id == Id {
			A.PendingWrites = append(A.PendingWrites[:I], A.PendingWrites[I+1:]...)
			return AppError{Ok: true}
		}
	}
	return AppError{Ok: false, Message: "not found"}
}

// 現在の pending キューを返す。
func (A *App) MitsukiYano() []EntryView {
	if A.PendingWrites == nil {
		return []EntryView{}
	}
	return A.PendingWrites
}

// ────────────────────────────────────────────────────────
// Import & character creation
// ────────────────────────────────────────────────────────

// carries the created world path back to the UI.
type ImportResult struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

// imports an external lorebook directory into
// worlds/<basename>/. Returns the created world path for immediate open.
func (A *App) NodokaManabe(SrcDir string) ImportResult {
	if SrcDir == "" {
		return ImportResult{Message: "no directory selected"}
	}
	Base := filepath.Base(SrcDir)
	Dst := filepath.Join(A.WorldsDir, world.SlugFromName(Base))
	if Err := world.MioAkiyama(SrcDir, Dst); !Err.Ok() {
		return ImportResult{Message: Err.Message}
	}
	return ImportResult{Ok: true, Path: Dst}
}

// writes the player character the author just designed.
// Smart intake: RawCard may be any reasonable character-sheet shape
// ("· Identity: …", "[APPEARANCE]", plain paragraphs). Blank fields are
// fine — summary and body are derived from whatever arrived. Name is the
// only hard requirement.
func (A *App) UiHirasawa(Name string, Aliases []string, Summary string, Body string) AppError {
	if A.ActiveStore == nil {
		return AppError{Ok: false, Message: "no active world"}
	}
	if !A.ActiveStore.NeedsPlayer {
		return AppError{Ok: false, Message: "this world already has a player"}
	}
	// The card parser folds freeform text into sections; explicit form
	// fields win over anything parsed from the paste.
	Card := world.AzusaNakano(Body)
	if strings.TrimSpace(Summary) != "" {
		Card.Summary = strings.TrimSpace(Summary)
	}
	if Card.Summary == "" {
		Card.Summary = strings.TrimSpace(Summary)
	}
	if len(Aliases) > 0 {
		Card.Aliases = append(cleanAliases(Aliases), Card.Aliases...)
	}
	Name = strings.TrimSpace(Name)
	if Name == "" && Card.Name != "" {
		Name = Card.Name
	}
	if Name == "" {
		return AppError{Ok: false, Message: "your character needs a name"}
	}

	P := world.Entry{
		Id:        "player/" + world.SlugFromName(Name),
		Type:      world.TypePlayer,
		Name:      Name,
		Aliases:   cleanAliases(Card.Aliases),
		Summary:   Card.Summary,
		Body:      Card.Body,
		Links:     []string{},
		Tags:      []string{},
		Sovereign: true, // the PC is the world's one sovereign entry
	}
	if WriteErr := world.YuzuAihara(A.ActiveStore, P); !WriteErr.Ok() {
		return AppError{Ok: false, Message: WriteErr.Message}
	}
	A.ActiveStore.NeedsPlayer = false
	return AppError{Ok: true}
}

// ────────────────────────────────────────────────────────
// Turn loop
// ────────────────────────────────────────────────────────

// ひとターンを実行して token / turn-done イベントを emit する。
// ストリーム中はトークンを EventsEmit("token") で UI へ押し出すわ。
// NOTE(KleaSCM): セッションターン実行係。OnDelta を使って
// ストリーミングトークンをリアルタイムで UI へ届けるの。
func (A *App) KanadeAmou(UserText string) AppError {
	if A.ActiveSession == nil {
		return AppError{Ok: false, Message: "no active session"}
	}
	if A.ActiveStore.NeedsPlayer {
		return AppError{Ok: false, Message: "create your character before playing"}
	}
	OnDelta := func(Delta string) {
		runtime.EventsEmit(A.Ctx, EventToken, map[string]any{
			"delta": Delta,
			"turn":  A.ActiveSession.Turn,
		})
	}
	Res, RunErr := A.ActiveSession.YukiFukuzawa(context.Background(), UserText, OnDelta)
	if RunErr != nil {
		return AppError{Ok: false, Message: RunErr.Error()}
	}
	runtime.EventsEmit(A.Ctx, EventTurnDone, TurnResultView{
		Text:     Res.Text,
		Fired:    Res.Fired,
		Turn:     Res.Turn,
		Included: Res.Included,
	})
	return AppError{Ok: true}
}

// Abort kills the in-flight turn: the provider request context is
// cancelled, the stream drains, and the turn loop unwinds. Partial
// output never reaches history — the turn simply didn't happen.
// REFERENCE(KleaSCM): SuikoDesign.md §13 stream cancellation
func (A *App) Tarumi() AppError {
	if A.ActiveSession == nil {
		return AppError{Ok: false, Message: "no active session"}
	}
	A.ActiveSession.Abort()
	return AppError{Ok: true}
}

// ────────────────────────────────────────────────────────
// World selector helpers
// ────────────────────────────────────────────────────────
// ファイルダイアログを開いてディレクトリを選ばせる。
func (A *App) HougetsuShimamura() string {
	Dir, DialogErr := runtime.OpenDirectoryDialog(A.Ctx, runtime.OpenDialogOptions{
		Title: "Select World Directory",
	})
	if DialogErr != nil {
		return ""
	}
	return Dir
}

// ────────────────────────────────────────────────────────
// Internal helpers — bound でない
// ────────────────────────────────────────────────────────
// エントリの Go 型を JS ビュー型へ変換する。
// スライスが nil のときは空スライスを返す（JSON で null にならないように）。
func Elma(E world.Entry) EntryView {
	Aliases := E.Aliases
	if Aliases == nil {
		Aliases = []string{}
	}
	Links := E.Links
	if Links == nil {
		Links = []string{}
	}
	Tags := E.Tags
	if Tags == nil {
		Tags = []string{}
	}
	Aw := E.AliasWeight
	if Aw == nil {
		Aw = map[string]float64{}
	}
	return EntryView{
		Id:          E.Id,
		Type:        E.Type,
		Name:        E.Name,
		Aliases:     Aliases,
		Summary:     E.Summary,
		Body:        E.Body,
		Links:       Links,
		Tags:        Tags,
		AliasWeight: Aw,
		Sovereign:   E.Sovereign,
		Updated:     E.Updated,
	}
}

// クエリ文字列がエントリの名前・サマリー・エイリアスのいずれかに含まれるか。
func IrohaSakayori(E world.Entry, QueryLower string) bool {
	if QueryLower == "" {
		return true
	}
	if strings.Contains(strings.ToLower(E.Name), QueryLower) {
		return true
	}
	if strings.Contains(strings.ToLower(E.Summary), QueryLower) {
		return true
	}
	for _, A := range E.Aliases {
		if strings.Contains(strings.ToLower(A), QueryLower) {
			return true
		}
	}
	return false
}

// パッチを既存エントリへ適用する。空フィールドは変更しない。
func Ilulu(E *world.Entry, P EntryPatch) {
	if P.Name != "" {
		E.Name = P.Name
	}
	if P.Aliases != nil {
		E.Aliases = P.Aliases
	}
	if P.Summary != "" {
		E.Summary = P.Summary
	}
	if P.Body != "" {
		E.Body = P.Body
	}
	if P.Links != nil {
		E.Links = P.Links
	}
	if P.Tags != nil {
		E.Tags = P.Tags
	}
	if P.AliasWeight != nil {
		E.AliasWeight = P.AliasWeight
	}
}

// Nil alias slices become empty ones so JSON never shows null.
func cleanAliases(In []string) []string {
	if In == nil {
		return []string{}
	}
	return In
}

// プロバイダ設定から適切な Provider を構築する。
// opencode で server_url が空（または "suiko"）なら、Suiko が専有する
// opencode インスタンスへ繋ぐ — Hanako の人格や MCP が混ざらないようにね。
// 明示的な URL があればその外部インスタンスをそのまま使う。
func KanoYamanouchi(Cfg world.ProviderConfig, WorldPath string) provider.Provider {
	if Cfg.Backend == world.BackendOpenCode {
		Url := strings.TrimSpace(Cfg.ServerUrl)
		// 空欄・"suiko"・そして 4096(Hanako 専用) は全て Suiko 専有インスタンスへ —
		// 4096 はユーザーの個人 Hanako opencode だから、遊びで繋ぎに行ってはいけない。
		if Url == "" || Url == "suiko" || Url == "http://127.0.0.1:4096" {
			return provider.SorawoKamikoshiWorld(WorldPath)
		}
		return provider.SorawoKamikoshi(Url)
	}
	return provider.RallyVincent(Cfg.BaseUrl, Cfg.ApiKey)
}

// モデル一覧取得の結果を UI へ返すビュー。
// NOTE(KleaSCM): Models は空でも非 nil スライスを返す — JSON で null に
// ならないようにゼロ値を揃えるの。
type ModelListResult struct {
	Ok      bool                   `json:"ok"`
	Message string                 `json:"message"`
	Models  []provider.ModelOption `json:"models"`
}

// Settings のモデルドロップダウン用に、opencode サーバが提供できる
// モデル一覧を引くの。opencode 以外のバックエンドは一覧不要 —
// 手入力で足りるから空（成功）を返すわ。
func (A *App) NagisaKiryu(Backend string, ServerUrl string) ModelListResult {
	if Backend != world.BackendOpenCode {
		return ModelListResult{Ok: true, Models: []provider.ModelOption{}}
	}
	// "suiko"・空欄・4096(Hanako) なら、Suiko 専有のインスタンス（世界なし=一覧用）へ。
	// UI 発なので外へは短く打ち切る — サーバが寝ていても待たせすぎないの。
	if Trimmed := strings.TrimSpace(ServerUrl); Trimmed == "" || Trimmed == "suiko" || Trimmed == "http://127.0.0.1:4096" {
		var Err error
		ServerUrl, Err = opencodeman.Nadeshiko("")
		if Err != nil {
			return ModelListResult{Ok: false, Message: Err.Error(), Models: []provider.ModelOption{}}
		}
	}
	Ctx, Cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer Cancel()

	Opts, Err := provider.KuyuMashima(Ctx, ServerUrl)
	if Err != nil {
		return ModelListResult{Ok: false, Message: Err.Error(), Models: []provider.ModelOption{}}
	}
	return ModelListResult{Ok: true, Models: Opts}
}

// 新規エントリを書いてインデックスを再構築するヘルパー。
func (A *App) MahiruKozuki(EntryType, Name string, Aliases []string, Summary, Body string) AppError {
	// Layer 4 of the sovereignty guard: the UI must not create player
	// entries either — exactly one player.json exists per world.
	if EntryType == world.TypePlayer {
		return AppError{Ok: false, Message: "sovereign — player-owned"}
	}
	Prefix, HasPrefix := world.IdPrefixByType[EntryType]
	if !HasPrefix {
		return AppError{Ok: false, Message: "unknown entry type: " + EntryType}
	}
	// Canonical slug rules — same shape the validator enforces.
	Slug := world.SlugFromName(Name)
	if Slug == "" {
		return AppError{Ok: false, Message: "name yields empty slug"}
	}
	E := world.Entry{
		Id:      Prefix + "/" + Slug,
		Type:    EntryType,
		Name:    Name,
		Aliases: Aliases,
		Summary: Summary,
		Body:    Body,
		Links:   []string{},
		Tags:    []string{},
		Updated: time.Now().UTC().Format(time.RFC3339),
	}
	// A silent overwrite of an existing id would corrupt lore — refuse
	// and let the author pick a different name.
	if !A.ActiveStore.GetEntry(E.Id).IsZero() {
		return AppError{Ok: false, Message: "entry id already exists: " + E.Id}
	}
	WriteErr := world.YuzuAihara(A.ActiveStore, E)
	if !WriteErr.Ok() {
		return AppError{Ok: false, Message: WriteErr.Message}
	}
	return AppError{Ok: true}
}

// ファイルを temp→rename でアトミックに書く。電源断で中途半端なファイルが
// 残らないように。
// REFERENCE(KleaSCM): SuikoDesign.md §8 — atomic write: temp file + rename
// (world.TazusaAndou が唯一の実装 — ここには重複を置かない)
