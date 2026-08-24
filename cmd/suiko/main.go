/**
 * Suiko CLI — ワールドツーリングのためのコマンドライン動詞ね。
 *
 * M0 で validate 動詞を出荷：ワールドディレクトリをロードして意味的チェックを
 * 実行し、安定した書式と意味のある終了コードで所見を印刷するの。
 * M1 で serve（stdio 上の MCP）を読み取り専用ツール／リソース面へ接続済み。
 * Wails アプリは後続マイルストーンね。
 *
 * EXIT CODES:
 * 0 — ワールド有効（警告は許容） · 1 — 無効または読めない ·
 * 2 — 使用方法エラー
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package main

import (
	"fmt"
	"os"

	"suiko/internal/mcpserver"
	"suiko/internal/world"
)

// シェルから見える終了コード。スクリプトはこれで分岐するの。
const (
	ExitValid   = 0
	ExitInvalid = 1
	ExitUsage   = 2
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(ExitUsage)
	}
	switch os.Args[1] {
	case "validate":
		CmdValidate(os.Args[2:])
	case "serve":
		CmdServe(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "suiko: unknown verb %q\n\n", os.Args[1])
		usage()
		os.Exit(ExitUsage)
	}
}

// 警告は決して妨げない — 反復が摩擦なしで回り続けるの。一方でエラーやロード
// 失敗は非ゼロ終了だから、CI もエディタもこのコードを信頼できるわ。
func CmdValidate(Args []string) {
	if len(Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: suiko validate <world-dir>")
		os.Exit(ExitUsage)
	}
	Root := Args[0]

	Store, LoadErr := world.Load(Root)
	if !LoadErr.Ok() {
		fmt.Fprintf(os.Stderr, "load failed: %s\n", LoadErr.Message)
		os.Exit(ExitInvalid)
	}

	Issues := Store.Validate()
	Errors, Warnings := 0, 0
	for _, Is := range Issues {
		if Is.Severity == world.SeverityError {
			Errors++
			fmt.Printf("error   %s: %s\n", Is.File, Is.Message)
		} else {
			Warnings++
			fmt.Printf("warning %s: %s\n", Is.File, Is.Message)
		}
	}
	fmt.Printf("\n%s: %d entries, index size %d, %d errors, %d warnings\n",
		Root, Store.Count(), Store.Index.Size(), Errors, Warnings)
	if world.HasErrors(Issues) {
		os.Exit(ExitInvalid)
	}
	fmt.Println("ok")
}

// NOTE(KleaSCM): stdout はプロトコルフレーム専用 — 人間向けの診断は stderr へ。
// MCPクライアントがstdoutをパイプしても、おしゃべりが混ざらないようにするの。
func CmdServe(Args []string) {
	if len(Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: suiko serve <world-dir>")
		os.Exit(ExitUsage)
	}
	Store, LoadErr := world.Load(Args[0])
	if !LoadErr.Ok() {
		fmt.Fprintf(os.Stderr, "load failed: %s\n", LoadErr.Message)
		os.Exit(ExitInvalid)
	}
	fmt.Fprintf(os.Stderr, "suiko serve: %s (%d entries) — mcp over stdio\n",
		Store.Manifest.Name, Store.Count())
	if ServeErr := mcpserver.Serve(os.Stdin, os.Stdout, Store); !ServeErr.Ok() {
		fmt.Fprintf(os.Stderr, "serve failed: %s\n", ServeErr.Message)
		os.Exit(ExitInvalid)
	}
}

func usage() {
	fmt.Print(`suiko — MCP-native roleplay engine

usage:
	suiko validate <world-dir>   check a world, report issues
	suiko serve <world-dir>      expose world over MCP (stdio)

exit codes:
	0 valid   1 invalid or unreadable   2 usage error
`)
}
