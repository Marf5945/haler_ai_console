package main

// popout_child.go — 釘選對話框（popout）子行程端。
//
// 以 `exe --popout --popout-agent=… --popout-port=… --popout-token=…` 啟動，
// 開一個小型獨立 OS 視窗（可拖到任何螢幕位置）。子行程是純顯示殼：
// 不碰記憶檔、不碰模型、不跑排程/sidecar；一切透過 loopback HTTP 問主行程。
//
// 前端如何知道自己是 popout：AssetServer middleware 在 index.html 注入
// <meta name="haler-popout" content="<base64 JSON>">（meta 不受 CSP script-src 限制），
// main.jsx 讀到即改渲染 PopoutChat。

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type popoutArgs struct {
	agent string
	name  string
	port  int
	token string
}

// parsePopoutArgs 從命令列判斷是否為 popout 模式；不是則回 nil。
func parsePopoutArgs(args []string) *popoutArgs {
	out := &popoutArgs{}
	isPopout := false
	for _, arg := range args {
		switch {
		case arg == "--popout":
			isPopout = true
		case strings.HasPrefix(arg, "--popout-agent="):
			out.agent = strings.TrimPrefix(arg, "--popout-agent=")
		case strings.HasPrefix(arg, "--popout-name="):
			out.name = strings.TrimPrefix(arg, "--popout-name=")
		case strings.HasPrefix(arg, "--popout-port="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--popout-port="), "%d", &out.port)
		case strings.HasPrefix(arg, "--popout-token="):
			out.token = strings.TrimPrefix(arg, "--popout-token=")
		}
	}
	if !isPopout || out.agent == "" || out.port <= 0 || out.token == "" {
		return nil
	}
	if out.name == "" {
		out.name = out.agent
	}
	return out
}

// PopoutApp 是子行程綁給前端的橋：全部轉發到主行程 loopback API。
type PopoutApp struct {
	ctx   context.Context
	args  *popoutArgs
	base  string
	httpc *http.Client // Send 用：LLM 可能很慢，不設 timeout
	quick *http.Client // state/ping/unpin 用：短 timeout
}

func newPopoutApp(args *popoutArgs) *PopoutApp {
	return &PopoutApp{
		args:  args,
		base:  fmt.Sprintf("http://127.0.0.1:%d", args.port),
		httpc: &http.Client{},
		quick: &http.Client{Timeout: 5 * time.Second},
	}
}

func (p *PopoutApp) do(client *http.Client, method, path string, body interface{}) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, p.base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Popout-Token", p.args.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("main process replied %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

// GetState 取視窗初始資料（人格/模型標籤 + 該對話自己的獨立歷史）。
func (p *PopoutApp) GetState() (map[string]interface{}, error) {
	raw, err := p.do(p.quick, http.MethodGet, "/api/state?agent="+p.args.agent, nil)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Send 送一句話：主行程負責寫入該 agent 的獨立記憶並呼叫模型。
func (p *PopoutApp) Send(text string) (map[string]interface{}, error) {
	raw, err := p.do(p.httpc, http.MethodPost, "/api/send", map[string]string{
		"agent": p.args.agent,
		"text":  text,
	})
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Unpin 解除釘選：通知主行程回彈分頁，然後自我了斷。
func (p *PopoutApp) Unpin() {
	p.notifyUnpin()
	if p.ctx != nil {
		runtime.Quit(p.ctx)
	}
}

func (p *PopoutApp) notifyUnpin() {
	_, _ = p.do(p.quick, http.MethodPost, "/api/unpin", map[string]string{"agent": p.args.agent})
}

func (p *PopoutApp) startup(ctx context.Context) {
	p.ctx = ctx
	// 心跳：主行程掛了就跟著退場，避免殭屍小視窗。
	go func() {
		failures := 0
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := p.do(p.quick, http.MethodGet, "/api/ping", nil); err != nil {
				failures++
				if failures >= 3 {
					runtime.Quit(ctx)
					return
				}
				continue
			}
			failures = 0
		}
	}()
}

// popoutMetaMiddleware 在 index.html 注入 popout 識別 meta（CSP 安全）。
func popoutMetaMiddleware(assets fs.FS, args *popoutArgs) assetserver.Middleware {
	payload, _ := json.Marshal(map[string]string{
		"agent": args.agent,
		"name":  args.name,
	})
	meta := `<meta name="haler-popout" content="` + base64.StdEncoding.EncodeToString(payload) + `">`
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" || path == "index.html" {
				raw, err := fs.ReadFile(assets, "index.html")
				if err == nil {
					html := strings.Replace(string(raw), "<head>", "<head>"+meta, 1)
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					_, _ = w.Write([]byte(html))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// runPopoutWindow 以獨立小視窗跑 popout 模式（取代主控台啟動流程）。
func runPopoutWindow(frontendAssets fs.FS, args *popoutArgs) error {
	app := newPopoutApp(args)
	return wails.Run(&options.App{
		Title:     args.name + " — HaLer",
		Width:     460,
		Height:    700,
		MinWidth:  360,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: frontendAssets,
			Middleware: func(next http.Handler) http.Handler {
				// 先注入 popout meta，再套與主視窗相同的 CSP。
				return cspMiddleware(popoutMetaMiddleware(frontendAssets, args)(next))
			},
		},
		BackgroundColour: &options.RGBA{R: 5, G: 5, B: 5, A: 255},
		OnStartup:        app.startup,
		// 使用者直接關窗＝解除釘選：先通知主行程回彈，再放行關閉。
		OnBeforeClose: func(ctx context.Context) bool {
			app.notifyUnpin()
			return false
		},
		// 注意：popout 子行程「不可」設 SingleInstanceLock，
		// 否則會被主行程的單一實例鎖擋下而直接退出。
		Bind: []interface{}{app},
	})
}
