// chat.ts 封装与后端对话接口的通信。
// 后端统一响应格式：{ code, message, data, trace_id }，code === 0 表示成功。

interface ChatResponse {
  code: number;
  message: string;
  data?: { reply: string };
  trace_id?: string;
}

// ChatMessage 是一轮历史对话。role 与后端/OpenAI 协议对齐：user 或 assistant。
export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

// sendChat 向 /api/v1/chat 发送一条消息，返回 AI 回复文本。
// 走 Vite 的 dev proxy 透传到 Go 服务(8091)，前端直接用相对路径免跨域。
export async function sendChat(
  message: string,
  signal?: AbortSignal
): Promise<string> {
  const res = await fetch("/api/v1/chat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message }),
    signal,
  });

  const body = (await res.json()) as ChatResponse;

  if (body.code !== 0 || !body.data) {
    throw new Error(body.message || "对话失败");
  }
  return body.data.reply;
}

// 一帧 SSE 解析后的结果。event 为 "message"(默认增量) | "done" | "error"。
interface SSEFrame {
  event: string;
  delta?: string;
  message?: string;
}

// parseFrame 解析单个 SSE 帧（形如 "event: xxx\ndata: {json}"）。
function parseFrame(frame: string): SSEFrame {
  let event = "message";
  let data = "";
  for (const line of frame.split("\n")) {
    if (line.startsWith("event:")) event = line.slice(6).trim();
    else if (line.startsWith("data:")) data += line.slice(5).trim();
  }
  try {
    const obj = JSON.parse(data || "{}");
    return { event, delta: obj.delta, message: obj.message };
  } catch {
    return { event };
  }
}

// sendChatStream 以 SSE 流式请求 /api/v1/chat/stream，每收到一段增量就回调 onDelta，
// 用于"边收边渲染"。history 是之前若干轮对话（不含本条），带上以支持多轮上下文。
// 收到 error 帧时抛异常，done 帧或流结束时正常返回。
export async function sendChatStream(
  message: string,
  history: ChatMessage[],
  onDelta: (delta: string) => void,
  signal?: AbortSignal
): Promise<void> {
  const res = await fetch("/api/v1/chat/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message, history }),
    signal,
  });

  if (!res.ok || !res.body) {
    throw new Error("对话失败");
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });

    // SSE 各帧以空行(\n\n)分隔，攒够一帧就取出解析，剩余半帧留在 buffer 里等下一块。
    let idx: number;
    while ((idx = buffer.indexOf("\n\n")) !== -1) {
      const frame = buffer.slice(0, idx);
      buffer = buffer.slice(idx + 2);

      const { event, delta, message: errMsg } = parseFrame(frame);
      if (event === "error") throw new Error(errMsg || "对话失败");
      if (event === "done") return;
      if (delta) onDelta(delta);
    }
  }
}
