package main

import (
	"os"
	"path/filepath"
)

func ensureSidecarScript(root string) (string, error) {
	sidecarDir := filepath.Join(root, "sidecar")
	if err := os.MkdirAll(sidecarDir, 0o700); err != nil {
		return "", err
	}
	scriptPath := filepath.Join(sidecarDir, "index.js")
	if err := os.WriteFile(scriptPath, []byte(sidecarIndexJS), 0o600); err != nil {
		return "", err
	}
	return scriptPath, nil
}

const sidecarIndexJS = `
const readline = require("readline");
const {spawn} = require("child_process");
const http = require("http");
const fs = require("fs");
const path = require("path");

const rl = readline.createInterface({
  input: process.stdin,
  terminal: false,
});

// --- IPC 回應工具 ---

// writeResponse: 將 JSON-RPC 回應寫到 stdout，Go 端的 readLoop 會讀取。
function writeResponse(id, result, error, traceID) {
  const payload = error ? {id, error: String(error)} : {id, result};
  const line = JSON.stringify(payload) + String.fromCharCode(10);
  const flushed = process.stdout.write(line, (err) => {
    traceNode("sidecar.response.flush", traceID, {
      id,
      error: err && err.message ? err.message : null,
      bytes: Buffer.byteLength(line),
      framing: "lf-char-code",
    });
  });
  if (!flushed) {
    traceNode("sidecar.response.backpressure", traceID, {
      id,
      bytes: Buffer.byteLength(line),
    });
    process.stdout.once("drain", () => {
      traceNode("sidecar.response.drain", traceID, {id});
    });
  }
}

// DEBUG_TRACE_REMOVE: Temporary sidecar -> local trace viewer bridge.
// Reads AI_CONSOLE_TRACE_URL so restarted apps can move the monitor port.
function traceNode(node, traceID, data) {
  if (!traceID) return;
  try {
    const endpoint = traceEndpoint();
    const body = JSON.stringify({node, trace_id: traceID, data});
    const req = http.request({
      hostname: endpoint.hostname,
      port: endpoint.port,
      path: endpoint.pathname.replace(/\/$/, "") + "/trace",
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Content-Length": Buffer.byteLength(body),
      },
      timeout: 300,
    });
    req.on("error", () => {});
    req.on("timeout", () => req.destroy());
    req.write(body);
    req.end();
  } catch {}
}

function traceEndpoint() {
  try {
    const raw = process.env.AI_CONSOLE_TRACE_URL || "http://127.0.0.1:48765";
    const parsed = new URL(raw);
    return {
      hostname: parsed.hostname || "127.0.0.1",
      port: parsed.port || (parsed.protocol === "https:" ? 443 : 80),
      pathname: parsed.pathname || "",
    };
  } catch {
    return {hostname: "127.0.0.1", port: 48765, pathname: ""};
  }
}

function isPing(text) {
  try {
    const parsed = JSON.parse(text || "{}");
    return parsed && parsed.type === "ping";
  } catch {
    return false;
  }
}

function isGeminiAdapter(adapterID, cliPath) {
  const id = String(adapterID || "").toLowerCase();
  const base = path.basename(cliPath || "").toLowerCase();
  return id.includes("gemini") || base === "gemini";
}

// --- CLI 指令對應表 ---
// 每個 adapter 的 CLI 有不同的參數格式，這裡統一對應。
function commandFor(adapterID, cliPath, prompt) {
  const id = String(adapterID || "").toLowerCase();
  const base = path.basename(cliPath || "").toLowerCase();
  if (id.includes("codex") || base === "codex") {
    return {cmd: cliPath, args: ["exec", "--skip-git-repo-check", prompt]};
  }
  if (id.includes("claude") || base === "claude") {
    return {cmd: cliPath, args: ["-p", prompt]};
  }
  if (id.includes("gemini") || base === "gemini") {
    return {cmd: cliPath, args: ["-p", prompt]};
  }
  return {cmd: cliPath, args: [prompt]};
}

// --- 授權提示偵測 ---

// hasAuthPrompt: 檢查 CLI 輸出是否包含授權/認證相關的互動式提示。
// Gemini CLI 首次執行會輸出類似 "Opening authentication page in your browser..."
// 或 "Would you like to continue? (Y/n)" 的文字。
function hasAuthPrompt(text) {
  return /Opening authentication page|Would you like to continue|Do you want to continue|authenticate your|authentication required/i.test(text || "");
}

// extractAuthURL: 嘗試從 CLI 輸出中擷取 OAuth 授權 URL。
// Gemini CLI 通常會輸出一個 https://accounts.google.com/... 的 URL。
function extractAuthURL(text) {
  const match = (text || "").match(/https:\/\/[^\s"'<>]+(?:accounts\.google\.com|oauth|authorize|login|auth)[^\s"'<>]*/i);
  if (match) return match[0];
  // fallback: 抓任何 https URL（可能不精確，但聊勝於無）
  const anyURL = (text || "").match(/https:\/\/[^\s"'<>]{20,}/);
  return anyURL ? anyURL[0] : "";
}

function publicCLIErrorMessage(err, stdout, stderr) {
  const combined = [err && err.message ? err.message : String(err || ""), stdout || "", stderr || ""].join("\n");
  if (/MODEL_CAPACITY_EXHAUSTED|No capacity available for model/i.test(combined)) {
    const modelMatch = combined.match(/model\s+([A-Za-z0-9._-]+)/i);
    const model = modelMatch ? modelMatch[1] : "目前選用的 Gemini 模型";
    return model + " 目前伺服器容量不足，請稍後重試，或切換到其他可用模型。";
  }
  if (/rateLimitExceeded|RESOURCE_EXHAUSTED|Too Many Requests|status 429/i.test(combined)) {
    return "Gemini CLI 目前受到限流，請稍後重試。";
  }
  const firstUsefulLine = (stderr || stdout || combined)
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find((line) =>
      line &&
      !/^Warning: 256-color support/i.test(line) &&
      !/^Ripgrep is not available/i.test(line)
    );
  return firstUsefulLine || "CLI 執行失敗，請查看監控台取得詳細資訊。";
}

// --- 核心 CLI 執行邏輯 ---

function runCLI(params) {
  return new Promise((resolve, reject) => {
    const adapterID = params.adapter_id || "";
    const cliPath = params.cli_path || "";
    const workspaceDir = params.workspace_dir || "";
    const prompt = params.user_text || "";
    const traceID = params.trace_id || "";
    traceNode("sidecar.runCLI.enter", traceID, {
      adapter_id: adapterID,
      cli_path: cliPath,
      workspace_dir: workspaceDir,
      user_text: prompt,
      has_skill_injection: !!params.skill_injection,
    });

    if (!cliPath) {
      traceNode("sidecar.runCLI.error", traceID, {error: "CLI executable path is missing for adapter " + adapterID});
      reject(new Error("CLI executable path is missing for adapter " + adapterID));
      return;
    }
    if (!fs.existsSync(cliPath)) {
      traceNode("sidecar.runCLI.error", traceID, {error: "CLI executable not found: " + cliPath});
      reject(new Error("CLI executable not found: " + cliPath));
      return;
    }
    if (isPing(prompt)) {
      traceNode("sidecar.runCLI.ping", traceID, {text: "pong:" + adapterID});
      resolve({text: "pong:" + adapterID});
      return;
    }
    if (workspaceDir) {
      try {
        fs.mkdirSync(workspaceDir, {recursive: true, mode: 0o700});
      } catch (err) {
        traceNode("sidecar.runCLI.error", traceID, {error: "Unable to create CLI workspace: " + err.message});
        reject(new Error("Unable to create CLI workspace: " + err.message));
        return;
      }
    }

    const spec = commandFor(adapterID, cliPath, prompt);
    traceNode("sidecar.spawn.command", traceID, {
      cmd: spec.cmd,
      args: spec.args,
      cwd: workspaceDir || process.cwd(),
    });
    const env = {...process.env};
    env.PATH = [
      path.dirname(cliPath),
      "/opt/homebrew/bin",
      "/usr/local/bin",
      "/usr/bin",
      "/bin",
      "/usr/sbin",
      "/sbin",
      env.PATH || "",
    ].filter(Boolean).join(":");
    // SEC-08: 已移除 GEMINI_CLI_TRUST_WORKSPACE=true，讓 Gemini CLI 自行處理 workspace 信任確認。

    // stdin 設為 "pipe"（非 "ignore"），預留給未來需要寫入 stdin 的場景。
    // 但目前偵測到授權提示後會直接 kill child，不會嘗試自動回應 Y/n。
    const child = spawn(spec.cmd, spec.args, {
      cwd: workspaceDir || undefined,
      env,
      stdio: ["pipe", "pipe", "pipe"],
      windowsHide: true,
    });
    traceNode("sidecar.spawn.started", traceID, {
      pid: child.pid,
      cmd: spec.cmd,
      args: spec.args,
      cwd: workspaceDir || process.cwd(),
    });

    let stdout = "";
    let stderr = "";
    let settled = false;
    const limit = 5 * 1024 * 1024;

    function fail(err) {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      try { child.kill("SIGTERM"); } catch {}
      const publicError = publicCLIErrorMessage(err, stdout, stderr);
      traceNode("sidecar.runCLI.fail", traceID, {
        error: publicError,
        raw_error: err && err.message ? err.message : String(err),
        stdout,
        stderr,
      });
      reject(new Error(publicError));
    }
    function finish(value) {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      traceNode("sidecar.runCLI.finish", traceID, value);
      resolve(value);
    }

    // 授權偵測回呼：偵測到授權提示時，不再直接 fail（那樣只會顯示錯誤文字），
    // 而是 resolve 一個帶有 auth_required 標記的特殊回應。
    // Go 端會讀取這個標記，用系統瀏覽器開啟 OAuth URL，並通知前端顯示授權對話框。
    function handleAuthDetected(combinedText) {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      const authURL = extractAuthURL(combinedText);
      // kill 掉卡住的 CLI — 授權完成後會由前端觸發重試
      try { child.kill("SIGTERM"); } catch {}
      traceNode("sidecar.auth.detected", traceID, {
        auth_url: authURL,
        stdout,
        stderr,
      });
      resolve({
        auth_required: true,
        adapter_id: adapterID,
        auth_url: authURL,
        message: adapterID + " 需要瀏覽器授權，請在瀏覽器中完成登入。",
      });
    }

    const timer = setTimeout(() => {
      fail(new Error("CLI timed out after 88s. 如果 CLI 正在等待登入或互動式確認，請先在 Terminal 完成該 CLI 的初始化。"));
    }, 88000);

    child.stdout.on("data", (chunk) => {
      const text = chunk.toString();
      stdout += text;
      if (stdout.length > limit) stdout = stdout.slice(-limit);
      traceNode("sidecar.stdout.chunk", traceID, {text});
      // 即時偵測授權提示，不等 close 事件
      if (hasAuthPrompt(stdout)) {
        handleAuthDetected(stdout + "\n" + stderr);
      }
    });
    child.stderr.on("data", (chunk) => {
      const text = chunk.toString();
      stderr += text;
      if (stderr.length > limit) stderr = stderr.slice(-limit);
      traceNode("sidecar.stderr.chunk", traceID, {text});
      if (hasAuthPrompt(stderr)) {
        handleAuthDetected(stdout + "\n" + stderr);
      }
    });

    // 立刻關閉 child stdin — 我們不會在此輪送任何輸入。
    // 授權完成後會由前端呼叫 retryCLI，重新 spawn 一個新的 CLI process。
    if (child.stdin) child.stdin.end();

    child.on("error", (err) => {
      fail(err);
    });
    child.on("close", (code) => {
      if (settled) return;
      const text = stdout.trim();
      const errText = stderr.trim();
      traceNode("sidecar.close", traceID, {
        code,
        stdout,
        stderr,
      });
      if (code !== 0 && !text) {
        fail(new Error(errText || ("CLI exited with code " + code)));
        return;
      }
      finish({text: text || errText || ""});
    });
  });
}

// --- IPC 請求處理 ---

let requestQueue = Promise.resolve();

async function handleRequest(req) {
  if (!req || !req.id || !req.method) return;
  const traceID = req.params && req.params.trace_id ? req.params.trace_id : "";
  traceNode("sidecar.request.received", traceID, {
    id: req.id,
    method: req.method,
    params: req.params || {},
  });
  try {
    if (req.method !== "sendMessage") {
      throw new Error("unknown method: " + req.method);
    }
    const result = await runCLI(req.params || {});
    traceNode("sidecar.response.write", traceID, {
      id: req.id,
      result,
      error: null,
    });
    writeResponse(req.id, result, null, traceID);
  } catch (err) {
    traceNode("sidecar.response.write", traceID, {
      id: req.id,
      result: null,
      error: err && err.message ? err.message : String(err),
    });
    writeResponse(req.id, null, err && err.message ? err.message : String(err), traceID);
  }
}

rl.on("line", (line) => {
  try {
    const req = JSON.parse(line);
    const traceID = req && req.params && req.params.trace_id ? req.params.trace_id : "";
    traceNode("sidecar.request.queued", traceID, {
      id: req && req.id,
      method: req && req.method,
    });
    requestQueue = requestQueue
      .then(() => handleRequest(req))
      .catch((err) => {
        traceNode("sidecar.request.queue.error", traceID, {
          id: req && req.id,
          error: err && err.message ? err.message : String(err),
        });
      });
  } catch (err) {
    writeResponse("unknown", null, err && err.message ? err.message : String(err), "");
  }
});
`
