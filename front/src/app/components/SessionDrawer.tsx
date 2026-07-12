import { motion, AnimatePresence } from "motion/react";
import { Plus, X, MessageSquare, RotateCcw } from "lucide-react";
import type { SessionListItem } from "../api/chat";

interface SessionDrawerProps {
  open: boolean;
  sessions: SessionListItem[];
  activeSessionID: string | null;
  loading: boolean;
  error: string | null;
  busy: boolean;
  onClose: () => void;
  onSelect: (sessionID: string) => void;
  onCreate: () => void;
  onRetry: () => void;
}

// formatSessionTime 把 RFC3339 时间格式化成简短的中文相对/日期文案。
function formatSessionTime(iso?: string): string {
  if (!iso) return "暂无消息";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "刚刚";
  if (diffMin < 60) return `${diffMin} 分钟前`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour} 小时前`;
  const diffDay = Math.floor(diffHour / 24);
  if (diffDay < 7) return `${diffDay} 天前`;
  return `${date.getMonth() + 1}月${date.getDate()}日`;
}

// SessionDrawer 从左侧滑出的会话列表抽屉：新建、切换、当前态、loading、空态、失败态。
export function SessionDrawer({
  open,
  sessions,
  activeSessionID,
  loading,
  error,
  busy,
  onClose,
  onSelect,
  onCreate,
  onRetry,
}: SessionDrawerProps) {
  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            key="session-backdrop"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-black/20 z-40"
            onClick={onClose}
          />
          <motion.aside
            key="session-panel"
            initial={{ x: "-100%" }}
            animate={{ x: 0 }}
            exit={{ x: "-100%" }}
            transition={{ type: "spring", stiffness: 320, damping: 34 }}
            className="absolute inset-y-0 left-0 z-50 flex w-[82%] max-w-[340px] flex-col bg-background shadow-2xl"
          >
            {/* Header */}
            <div className="flex items-center justify-between px-5 pt-8 pb-4 border-b border-border flex-shrink-0">
              <h2 className="text-[17px] font-semibold text-foreground">会话</h2>
              <button
                type="button"
                onClick={onClose}
                aria-label="关闭"
                className="w-8 h-8 rounded-full bg-secondary flex items-center justify-center hover:bg-accent transition-colors"
              >
                <X size={15} className="text-muted-foreground" />
              </button>
            </div>

            {/* New session */}
            <div className="px-4 py-3 flex-shrink-0">
              <button
                type="button"
                onClick={onCreate}
                disabled={busy}
                className="w-full flex items-center justify-center gap-2 rounded-xl bg-primary text-primary-foreground py-2.5 text-[14px] font-medium transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
              >
                <Plus size={16} />
                新建会话
              </button>
            </div>

            {/* List */}
            <div
              className="flex-1 overflow-y-auto px-3 pb-6 space-y-1"
              style={{ scrollbarWidth: "none" }}
            >
              {loading && (
                <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">
                  正在加载会话...
                </p>
              )}

              {!loading && error && (
                <div className="px-3 py-6 flex flex-col items-center gap-3">
                  <p className="text-[13px] text-destructive">{error}</p>
                  <button
                    type="button"
                    onClick={onRetry}
                    className="flex items-center gap-1.5 rounded-lg bg-secondary px-3 py-1.5 text-[13px] text-primary hover:bg-accent transition-colors"
                  >
                    <RotateCcw size={13} />
                    重试
                  </button>
                </div>
              )}

              {!loading && !error && sessions.length === 0 && (
                <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">
                  还没有会话，点击上方“新建会话”开始吧。
                </p>
              )}

              {!loading &&
                !error &&
                sessions.map((session) => {
                  const active = session.session_id === activeSessionID;
                  const title =
                    session.title && session.title.trim()
                      ? session.title
                      : "新会话";
                  return (
                    <button
                      key={session.session_id}
                      type="button"
                      onClick={() => onSelect(session.session_id)}
                      disabled={busy}
                      className={`w-full flex items-start gap-2.5 rounded-xl px-3 py-2.5 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${
                        active
                          ? "bg-accent"
                          : "hover:bg-secondary"
                      }`}
                    >
                      <MessageSquare
                        size={15}
                        className={`mt-0.5 flex-shrink-0 ${
                          active ? "text-primary" : "text-muted-foreground"
                        }`}
                      />
                      <span className="min-w-0 flex-1">
                        <span
                          className={`block truncate text-[14px] ${
                            active
                              ? "font-semibold text-foreground"
                              : "text-foreground"
                          }`}
                        >
                          {title}
                        </span>
                        <span className="mt-0.5 flex items-center gap-2 text-[11px] text-muted-foreground">
                          <span>{session.message_count} 条消息</span>
                          <span>·</span>
                          <span>
                            {formatSessionTime(session.last_message_at)}
                          </span>
                        </span>
                      </span>
                    </button>
                  );
                })}
            </div>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  );
}
