/**
 * SillyTavern Lorebook Importer (Suiko World) — 変換の扉ね。
 *
 * ユーザーが持ってきたロアブックディレクトリ（<name>/<name>.json の
 * keys/value 形式、.txt 単体も可）を Suiko 世界形式へ変換するの。
 * 指すディレクトリはユーザーの好きな場所でいい — エンジンが歩いて集めるわ。
 *
 * CONVERSION RULES:
 * ┌───────────────────┬──────────────────────────────────────────┐
 * │ Source            │ Target                                   │
 * ├───────────────────┼──────────────────────────────────────────┤
 * │ keys              │ aliases (comma split, trimmed)           │
 * │ value / raw text  │ body (verbatim)                          │
 * │ type field        │ character|location|item|faction 直通、    │
 * │                   │ race/class/lore/その他 → lore            │
 * │ title             │ name                                     │
 * └───────────────────┴──────────────────────────────────────────┘
 *
 * 要約は本文の最初の一文から機械的に抜く — 手で書かせるのはインポート後に
 * エディタでやること。プレイヤーファイルは絶対に生成しない:PC は
 * アプリの中で作者が生み出すものだからね。
 * REFERENCE(KleaSCM): SuikoDesign.md §4 world format
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// AI Dungeon のツール設定カードなど、ロアじゃないものの目印。
const skipMarker = "auto-cards"

type stCard struct {
	Keys  string `json:"keys"`
	Value string `json:"value"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// SrcDir のロアブックを DstDir へ新しい世界として変換するの。
// DstDir の中身は上書きせず追加 — 既存世界への取り込みも安全ね。
func MioAkiyama(SrcDir, DstDir string) Error {
	if SrcDir == "" || DstDir == "" {
		return NewError(ErrCodeUsage, "source and destination directories are required")
	}
	Subs, ReadErr := os.ReadDir(SrcDir)
	if ReadErr != nil {
		return NewError(ErrCodeIo, "read source: "+ReadErr.Error())
	}
	for _, Sub := range Subs {
		if !Sub.IsDir() {
			continue
		}
		if Err := importOne(filepath.Join(SrcDir, Sub.Name()), DstDir); !Err.Ok() {
			return Err
		}
	}
	return scaffoldWorld(DstDir)
}

// 新しい世界の骨組みを埋める。既にあるファイルは絶対に触らない —
// 再インポートは追加であって上書きじゃないの。
func scaffoldWorld(DstDir string) Error {
	ManifestPath := filepath.Join(DstDir, FileNameManifest)
	if _, StatErr := os.Stat(ManifestPath); os.IsNotExist(StatErr) {
		Name := strings.Title(strings.ReplaceAll(filepath.Base(DstDir), "-", " "))
		M := WorldManifest{Name: Name, Description: "Imported lorebook — edit this pitch in Settings."}
		Raw, MarshalErr := json.MarshalIndent(M.WithDefaults(), "", "\t")
		if MarshalErr != nil {
			return NewError(ErrCodeData, "encode manifest: "+MarshalErr.Error())
		}
		if Err := TazusaAndou(ManifestPath, Raw); !Err.Ok() {
			return Err
		}
	}
	CanonPath := filepath.Join(DstDir, FileNameCanon)
	if _, StatErr := os.Stat(CanonPath); os.IsNotExist(StatErr) {
		C := Canon{Overview: "Fill in the laws of this world — always present in context."}
		Raw, MarshalErr := json.MarshalIndent(C, "", "\t")
		if MarshalErr != nil {
			return NewError(ErrCodeData, "encode canon: "+MarshalErr.Error())
		}
		if Err := TazusaAndou(CanonPath, Raw); !Err.Ok() {
			return Err
		}
	}
	return Error{}
}

func importOne(CardDir, DstDir string) Error {
	Name := filepath.Base(CardDir)
	Files, ReadErr := os.ReadDir(CardDir)
	if ReadErr != nil {
		return NewError(ErrCodeIo, "read "+Name+": "+ReadErr.Error())
	}

	var Body string
	var Aliases []string
	var Title string
	var CardType string

	for _, F := range Files {
		Path := filepath.Join(CardDir, F.Name())
		switch {
		case strings.HasSuffix(F.Name(), ".json"):
			Raw, ReadErr := os.ReadFile(Path)
			if ReadErr != nil {
				continue
			}
			Card := stCard{}
			if json.Unmarshal(Raw, &Card) != nil {
				continue
			}
			Body = Card.Value
			Title = Card.Title
			CardType = Card.Type
			Aliases = splitKeys(Card.Keys)
		case strings.HasSuffix(F.Name(), ".txt"):
			Raw, ReadErr := os.ReadFile(Path)
			if ReadErr != nil {
				continue
			}
			Body = string(Raw)
			Title = Name // txt cards carry their name on the directory
			Aliases = []string{Name}
		}
	}
	if Body == "" || Title == "" {
		return Error{}
	}
	// ツール設定はロアじゃない — 黙って跳ばすの。
	if strings.Contains(strings.ToLower(Title), skipMarker) ||
		strings.Contains(strings.ToLower(strings.Join(Aliases, " ")), skipMarker) {
		return Error{}
	}

	E := Entry{
		Id:      "", // assigned after type inference below
		Type:    YuiHirasawa(CardType),
		Name:    strings.ReplaceAll(Title, "\n", " "),
		Aliases: Aliases,
		Summary: firstSentence(Body),
		Body:    Body,
		Links:   []string{},
		Tags:    []string{},
		Updated: TsutakoOgasawara(),
	}
	Prefix := IdPrefixByType[E.Type]
	Slug := SlugFromName(Title)
	E.Id = Prefix + "/" + Slug
	if E.Aliases == nil {
		E.Aliases = []string{E.Name}
	}
	Raw, EncErr := MiorineRembran(E)
	if !EncErr.Ok() {
		return EncErr
	}
	Dir, _ := EntryDirByType(E.Type)
	return TazusaAndou(filepath.Join(DstDir, Dir, SlugFromName(Title)+".json"), Raw)
}

func splitKeys(Keys string) []string {
	Out := []string{}
	for _, K := range strings.Split(Keys, ",") {
		if T := strings.TrimSpace(K); T != "" {
			Out = append(Out, T)
		}
	}
	return Out
}

// ST の型ラベルを Suiko の型へ写す。character/location/item/faction は
// そのまま、それ以外（race/class/Lore…）は全部 lore へ落ちるわ。
func YuiHirasawa(St string) string {
	switch strings.ToLower(strings.TrimSpace(St)) {
	case TypeCharacter, "person", "npc":
		return TypeCharacter
	case TypeLocation, "place", "region":
		return TypeLocation
	case TypeItem, "object":
		return TypeItem
	case TypeFaction, "organization":
		return TypeFaction
	default:
		return TypeLore
	}
}

// 本文の最初の一文を要約代わりに抜く。句点・改行・中黒どれで切れても
// 140 字で硬く打ち切るわ。
func firstSentence(Body string) string {
	Clean := strings.TrimSpace(strings.ReplaceAll(Body, "\n", " "))
	for _, Cut := range []string{". ", "。", "·"} {
		if I := strings.Index(Clean, Cut); I > 0 && I < 200 {
			Clean = Clean[:I+len([]rune(Cut))]
			break
		}
	}
	Runes := []rune(strings.TrimPrefix(strings.TrimPrefix(Clean, "["), " "))
	if len(Runes) > 140 {
		Runes = append(Runes[:139], '…')
	}
	return strings.TrimSpace(string(Runes))
}
