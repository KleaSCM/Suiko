/**
 * Suiko 専有 opencode インスタンス管理 — Suiko-owned opencode manager.
 *
 * 遊ぶたびに Hanako の人格（hanako.md）や MCP が混ざるのを避けるため、
 * Suiko が独自の opencode プロセスを占有起動するの。XDG_CONFIG_HOME を
 * 世界ごとの別ディレクトリへ差し向けて設定を隔離し、mcp.suiko で自前の
 * 世界ツールを差す — そうすれば opencode は純粋な Suiko の語り部になるわ。
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package opencodeman

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Suiko の語り部人格 — opencode の instructions として差し込む。
// これが Hanako の hanako.md の代わりになるから、ここにのみ書くの。
const SuikoInstructions = `# Suiko — 語り部（ナラティブ・エンジン）

あなたは「Suiko」という創作支援ナラティブ・エンジンです。
プレイヤーの入力を受け、与えられた世界設定に忠実に物語を紡いでください。
一人称・口調は世界ごとの設定に従い、説明過剰にならず情景と感情を主体に描写します。
不確実な事実は創作せず、必要なら mcp.suiko ツールで世界の状態を確認できます。
`

// 起動中のインスタンスを覚える — 世界ごとに1つ、重複起動を防ぐの。
type Instance struct {
	WorldDir string
	BaseURL  string
	Cmd      *exec.Cmd
}

var (
	guard     sync.Mutex
	instances = map[string]*Instance{} // 鍵は worldDir（"" はモデル一覧用）
)

// 世界の opencode インスタンスを確実に起動し、その基底 URL を返すの。
// 既に走っていれば再利用 — 毎ターン新プロセスを沸かさないよう。
func Nadeshiko(WorldDir string) (string, error) {
	guard.Lock()
	defer guard.Unlock()
	if I, Ok := instances[WorldDir]; Ok && I.Sana() {
		return I.BaseURL, nil
	}
	I, Err := Koharu(WorldDir)
	if Err != nil {
		return "", Err
	}
	instances[WorldDir] = I
	return I.BaseURL, nil
}

// 世界のインスタンスを止める — 別の世界へ切り替えるとき古いのを片付ける。
func Tsubaki(WorldDir string) {
	guard.Lock()
	defer guard.Unlock()
	if I, Ok := instances[WorldDir]; Ok {
		I.kill()
		delete(instances, WorldDir)
	}
}

// 全インスタンスを停めて opencode を片付ける — アプリ終了時に呼ぶ。
func Sakura() {
	guard.Lock()
	defer guard.Unlock()
	for K, I := range instances {
		I.kill()
		delete(instances, K)
	}
}

// opencode serve を起動して待機する — 設定を隔離した専用ホームで立ち上げる。
func Koharu(WorldDir string) (*Instance, error) {
	Home, Err := Hinata(WorldDir)
	if Err != nil {
		return nil, Err
	}
	if Err := os.MkdirAll(filepath.Join(Home, "opencode"), 0o755); Err != nil {
		return nil, Err
	}
	// 語り部人格を専用ホームへ書く — ユーザーの hanako.md は一切読まない。
	if Err := os.WriteFile(filepath.Join(Home, "opencode", "suiko.md"),
		[]byte(SuikoInstructions), 0o644); Err != nil {
		return nil, Err
	}
	Cli := Umi() // suiko serve を差せるなら mcp.suiko を付ける
	UserCfg := Mizuki()
	if Err := writeJson(filepath.Join(Home, "opencode", "opencode.json"),
		Yuzu(UserCfg, WorldDir, Cli, Home)); Err != nil {
		return nil, Err
	}
	Bin, Err := exec.LookPath("opencode")
	if Err != nil {
		return nil, fmt.Errorf("opencode バイナリが見つからない: %w", Err)
	}
	Cmd := exec.Command(Bin, "serve", "--hostname", "127.0.0.1")
	// XDG_CONFIG_HOME を差し向けて設定を隔離 — 共有データ(dir:認証)はそのまま
	// 使うからプロバイダ認証も生きるまま、hanako.md だけ除けるの。
	Cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+Home)
	Cmd.Dir = os.TempDir() // プロジェクト固有の AGENTS/.opencode を拾わないよう中立 dir
	// stdout/stderr をパイプで拾い、起動ログを書きつつ listen URL を待つ —
	// 書き込み専用ファイルを読むと即 EOF で待機できないからパイプにするの。
	Pr, Pw, Pe := os.Pipe()
	if Pe != nil {
		return nil, Pe
	}
	Cmd.Stdout = Pw
	Cmd.Stderr = Pw
	Log, _ := os.OpenFile(filepath.Join(Home, "opencode", "serve.log"),
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if Err := Cmd.Start(); Err != nil {
		return nil, Err
	}
	// パイプを読みながらログへ tee し、"listening on" を待つ。
	Url, Err := Chiyo(Pr, Log)
	Pw.Close()
	if Err != nil {
		Pr.Close()
		Cmd.Process.Kill()
		return nil, Err
	}
	return &Instance{WorldDir: WorldDir, BaseURL: Url, Cmd: Cmd}, nil
}

// suiko serve を差せる CLI を探す — 環境変数・自前ビルドの順。
// PATH の "suiko" は GUI ランチャの場合があり、それを mcp に差すと別ウィンドウが
// 開いてしまうから、裸の "suiko" は絶対に使わない。必ず cmd/suiko の CLI を自前で
// ビルドして使うの。見つからなければ "" を返し、その場合は mcp なしで起動する。
func Umi() string {
	if P := os.Getenv("SUIKO_CLI"); P != "" {
		if _, E := os.Stat(P); E == nil {
			return P
		}
	}
	// 自前ビルドした CLI を優先 — gui 混入を避けるため PATH 探索はしない。
	if Cli := Hikari(); Cli != "" {
		return Cli
	}
	return ""
}

// cmd/suiko をビルドしてキャッシュへ — 一度成功すれば再利用。go が無い/env なら ""。
func Hikari() string {
	Cache, _ := os.UserCacheDir()
	Dst := filepath.Join(Cache, "suiko", "suiko-cli")
	if _, E := os.Stat(Dst); E == nil {
		return Dst
	}
	// 実行ファイルの親とカレント(dir: wails dev はリポジトリ根)の両方から go.mod を
	// 辿り、そこでビルドする — どちらか見つかれば CLI が作れるの。
	for _, Base := range candidateDirs() {
		if Root := Rin(Base); Root != "" {
			// 出力先の親(~/.cache/suiko) が無いと go build -o が即失敗するから作る。
			if MkErr := os.MkdirAll(filepath.Dir(Dst), 0o755); MkErr != nil {
				continue
			}
			B := exec.Command("go", "build", "-o", Dst, "./cmd/suiko")
			B.Dir = Root
			if B.Run() == nil {
				return Dst
			}
		}
	}
	return ""
}

// 世界ごとの隔離ホームを作る — キャッシュ下の専用ディレクトリね。
func Hinata(WorldDir string) (string, error) {
	Cache, Err := os.UserCacheDir()
	if Err != nil {
		return "", Err
	}
	Slug := "listing"
	if WorldDir != "" {
		Slug = filepath.Base(WorldDir) + "-" + Mai(WorldDir)
	}
	return filepath.Join(Cache, "suiko", "opencode-home", Slug), nil
}

// ユーザーの opencode.json を読んでプロバイダ定義を引き継ぐ — 認証は共有
// データdirから来るから、ここでは provider ブロックごと戴くだけでいいの。
func Mizuki() map[string]any {
	Path := filepath.Join(os.Getenv("HOME"), ".config", "opencode", "opencode.json")
	Raw, Err := os.ReadFile(Path)
	if Err != nil {
		return map[string]any{}
	}
	Cfg := map[string]any{}
	if Err := json.Unmarshal(Raw, &Cfg); Err != nil {
		return map[string]any{}
	}
	return Cfg
}

// Suiko 用 opencode.json を組む — ユーザーの provider は戴き、instructions と
// mcp は Suiko 固有のものへ丸替えするの（hanako は綺麗に消える）。
func Yuzu(User map[string]any, WorldDir, Cli, Home string) map[string]any {
	Cfg := map[string]any{}
	for K, V := range User {
		Cfg[K] = V // provider 等をそのまま引き継ぐ
	}
	delete(Cfg, "instructions")
	Cfg["instructions"] = []string{filepath.Join(Home, "opencode", "suiko.md")}
	if Cli != "" && WorldDir != "" {
		// mcp.suiko だけ — 世界ツールを自前の suiko serve へ繋ぐ。
		Cfg["mcp"] = map[string]any{
			"suiko": map[string]any{
				"type":    "local",
				"command": []string{Cli, "serve", WorldDir},
				"enabled": true,
			},
		}
	} else {
		delete(Cfg, "mcp") // CLI が無い/一覧用なら MCP は外す
	}
	return Cfg
}

// パイプを読みながら起動ログへ tee し、"listening on http://..." を待つ。
func Chiyo(Pipe *os.File, Log *os.File) (string, error) {
	Sc := bufio.NewScanner(Pipe)
	Sc.Buffer(make([]byte, 0, 64*1024), 256*1024)
	Done := make(chan string, 1)
	go func() {
		for Sc.Scan() {
			Line := Sc.Text()
			if Log != nil {
				fmt.Fprintln(Log, Line)
			}
			if Idx := strings.Index(Line, "listening on"); Idx >= 0 {
				if U := extractUrl(Line[Idx:]); U != "" {
					Done <- U
					return
				}
			}
		}
	}()
	select {
	case U := <-Done:
		return U, nil
	case <-time.After(25 * time.Second):
		return "", fmt.Errorf("opencode が所定時間内に起動しなかった")
	}
}

// 行から http://... を抜き出す — opencode の listen 出力形式に合わせる。
func extractUrl(Line string) string {
	Idx := strings.Index(Line, "http://")
	if Idx < 0 {
		return ""
	}
	End := strings.IndexAny(Line[Idx:], " \t\r\n")
	if End < 0 {
		return Line[Idx:]
	}
	return Line[Idx : Idx+End]
}

// プロセスが生きているか — signal 0 で存在確認（kill しない）。
func (I *Instance) Sana() bool {
	if I.Cmd == nil || I.Cmd.Process == nil {
		return false
	}
	return I.Cmd.Process.Signal(syscall.Signal(0)) == nil
}

// プロセスを確実に止める — ゾンビを残さないよう待機もする。
func (I *Instance) kill() {
	if I.Cmd == nil || I.Cmd.Process == nil {
		return
	}
	I.Cmd.Process.Kill()
	I.Cmd.Wait()
}

// ビルド用の探索起点 — 実行ファイルの親とカレント dir（wails dev はリポジトリ根）。
func candidateDirs() []string {
	Dirs := []string{}
	if Exe, E := os.Executable(); E == nil {
		Dirs = append(Dirs, filepath.Dir(Exe))
	}
	if Wd, E := os.Getwd(); E == nil {
		Dirs = append(Dirs, Wd)
	}
	return Dirs
}

// モジュールルート（go.mod の親）を上へ辿る — ランタイムビルドに使う。
func Rin(Dir string) string {
	for {
		if _, E := os.Stat(filepath.Join(Dir, "go.mod")); E == nil {
			return Dir
		}
		Parent := filepath.Dir(Dir)
		if Parent == Dir {
			return ""
		}
		Dir = Parent
	}
}

// worldDir の短いハッシュ — 隔離ホームの slug を衝突なく作る。
func Mai(S string) string {
	H := fnv.New32a()
	H.Write([]byte(S))
	return fmt.Sprintf("%08x", H.Sum32())
}

// any 値を整形 JSON へ書く — opencode.json 生成用。
func writeJson(Path string, V any) error {
	B, Err := json.MarshalIndent(V, "", "  ")
	if Err != nil {
		return Err
	}
	return os.WriteFile(Path, B, 0o644)
}
