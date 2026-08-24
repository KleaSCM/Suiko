/**
 * Suiko Frontend Types — GoのAPI境界をTypeScriptミラーで写したものね。
 *
 * Wails v2 がバインディング生成する前の型スタブ。生成後は wailsjs/go/main/App に
 * 同等の型が出るけど、このファイルはドメイン型としての役割を保持するわ。
 *
 * DESIGN PHILOSOPHY:
 * Go の Entry / WorldManifest / Event を PascalCase フィールドそのままに型付けする。
 * JSON タグ（snake_case）はワイヤ上にあるけど、JS 側には PascalCase でも snake_case
 * でも Wails が変換してくれるから、ここは Go と対称に保つのが一番明快なの。
 *
 * DATA LAYOUT:
 * ┌──────────────────┬────────────────────────────────────────────────┐
 * │ Type             │ Mirrors                                        │
 * ├──────────────────┼────────────────────────────────────────────────┤
 * │ Entry            │ world.Entry + EntryView (app.go)               │
 * │ WorldManifest    │ world.WorldManifest                            │
 * │ Budget           │ world.Budget                                   │
 * │ ProviderConfig   │ world.ProviderConfig                           │
 * │ WorldEvent       │ world.Event (app.go EventView)                 │
 * │ SceneState       │ scene.State (app.go SceneView)                 │
 * │ TurnResult       │ session.TurnResult (app.go TurnResultView)     │
 * │ SearchHit        │ app.go SearchHit                               │
 * │ WorldInfo        │ app.go WorldInfo                               │
 * │ AppError         │ app.go AppError                                │
 * └──────────────────┴────────────────────────────────────────────────┘
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */

// エントリタイプの列挙。Go の world パッケージの定数と対応する。
export type EntryType =
	| "player"
	| "character"
	| "location"
	| "item"
	| "faction"
	| "lore";

// イベント種別。events.go の定数と対応する。
export type EventKind =
	| "scene"
	| "move"
	| "thread"
	| "resolution"
	| "offscreen"
	| "note";

// ロアエントリの完全ビュー。
export interface Entry {
	readonly Id: string;
	readonly Type: EntryType;
	readonly Name: string;
	readonly Aliases: string[];
	readonly Summary: string;
	readonly Body: string;
	readonly Links: string[];
	readonly Tags: string[];
	readonly AliasWeight: Record<string, number>;
	readonly Sovereign: boolean;
	readonly Updated: string;
}

// AddEntry / UpdateEntry で送るパッチ。空文字フィールドは変更しない。
export interface EntryPatch {
	Name: string;
	Aliases: string[];
	Summary: string;
	Body: string;
	Links: string[];
	Tags: string[];
	AliasWeight: Record<string, number>;
}

// 検索ヒット — body なしの軽量ビュー。
export interface SearchHit {
	readonly Id: string;
	readonly Type: EntryType;
	readonly Name: string;
	readonly Summary: string;
}

// world.json の予算設定。
export interface Budget {
	InjectMaxTokens: number;
	TopKEntries: number;
	RecencyBoostTurns: number;
	DedupWindowTurns: number;
}

// world.json のプロバイダ設定。
export interface ProviderConfig {
	Backend: "openai" | "opencode" | string;
	ServerUrl: string;
	BaseUrl: string;
	ModelId: string;
	ApiKey: string;
}

// world.json のマニフェスト全体。
export interface WorldManifest {
	Name: string;
	Description: string;
	StartingScene: string;
	NarratorRules: string[];
	AutoAcceptWrites: boolean;
	Budget: Budget;
	Provider: ProviderConfig;
}

// worlds/ 走査の一項目。
export interface WorldInfo {
	readonly Name: string;
	readonly Path: string;
	readonly Description: string;
}

// セッションイベントログの一行。
export interface WorldEvent {
	readonly Timestamp: string;
	readonly Turn: number;
	readonly Kind: EventKind;
	readonly Text: string;
	readonly Participants: string[];
	readonly Location: string;
}

// シーン状態（events/ から導出される）。
export interface SceneState {
	readonly Now: string;
	readonly Location: string;
	readonly Present: string[];
	readonly OpenThreads: string[];
}

// ひとターンの成果。turn-done イベントで届く。
export interface TurnResult {
	readonly Text: string;
	readonly Fired: string[];
	readonly Turn: number;
	readonly Included: boolean;
}

// Go バインディングの標準エラー型。
export interface AppError {
	readonly Ok: boolean;
	readonly Message: string;
}

// モデルドロップダウンの1行。value は "providerID/modelID" の形ね。
export interface ModelOption {
	readonly Value: string;
	readonly Label: string;
}

// opencode サーバから引いたモデル一覧の成果。
export interface ModelListResult {
	readonly Ok: boolean;
	readonly Message: string;
	readonly Models: ModelOption[];
}

// チャット履歴の1メッセージ（フロントエンド専用）。
export type MessageRole = "user" | "narrator";

export interface ChatMessage {
	readonly Id: string;           // ブラウザ側の UUID — DOM キー用ね
	readonly Role: MessageRole;
	readonly Content: string;
	readonly Turn: number;
	readonly FiredIds: string[];   // ロアカードを表示するエントリidの集合
	readonly Streaming: boolean;   // ストリーム中は true
}

// pending キューのアイテム型（UI 専用）。
export interface PendingWrite {
	readonly Entry: Entry;
	readonly AddedAt: number;      // Date.now() — 表示順の安定のため
}

// ナビゲーション先の識別子。
export type ViewId = "play" | "world" | "sessions" | "settings";
