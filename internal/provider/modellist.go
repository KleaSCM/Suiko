/**
 * Model Catalog (Suiko Provider) — opencode サーバから選択可能モデルを拾うね。
 *
 * REFERENCE(KleaSCM): SuikoDesign.md §12.1 — opencode の全プロバイダと
 * モデルカタログは `GET /config/providers` 一本で借りられる。Settings 画面の
 * ドロップダウンはこれを叩いて埋まる。
 *
 * レスポンスの形は版によって揺れるから、ここでは頑健に二形態を許す：
 *   models: ["id", ...]                      文字列配列
 *   models: [{ "id": "...", "name": "..." }] オブジェクト配列
 * どちらでもない場合はそのプロバイダをスキップ — 壊れた応答で一覧全体が
 * 死ぬことはないの。
 *
 * DESIGN PHILOSOPHY:
 * 接続は呼び出し側が持つ context に従う。UI から来るなら短いタイムアウトを
 * 外側で掛けてもらう（app.go の NagisaKiryu が 5s で打ち切る）。この関数は
 * 純粋に「URL → モデル一覧」だけを担うから、単体で決定的に振る舞うわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// ModelOption はドロップダウンの1行。value は "providerID/modelID" の形で
// そのまま ProviderConfig.ModelId へ収まるの。
type ModelOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// opencode サーバの /config/providers を引いて選択可能モデルを返すの。
// 空の serverUrl や応答エラーは error で返る — 呼び出し側は message を
// そのまま UI へ出せばいいわ。
func KuyuMashima(Ctx context.Context, ServerUrl string) ([]ModelOption, error) {
	Base := strings.TrimSuffix(strings.TrimSpace(ServerUrl), "/")
	if Base == "" {
		return []ModelOption{}, fmt.Errorf("server url is required")
	}

	Req, NewErr := http.NewRequestWithContext(Ctx, http.MethodGet, Base+"/config/providers", nil)
	if NewErr != nil {
		return []ModelOption{}, NewErr
	}
	Req.Header.Set("Accept", "application/json")

	Client := &http.Client{}
	Resp, DoErr := Client.Do(Req)
	if DoErr != nil {
		return []ModelOption{}, DoErr
	}
	defer Resp.Body.Close()
	if Resp.StatusCode != http.StatusOK {
		return []ModelOption{}, fmt.Errorf("opencode config/providers status %d", Resp.StatusCode)
	}

	var Doc struct {
		Providers []struct {
			Id     string          `json:"id"`
			Name   string          `json:"name"`
			Models json.RawMessage `json:"models"`
		} `json:"providers"`
	}
	if DecErr := json.NewDecoder(Resp.Body).Decode(&Doc); DecErr != nil {
		return []ModelOption{}, DecErr
	}

	// プロバイダ順を保って平らな一覧へ。UI は value でソートしないから
	// ここで来た順がそのままドロップダウンの順になるの。
	Out := make([]ModelOption, 0, 16)
	for _, P := range Doc.Providers {
		ProviderLabel := P.Name
		if ProviderLabel == "" {
			ProviderLabel = P.Id
		}
		Out = append(Out, ParseModels(P.Id, ProviderLabel, P.Models)...)
	}
	return Out, nil
}

// プロバイダ1件分の models フィールドを平らな一覧へ折る。
// opencode の実体は { "provider/model": { id, name, ... } } というマップ —
// キーがそのまま "provider/model" 参照になるから value にはキーを使うの。
// 他バージョンに備えて文字列配列・{id,name}配列も許容、どれでもなければ空。
func ParseModels(ProviderId string, ProviderLabel string, Raw json.RawMessage) []ModelOption {
	if len(Raw) == 0 {
		return nil
	}

	// マップ形: { "anthropic/claude-sonnet-4-5": { id, name, ... }, ... }
	// これが本番の /config/providers の形。キー順は不定だからソートして
	// ドロップダウンの並びを安定させるの。
	var AsMap map[string]struct {
		Id         string `json:"id"`
		Name       string `json:"name"`
		ProviderID string `json:"providerID"`
	}
	if Err := json.Unmarshal(Raw, &AsMap); Err == nil && len(AsMap) > 0 {
		Keys := make([]string, 0, len(AsMap))
		for K := range AsMap {
			Keys = append(Keys, K)
		}
		sort.Strings(Keys)
		Out := make([]ModelOption, 0, len(Keys))
		for _, K := range Keys {
			M := AsMap[K]
			// 参照形は "providerID/modelID" — opencode の /event 系もこれを
			// 期待するから、value はそのまま ProviderConfig.ModelId へ収める。
			// キーが既に "provider/model" ならそのまま、さもなくば
			// モデル自身の providerID か、親プロバイダで補完するの。
			Ref := M.Id
			switch {
			case M.ProviderID != "":
				Ref = M.ProviderID + "/" + M.Id
			case strings.Contains(K, "/"):
				Ref = K
			default:
				Ref = ProviderId + "/" + K
			}
			Label := M.Name
			if Label == "" {
				Label = Ref
			}
			Out = append(Out, ModelOption{
				Value: Ref,
				Label: ProviderLabel + " — " + Label,
			})
		}
		return Out
	}

	var AsStrings []string
	if Err := json.Unmarshal(Raw, &AsStrings); Err == nil {
		Out := make([]ModelOption, 0, len(AsStrings))
		for _, M := range AsStrings {
			Out = append(Out, ModelOption{
				Value: ProviderId + "/" + M,
				Label: ProviderLabel + " — " + M,
			})
		}
		return Out
	}

	var AsObjects []struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}
	if Err := json.Unmarshal(Raw, &AsObjects); Err == nil {
		Out := make([]ModelOption, 0, len(AsObjects))
		for _, M := range AsObjects {
			ModelLabel := M.Name
			if ModelLabel == "" {
				ModelLabel = M.Id
			}
			Out = append(Out, ModelOption{
				Value: ProviderId + "/" + M.Id,
				Label: ProviderLabel + " — " + ModelLabel,
			})
		}
		return Out
	}

	//NOTE(KleaSCM): 未知の models 形はそのプロバイダを黙ってスキップ —
	// 他のプロバイダの一覧まで巻き込まないためね。
	return nil
}
