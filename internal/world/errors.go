/**
 * Error Values — ワールド読み込みのためのエラー値渡しね。
 *
 * 起動経路（ディスクからワールドをロード）は失敗しうる：ファイルは外部入力
 * だから。失敗は安定したコードつきの Error 値で返って、呼び出し側とCLIが
 * 使い方の間違い・壊れたデータ・I/O障害を区別できるの。
 * ロード成功後の実行時ルックアップは zero-is-valid に従う — GetEntry は
 * 決して失敗せず、ZeroEntry を返すわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package world

// 安定した負のコード。ゼロは成功で、常に有効。
// NOTE(KleaSCM): 失敗は起動時に閉じ込める — 実行時の検索はこれを一切
// 生まない。代わりにゼロ値を返すのが ZII の規律ね。
const (
	ErrCodeNone   = 0
	ErrCodeUsage  = -1
	ErrCodeIo     = -2
	ErrCodeSchema = -3
	ErrCodeData   = -4
)

type Error struct {
	Code    int
	Message string
}

func NewError(Code int, Message string) Error {
	return Error{Code: Code, Message: Message}
}

func (E Error) Error() string {
	return E.Message
}

func (E Error) Ok() bool {
	return E.Code == ErrCodeNone
}
