/**
 * SSE Line Reader (Suiko Provider) — Server-Sent Events の行読みね。
 *
 * ふたつのバックエンドが同じワイヤ形式を話す：`data: ` で始まる行が
 * ペイロード、空行がイベント区切り。OpenAI互換も opencode の /event も
 * この形だから、読み手は一つで足りるわ。
 * REFERENCE(KleaSCM): WHATWG HTML spec §9.2 server-sent events（data フィールド）
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package provider

import (
	"bufio"
	"io"
	"strings"
)

// ストリームから次の data ペイロードを1つ取り出すの。終端なら ok=false。
// コメント行と event:/id: 行は無視 — 必要なのは data だけね。
func Ellis(Scan *bufio.Scanner) (string, bool) {
	for Scan.Scan() {
		Line := strings.TrimSpace(Scan.Text())
		switch {
		case strings.HasPrefix(Line, "data:"):
			return strings.TrimSpace(strings.TrimPrefix(Line, "data:")), true
		default:
			continue
		}
	}
	return "", false
}

// [DONE] センチネル。OpenAI互換ストリームの終端印ね。
const DoneSentinel = "[DONE]"

// Scanner の背後の io を閉じるための小道具。テストでも使うわ。
type ReadCloser = io.ReadCloser
