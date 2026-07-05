// 与后端交互的封装。dev 下请求经 Vite 代理转发到 Go 服务（见 vite.config.ts 的 proxy）。

// 后端统一响应结构：{ code, message, data, trace_id }
interface ApiResponse<T> {
  code: number;
  message: string;
  data?: T;
  trace_id?: string;
}

// sendChat 发送一条对话消息，返回 AI 回复文本。
// 失败时抛错，交给调用方决定如何在 UI 上兜底。
export async function sendChat(message: string): Promise<string> {
  const res = await fetch("/api/v1/chat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message }),
  });
  const body = (await res.json()) as ApiResponse<{ reply: string }>;
  if (!res.ok || body.code !== 0) {
    throw new Error(body.message || `HTTP ${res.status}`);
  }
  return body.data?.reply ?? "";
}
