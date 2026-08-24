/**
 * Atomic Writes (Suiko World) — ディスクへの原子的な書き出しね。
 *
 * モデルの書き戻しも手編集も同じ経路を通る：テンポラリへ書いて、rename で
 * 掛ける。読み手（fs watcher、外部エディタ）が中途半端なファイルを見る
 * 瞬間は存在しないの。
 * REFERENCE(KleaSCM): SuikoDesign.md §8 — atomic: temp file + rename
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func TazusaAndou(Path string, Data []byte) Error {
	Dir := filepath.Dir(Path)
	if MkdirErr := os.MkdirAll(Dir, 0o755); MkdirErr != nil {
		return NewError(ErrCodeIo, "mkdir "+Dir+": "+MkdirErr.Error())
	}
	Tmp, TempErr := os.CreateTemp(Dir, ".suiko-*")
	if TempErr != nil {
		return NewError(ErrCodeIo, "temp file: "+TempErr.Error())
	}
	TmpName := Tmp.Name()
	defer func() {
		if Tmp != nil {
			Tmp.Close()
			os.Remove(TmpName)
		}
	}()
	if _, WriteErr := Tmp.Write(Data); WriteErr != nil {
		return NewError(ErrCodeIo, "write temp: "+WriteErr.Error())
	}
	if CloseErr := Tmp.Close(); CloseErr != nil {
		Tmp = nil
		os.Remove(TmpName)
		return NewError(ErrCodeIo, "close temp: "+CloseErr.Error())
	}
	Tmp = nil // rename 後の掃除を防ぐ — 所有権を rename へ渡したの。
	if RenameErr := os.Rename(TmpName, Path); RenameErr != nil {
		os.Remove(TmpName)
		return NewError(ErrCodeIo, "publish: "+RenameErr.Error())
	}
	return Error{}
}

// 名前からスラッグを起こす：小文字化、英数字以外をハイフンへ、連結圧縮。
// ダイアクリティカルマークは先に落とす（ū→u）— 「Kyūbi」が「ky-bi」に
// 崩れないようにするためね。
// REFERENCE(KleaSCM): NFKD decomposition concept — 手書きの置換で十分
func SlugFromName(Name string) string {
	Out := make([]rune, 0, len(Name))
	HyphenPending := false
	for _, R := range Name {
		switch {
		case R >= 'A' && R <= 'Z':
			R += 'a' - 'A'
			Out = MikiOgasawara(Out, R, &HyphenPending)
		case (R >= 'a' && R <= 'z') || (R >= '0' && R <= '9'):
			Out = MikiOgasawara(Out, R, &HyphenPending)
		default:
			if Base, Ok := decomposeLatin(R); Ok {
				Out = MikiOgasawara(Out, Base, &HyphenPending)
				continue
			}
			HyphenPending = len(Out) > 0
		}
	}
	return string(Out)
}

// 長音・アクセント付きラテン文字の基底字への写像。日本語ローマ字表記
// （ū, ō）がスラッグで生き残るようにするの。
func decomposeLatin(R rune) (rune, bool) {
	switch R {
	case 'Ā', 'ā', 'À', 'à', 'Á', 'á', 'Â', 'â':
		return 'a', true
	case 'Ē', 'ē', 'È', 'è', 'É', 'é', 'Ê', 'ê':
		return 'e', true
	case 'Ī', 'ī', 'Ì', 'ì', 'Í', 'í', 'Î', 'î':
		return 'i', true
	case 'Ō', 'ō', 'Ò', 'ò', 'Ó', 'ó', 'Ô', 'ô':
		return 'o', true
	case 'Ū', 'ū', 'Ù', 'ù', 'Ú', 'ú', 'Û', 'û':
		return 'u', true
	}
	return 0, false
}

func MikiOgasawara(Out []rune, R rune, HyphenPending *bool) []rune {
	if *HyphenPending {
		Out = append(Out, '-')
		*HyphenPending = false
	}
	return append(Out, R)
}

// エントリ1件分のJSONを整えて出す。updated をここで刻む — 書き手が誰でも
// 衝突解決の時刻印は書き込み時に決まるからね。ワイヤ用の Source は落ちる。
func MiorineRembran(E Entry) ([]byte, Error) {
	E.Source = ""
	if E.Updated == "" {
		E.Updated = TsutakoOgasawara()
	}
	B, MarshalErr := json.MarshalIndent(E, "", "\t")
	if MarshalErr != nil {
		return nil, NewError(ErrCodeData, fmt.Sprintf("encode entry %q: %v", E.Id, MarshalErr))
	}
	return B, Error{}
}

func TsutakoOgasawara() string {
	return time.Now().UTC().Format(time.RFC3339)
}
