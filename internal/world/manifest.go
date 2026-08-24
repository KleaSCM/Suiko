/**
 * World Manifest (world.json) — ワールドごとの識別と調整の設定ファイルね。
 *
 * player.json と並んで必須の唯一のファイル。ワールドの名前と、エンジンの
 * 調整値を持つの：注入予算の既定、プロンプトコンパイラが Tier 0 に折り込む
 * ナレーター文体のルール、内蔵チャットループ用のプロバイダ設定。
 * Name 以外は全部任意で、ゼロ値の予算フィールドはロード時にエンジン既定へ
 * 置き換えられるわ。
 *
 * JSON タグは snake_case — このコードベースで許された唯一のワイヤ形式例外ね。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

// マニフェストが予算フィールドを空けていたときに適用されるエンジン既定。
// REFERENCE(KleaSCM): SuikoDesign.md §5/§7 — 注入予算と重複排除窓を束ねる
const (
	DefaultInjectTokens    = 3000
	DefaultTopKEntries     = 8
	DefaultRecencyTurns    = 20
	DefaultDedupWindowTurn = 10
)

// プロバイダバックエンドの選択肢。
// REFERENCE(KleaSCM): SuikoDesign.md §12 — opencode サーバか素の
// OpenAI互換クライアントか。下流のセッションループは同一の振る舞いね。
const (
	BackendOpenAI   = "openai"
	BackendOpenCode = "opencode"
)

type Budget struct {
	InjectMaxTokens   int `json:"inject_max_tokens"`
	TopKEntries       int `json:"top_k_entries"`
	RecencyBoostTurns int `json:"recency_boost_turns"`
	DedupWindowTurns  int `json:"dedup_window_turns"`
}

type ProviderConfig struct {
	Backend   string `json:"backend"`
	ServerUrl string `json:"server_url"`
	BaseUrl   string `json:"base_url"`
	ModelId   string `json:"model_id"`
	ApiKey    string `json:"api_key,omitempty"`
}

type WorldManifest struct {
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	StartingScene    string         `json:"starting_scene"`
	NarratorRules    []string       `json:"narrator_rules"`
	AutoAcceptWrites bool           `json:"auto_accept_writes"`
	Budget           Budget         `json:"budget"`
	Provider         ProviderConfig `json:"provider"`
}

// ゼロ値の調整フィールドはエンジン既定で埋める。正の値にクランプするのは、
// 下流の算術がゼロ除算したり空っぽの文脈を組んだりしないようにするためね。
// バックエンドが未知の文字列だったら素の OpenAI互換へ落ちる — 起動は止めず、
// validate が警告を挙げる道を残すの。
func (M WorldManifest) WithDefaults() WorldManifest {
	if M.Budget.InjectMaxTokens <= 0 {
		M.Budget.InjectMaxTokens = DefaultInjectTokens
	}
	if M.Budget.TopKEntries <= 0 {
		M.Budget.TopKEntries = DefaultTopKEntries
	}
	if M.Budget.RecencyBoostTurns <= 0 {
		M.Budget.RecencyBoostTurns = DefaultRecencyTurns
	}
	if M.Budget.DedupWindowTurns <= 0 {
		M.Budget.DedupWindowTurns = DefaultDedupWindowTurn
	}
	if M.Provider.Backend == "" {
		M.Provider.Backend = BackendOpenAI
	}
	return M
}

func (P ProviderConfig) KnownBackend() bool {
	return P.Backend == BackendOpenAI || P.Backend == BackendOpenCode
}
