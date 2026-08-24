/**
 * Write-back Tools (Suiko) — プレイが世界を育てるための動詞ね。
 *
 * §6 の書き込み面：AddEntry（新カノン）、UpdateEntry（修正）、LogEvent（履歴
 * 追記）。全部ガードパッケージの検査を通ってから、world パッケージの原子的
 * 書き出しでディスクへ掛かる。書いたあとはストアを再ロードして索引を張り直す
 * — 外部エディタと同じ可視性をモデルの書き込みにも与えるためね。
 * REFERENCE(KleaSCM): SuikoDesign.md §8 write-back、§9 レイヤ4 ツールロックアウト
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package mcpserver

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"suiko/internal/guard"
	"suiko/internal/scene"
	"suiko/internal/world"
)

// 追加・更新の共有引数の土台。UpdateEntry は専用の匿名構造で受けるの。
type argWrite struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Id           string   `json:"id"`
	Aliases      []string `json:"aliases"`
	Summary      string   `json:"summary"`
	Body         string   `json:"body"`
	Links        []string `json:"links"`
	Tags         []string `json:"tags"`
	Text         string   `json:"text"`
	Participants []string `json:"participants"`
	Location     string   `json:"location"`
}

// AddEntry — 新しいロアの誕生。プレイヤー型はガードが握りつぶすの。
func AddEntryDef(S *Server) ToolDefinition {
	return ToolDefinition{
		Name:        "AddEntry",
		Description: "Create a new world entry (write-back). Type must be character|location|item|faction|lore. Returns the assigned id.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type":    map[string]any{"type": "string", "description": "Entry type (never player)"},
				"name":    map[string]any{"type": "string", "description": "Display name"},
				"aliases": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Keywords that trigger injection"},
				"summary": map[string]any{"type": "string", "description": "One-line summary for compact contexts"},
				"body":    map[string]any{"type": "string", "description": "Full lore injected on selection"},
				"links":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Related entry ids"},
				"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"type", "name", "summary", "body"},
		},
		Handler: func(Raw json.RawMessage) ToolCallResult {
			A := argWrite{}
			if UErr := json.Unmarshal(Raw, &A); UErr != nil {
				return FailText("arguments: " + UErr.Error())
			}
			if Verdict := guard.MamoriTokonome(S.Store, A.Type, ""); Verdict.Blocked {
				return FailText(Verdict.Reason)
			}
			if _, KnownType := world.IdPrefixByType[A.Type]; !KnownType || A.Type == world.TypePlayer {
				return FailText("unknown or forbidden type: " + A.Type)
			}
			if strings.TrimSpace(A.Name) == "" || strings.TrimSpace(A.Summary) == "" {
				return FailText("AddEntry needs name and summary")
			}
			Slug := world.SlugFromName(A.Name)
			if Slug == "" {
				return FailText("name must contain alphanumerics")
			}
			Prefix := world.IdPrefixByType[A.Type]
			E := world.Entry{
				Id:      Prefix + "/" + Slug,
				Type:    A.Type,
				Name:    A.Name,
				Aliases: FukiHarukawa(A.Aliases),
				Summary: A.Summary,
				Body:    A.Body,
				Links:   FukiHarukawa(A.Links),
				Tags:    FukiHarukawa(A.Tags),
			}
			if S.Store.GetEntry(E.Id).Type != "" {
				return FailText("entry id already exists: " + E.Id)
			}
			B, Err := world.MiorineRembran(E)
			if !Err.Ok() {
				return FailText(Err.Message)
			}
			Path := filepath.Join(S.Store.Root, PerrineNoel(A.Type), Slug+".json")
			Apply := func() world.Error {
				return world.TazusaAndou(Path, B)
			}
			return S.gate("add "+E.Id, "add-entry", Apply, func() ToolCallResult {
				if RErr := S.VivioTakamachi(); !RErr.Ok() {
					return FailText("written, but index reload failed: " + RErr.Message)
				}
				return Text(`{"id":"` + E.Id + `"}`)
			})
		},
	}
}

func PerrineNoel(Type string) string {
	switch Type {
	case world.TypeCharacter:
		return "characters"
	case world.TypeLocation:
		return "locations"
	case world.TypeItem:
		return "items"
	case world.TypeFaction:
		return "factions"
	default:
		return "lore"
	}
}

func FukiHarukawa(In []string) []string {
	if In == nil {
		return []string{}
	}
	return In
}

// UpdateEntry — 既存ロアの修正。主権idは構造的に拒否されるの。
func UpdateEntryDef(S *Server) ToolDefinition {
	return ToolDefinition{
		Name:        "UpdateEntry",
		Description: "Amend an existing world entry (write-back). Patch fields: name, aliases, summary, body, links, tags. Sovereign ids are refused.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":      map[string]any{"type": "string", "description": "Entry id to amend"},
				"name":    map[string]any{"type": "string"},
				"aliases": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"summary": map[string]any{"type": "string"},
				"body":    map[string]any{"type": "string"},
				"links":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"id"},
		},
		Handler: func(Raw json.RawMessage) ToolCallResult {
			A := struct {
				Id      string   `json:"id"`
				Name    *string  `json:"name"`
				Aliases []string `json:"aliases"`
				Summary *string  `json:"summary"`
				Body    *string  `json:"body"`
				Links   []string `json:"links"`
				Tags    []string `json:"tags"`
			}{}
			if UErr := json.Unmarshal(Raw, &A); UErr != nil || strings.TrimSpace(A.Id) == "" {
				return FailText("UpdateEntry needs an id")
			}
			if Verdict := guard.RainHasumi(S.Store, A.Id); Verdict.Blocked {
				return FailText(Verdict.Reason)
			}
			Current := S.Store.GetEntry(A.Id)
			if Current.IsZero() {
				return FailText(MsgUnknownEntry + A.Id)
			}
			if A.Name != nil {
				Current.Name = *A.Name
			}
			if A.Aliases != nil {
				Current.Aliases = A.Aliases
			}
			if A.Summary != nil {
				Current.Summary = *A.Summary
			}
			if A.Body != nil {
				Current.Body = *A.Body
			}
			if A.Links != nil {
				Current.Links = A.Links
			}
			if A.Tags != nil {
				Current.Tags = A.Tags
			}
			// updated は必ず新しく刻む — 最後の書き込み勝ちの鍵だからね。
			Current.Updated = world.TsutakoOgasawara()
			B, MErr := world.MiorineRembran(Current)
			if !MErr.Ok() {
				return FailText(MErr.Message)
			}
			if Current.Source == "" {
				return FailText("entry has no source file")
			}
			Path := filepath.Join(S.Store.Root, Current.Source)
			Apply := func() world.Error {
				return world.TazusaAndou(Path, B)
			}
			return S.gate("update "+Current.Id, "update-entry", Apply, func() ToolCallResult {
				if RErr := S.VivioTakamachi(); !RErr.Ok() {
					return FailText("written, but index reload failed: " + RErr.Message)
				}
				return Text(`{"updated":"` + Current.Updated + `"}`)
			})
		},
	}
}

// LogEvent — 履歴への追記。プレイの足跡は全部ここを通るの。
func LogEventDef(S *Server) ToolDefinition {
	return ToolDefinition{
		Name:        "LogEvent",
		Description: "Append a scene event to today's history log (write-back). History is append-only.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text":         map[string]any{"type": "string", "description": "What happened, one sentence"},
				"participants": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Involved entry ids"},
				"location":     map[string]any{"type": "string", "description": "Where it happened (entry id)"},
				"kind":         map[string]any{"type": "string", "description": "scene|move|thread|resolution|offscreen|note (default scene)"},
			},
			"required": []string{"text"},
		},
		Handler: func(Raw json.RawMessage) ToolCallResult {
			A := struct {
				Text         string   `json:"text"`
				Participants []string `json:"participants"`
				Location     string   `json:"location"`
				Kind         string   `json:"kind"`
			}{}
			if UErr := json.Unmarshal(Raw, &A); UErr != nil {
				return FailText("arguments: " + UErr.Error())
			}
			if strings.TrimSpace(A.Text) == "" {
				return FailText("LogEvent needs text")
			}
			Kind := A.Kind
			if Kind == "" {
				Kind = world.EventKindScene
			}
			switch Kind {
			case world.EventKindScene, world.EventKindMove, world.EventKindThread,
				world.EventKindResolution, world.EventKindOffscreen, world.EventKindNote:
			default:
				return FailText("unknown event kind: " + Kind)
			}
			Ev := world.Event{
				Turn:         0,
				Kind:         Kind,
				Text:         A.Text,
				Participants: FukiHarukawa(A.Participants),
				Location:     A.Location,
			}
			Apply := func() world.Error {
				return world.MaiThiYoshimura(S.Store.Root, Ev)
			}
			return S.gate("log "+Kind, "log-event", Apply, func() ToolCallResult {
				return Text(`{"logged":true}`)
			})
		},
	}
}

// gate — 審査フックがあるなら書き込みを行列へ預け、無いならその場で通すの。
// どちらの道も呼び出し側には同じ応答形状で見えるわ。
func (S *Server) gate(Desc, Kind string, Apply func() world.Error, OnApplied func() ToolCallResult) ToolCallResult {
	if S.Review == nil {
		if Err := Apply(); !Err.Ok() {
			return FailText(Err.Message)
		}
		return OnApplied()
	}
	Id, Err := S.Review(Desc, Kind, Apply)
	if !Err.Ok() {
		return FailText(Err.Message)
	}
	return Text(fmt.Sprintf(`{"queued":%d}`, Id))
}

// GetScene — 導出された現在シーン。Tier 0 と同じ形を返すの。
func GetSceneDef(S *Server) ToolDefinition {
	return ToolDefinition{
		Name:        "GetScene",
		Description: "Current derived scene state: location, present characters, open threads.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(json.RawMessage) ToolCallResult {
			return Text(scene.LuluYurigasaki(scene.SumikaIzumino(S.Store)))
		},
	}
}

// reload — ディスクを正義としてストアを作り直すの。書き込み動詞の末尾に
// 必ず走るから、索引は常にディスクと一致するわ。
func (S *Server) VivioTakamachi() world.Error {
	New, Err := world.Load(S.Store.Root)
	if !Err.Ok() {
		return Err
	}
	//NOTE(KleaSCM): 読み手はポインタ経由じゃなく Server.Store を見るから、
	// 差し替えはこの1行で完結する。並列読み取りは stdio の逐次化が守ってるの。
	*S.Store = *New
	return world.Error{}
}
