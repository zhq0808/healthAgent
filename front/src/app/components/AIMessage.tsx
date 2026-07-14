import { motion } from "motion/react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { RotateCcw } from "lucide-react";

interface AIMessageProps {
  message: string;
  time?: string;
  // failed 为 true 时该气泡对应一次失败发送，展示重试入口。
  failed?: boolean;
  // onRetry 复用原 client_message_id 重新发送；仅失败气泡提供。
  onRetry?: () => void;
}

// AIMessage 渲染助手回复。回复内容可能包含 markdown（加粗、有序/无序列表、链接等），
// 用 react-markdown + remark-gfm 渲染，并通过 arbitrary variant 给嵌套元素补样式，
// 避免额外引入 typography 插件。
// 当内容为空（回复尚未吐字）时，显示三个主题绿色的跳动圆点作为“正在输入”指示。
export function AIMessage({ message, time, failed, onRetry }: AIMessageProps) {
  const isTyping = message.trim().length === 0;

  return (
    <motion.div
      initial={{ opacity: 0, x: -20 }}
      animate={{ opacity: 1, x: 0 }}
      className="flex flex-col items-start mb-4 px-5"
    >
      <div className="bg-white rounded-2xl rounded-tl-md px-4 py-3 max-w-[80%] shadow-[0_2px_20px_rgba(0,0,0,0.04)]">
        {isTyping ? (
          <div className="flex items-center gap-1 py-0.5">
            {[0, 0.18, 0.36].map((delay, i) => (
              <motion.span
                key={i}
                className="w-1.5 h-1.5 rounded-full bg-primary"
                animate={{ opacity: [0.3, 1, 0.3], y: [0, -3, 0] }}
                transition={{
                  duration: 1.1,
                  repeat: Infinity,
                  delay,
                  ease: "easeInOut",
                }}
              />
            ))}
          </div>
        ) : (
          <div className="text-sm leading-relaxed text-gray-700 [&_a]:text-primary [&_a]:underline [&_code]:rounded [&_code]:bg-gray-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-[13px] [&_h1]:mb-1 [&_h1]:text-base [&_h1]:font-semibold [&_h2]:mb-1 [&_h2]:font-semibold [&_h3]:mb-1 [&_h3]:font-semibold [&_li]:mb-1 [&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_p]:mb-2 [&_p:last-child]:mb-0 [&_strong]:font-semibold [&_strong]:text-gray-900 [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{message}</ReactMarkdown>
          </div>
        )}
      </div>
      {!isTyping && time && (
        <span className="mt-1 pl-1 text-[11px] text-muted-foreground">{time}</span>
      )}
      {failed && onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="mt-1.5 flex items-center gap-1 rounded-lg px-1 text-[12px] text-primary hover:underline"
        >
          <RotateCcw size={12} />
          重试
        </button>
      )}
    </motion.div>
  );
}
