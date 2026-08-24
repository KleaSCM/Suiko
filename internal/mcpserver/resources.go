/**
 * Suiko Resources — まとめて文脈を読むための suiko:// ドキュメントね。
 *
 * ツールが的を絞った質問に答えるのに対して、リソースはドキュメント丸ごとを
 * 渡すの：方向づけのためのワールドツリー、生エントリ、カノン、今日のイベント
 * ログ。URI は SuikoDesign.md §6 どおりだから、クライアントはそのまま
 * ブックマークできるわ。
 *
 * DESIGN PHILOSOPHY:
 * 同じストア、別の形状 — リソースはスナップショットの投影であって、新鮮な
 * ディスク読み取りではないの。例外は events/today.json の遅延読みだけね。
 * エントリ本文もここに乗る（コンパクトなツール行と違って）。明示的に
 * リソースを求めたクライアントは完全なテキストを欲しがってるはずだからね。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package mcpserver

import (
	"encoding/json"
	"os"
	"strings"

	"suiko/internal/world"
)

// リソース URI のルート群。クライアントがそのままブックマークするの。
const (
	UriTree   = "suiko://world/tree"
	UriCanon  = "suiko://canon"
	UriEvents = "suiko://events/today"
	UriEntryP = "suiko://entry/"
)

// 何も起きる前に出す空ログのドキュメント。
const EmptyEventsDoc = `{"date":"","events":[]}`

// 静的な記述子だけ。エントリごとのURIはこの一覧を溢れさせる代わりに、ツリー
// リソース経由で発見できるの。
func (S *Server) ListResources() []ResourceDescriptor {
	return []ResourceDescriptor{
		{
			Uri:         UriTree,
			Name:        "World tree",
			Description: "Every entry grouped by type with summaries — the map of the world.",
			MimeType:    MimeTypeJson,
		},
		{
			Uri:         UriCanon,
			Name:        "Canon document",
			Description: "Tier-0 lore present in every context: overview, laws, tone, hard facts.",
			MimeType:    MimeTypeJson,
		},
		{
			Uri:         UriEntryP + "{id}",
			Name:        "Entry",
			Description: "Raw entry JSON, e.g. suiko://entry/char/kaori.",
			MimeType:    MimeTypeJson,
		},
		{
			Uri:         UriEvents,
			Name:        "Today's events",
			Description: "Append-only scene log for today; empty until something happens.",
			MimeType:    MimeTypeJson,
		},
	}
}

// NOTE(KleaSCM): ここは全部ロード済みスナップショットの投影。ディスクに触れるのは
// 今日のイベントログだけで、プレイ中に伸びるから遅延読みなの。
func (S *Server) ReadResource(Uri string) (string, string, world.Error) {
	switch {
	case Uri == UriTree:
		return string(S.treeJson()), MimeTypeJson, world.Error{}

	case Uri == UriCanon:
		B, MErr := json.MarshalIndent(S.Store.Canon, "", "  ")
		if MErr != nil {
			return "", "", world.NewError(world.ErrCodeData, "encode canon: "+MErr.Error())
		}
		return string(B), MimeTypeJson, world.Error{}

	case strings.HasPrefix(Uri, UriEntryP):
		Id := strings.TrimPrefix(Uri, UriEntryP)
		E := S.Store.GetEntry(Id)
		if E.IsZero() {
			return "", "", world.NewError(world.ErrCodeUsage, MsgUnknownEntry+Id)
		}
		B, MErr := json.MarshalIndent(E, "", "  ")
		if MErr != nil {
			return "", "", world.NewError(world.ErrCodeData, "encode entry: "+MErr.Error())
		}
		return string(B), MimeTypeJson, world.Error{}

	case Uri == UriEvents:
		Text, Err := S.eventsToday()
		return Text, MimeTypeJson, Err

	default:
		return "", "", world.NewError(world.ErrCodeUsage, "unknown resource: "+Uri)
	}
}

// 型ごとにグルーピングした投影 — 一目で世界の見取りが分かるの。
func (S *Server) treeJson() []byte {
	TypeOrder := []string{
		world.TypePlayer, world.TypeCharacter, world.TypeLocation,
		world.TypeItem, world.TypeFaction, world.TypeLore,
	}
	Grouped := map[string][]summaryRow{}
	for _, E := range S.Store.Entries() {
		Grouped[E.Type] = append(Grouped[E.Type], rowOf(E))
	}
	Tree := map[string]any{
		"world":  S.Store.Manifest.Name,
		"count":  S.Store.Count(),
		"groups": map[string]any{},
	}
	Groups := Tree["groups"].(map[string]any)
	for _, T := range TypeOrder {
		if Rows := Grouped[T]; len(Rows) > 0 {
			Groups[T] = Rows
		}
	}
	B, _ := json.MarshalIndent(Tree, "", "  ")
	return B
}

// イベントツールと同じ遅延読み。こちらのビューは素のバイトをそのまま返すわ。
func (S *Server) eventsToday() (string, world.Error) {
	Data, ReadErr := os.ReadFile(eventsTodayPath(S.Store.Root))
	if os.IsNotExist(ReadErr) {
		return EmptyEventsDoc, world.Error{}
	}
	if ReadErr != nil {
		return "", world.NewError(world.ErrCodeIo, "read events: "+ReadErr.Error())
	}
	return string(Data), world.Error{}
}
