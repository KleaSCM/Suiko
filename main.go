/**
 * Suiko Desktop Entry Point — Wails v2 アプリのエントリね。
 *
 * `wails dev` および `wails build` はこのファイルを package main として読む。
 * CLI 動詞（validate / serve）は cmd/suiko/main.go 側に置いたまま。
 * このファイルは Wails バイナリ専用で、ターミナルから直接叩くものじゃないの。
 *
 * DESIGN PHILOSOPHY:
 * App 構造体はここで生まれて wails.Run に渡される。依存はすべてコンストラクタで
 * 注入 — グローバル状態はひとつもない。WorldsDir だけは起動引数かデフォルトパスで
 * 決まるの。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var Assets embed.FS

func main() {
	// worlds/ のデフォルトは実行バイナリ横に置く — ユーザーにとって見つけやすい場所ね。
	WorldsDir := defaultWorldsDir()

	App := &App{
		WorldsDir: WorldsDir,
	}

	if RunErr := wails.Run(&options.App{
		Title:  "Suiko",
		Width:  1400,
		Height: 900,
		AssetServer: &assetserver.Options{
			Assets: Assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 20, B: 28, A: 255},
		OnStartup:        App.Startup,
		Bind: []any{
			App,
		},
	}); RunErr != nil {
		// 起動失敗は即終了。Wails 自体がウィンドウを作れない場合は stderr へ。
		os.Stderr.WriteString("suiko: " + RunErr.Error() + "\n")
		os.Exit(1)
	}
}

// 実行バイナリのディレクトリ横の worlds/ をデフォルトにする。
// 失敗したらカレントディレクトリの worlds/ へフォールバック — 起動を止めないの。
func defaultWorldsDir() string {
	Exe, ExeErr := os.Executable()
	if ExeErr != nil {
		return "worlds"
	}
	return filepath.Join(filepath.Dir(Exe), "worlds")
}
