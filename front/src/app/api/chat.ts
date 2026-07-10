// chat.ts 封装与后端对话接口的通信。
// 后端统一响应格式：{ code, message, data, trace_id }，code === 0 表示成功。

interface APIResponse {
  code: number;
  message: string;
  trace_id?: string;
}

interface GuestResponse {
  user_id: string;
  created: boolean;
}

const guestUserIDKey = "health_agent_guest_user_id";
const sessionIDKey = "health_agent_session_id";

// createOrResumeGuest 始终请求后端：有效 HttpOnly Cookie 会恢复原 Guest，
// 没有有效凭证时才原子创建新用户。localStorage 中的 user_id 仅兼容尚未切换鉴权的聊天请求。
export async function createOrResumeGuest(): Promise<GuestResponse> {
  const res = await fetch("/api/v1/guest", {
    method: "POST",
    credentials: "include",
  });
  const body = (await res.json()) as APIResponse & { data?: GuestResponse };
  if (!res.ok || body.code !== 0 || !body.data?.user_id) {
    throw new Error(body.message || "创建试用用户失败");
  }

  localStorage.setItem(guestUserIDKey, body.data.user_id);
  return body.data;
}

// ensureGuestUserID 为试用用户申请一个独立 user_id，并在当前浏览器保存它。
// 这里保存的是用户归属，不是会话线程；后续一个 Guest 可以拥有多个 session_id。
export async function ensureGuestUserID(): Promise<string> {
  const existing = localStorage.getItem(guestUserIDKey);
  if (existing) return existing;

  const guest = await createOrResumeGuest();
  return guest.user_id;
}

export function ensureSessionID(): string {
  const existing = localStorage.getItem(sessionIDKey);
  if (existing) return existing;

  const sessionID = `session_${crypto.randomUUID()}`;
  localStorage.setItem(sessionIDKey, sessionID);
  return sessionID;
}

export function createNewSession(): string {
  const sessionID = `session_${crypto.randomUUID()}`;
  localStorage.setItem(sessionIDKey, sessionID);
  return sessionID;
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

// sendChatStream 以 SSE 流式请求 /api/v1/chat/stream，每收到一段增量就回调 onDelta。
// 历史上下文由后端存储层管理，前端只提交本条消息。
// 收到 error 帧时抛异常，done 帧或流结束时正常返回。
export async function sendChatStream(
  userID: string,
  sessionID: string,
  message: string,
  onDelta: (delta: string) => void,
  signal?: AbortSignal
): Promise<void> {
  const res = await fetch("/api/v1/chat/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      user_id: userID,
      session_id: sessionID,
      message,
    }),
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
