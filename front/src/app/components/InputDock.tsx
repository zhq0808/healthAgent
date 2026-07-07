import { useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { Mic, Send } from "lucide-react";

const PROMPTS = [
  { emoji: "🍱", label: "推荐低GI午餐" },
  { emoji: "📊", label: "看本周进度" },
  { emoji: "💧", label: "喝水打卡" },
];

interface InputDockProps {
  onSendMessage: (message: string) => void;
  onVoiceInput: () => void;
}

export function InputDock({ onSendMessage, onVoiceInput }: InputDockProps) {
  const [input, setInput] = useState("");
  const [isRecording, setIsRecording] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (input.trim()) {
      onSendMessage(input.trim());
      setInput("");
    }
  };

  return (
    <div className="fixed bottom-0 left-0 right-0 bg-gradient-to-t from-[#FAF9F6] via-[#FAF9F6]/96 to-transparent pt-8 pb-7 px-5">
      <div className="flex items-center gap-2 mb-3 overflow-x-auto no-scrollbar">
        {PROMPTS.map((p) => (
          <button
            key={p.label}
            type="button"
            onClick={() => onSendMessage(`${p.emoji} ${p.label}`)}
            className="flex-shrink-0 flex items-center gap-1.5 bg-white rounded-full px-3.5 py-2 text-xs text-gray-600 shadow-[0_2px_12px_rgba(0,0,0,0.05)] hover:bg-gray-50 transition-colors"
          >
            <span>{p.emoji}</span>
            <span>{p.label}</span>
          </button>
        ))}
      </div>

      <form onSubmit={handleSubmit} className="flex items-center gap-2.5">
        <div className="flex-1 relative">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="说说现在的感受..."
            className="w-full bg-white rounded-full px-5 py-3.5 pr-12 shadow-[0_2px_20px_rgba(0,0,0,0.06)] focus:outline-none focus:ring-2 focus:ring-[#A8D5BA]/30 transition-shadow text-sm"
          />
          <AnimatePresence>
            {input && (
              <motion.button
                type="submit"
                initial={{ opacity: 0, scale: 0.7 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.7 }}
                transition={{ duration: 0.15 }}
                className="absolute right-2 top-1/2 -translate-y-1/2 w-8 h-8 rounded-full bg-[#A8D5BA] text-white flex items-center justify-center hover:bg-[#95C4A8] transition-colors"
              >
                <Send className="w-3.5 h-3.5" />
              </motion.button>
            )}
          </AnimatePresence>
        </div>

        <motion.button
          type="button"
          onClick={() => {
            setIsRecording(!isRecording);
            onVoiceInput();
          }}
          animate={{ scale: isRecording ? [1, 1.08, 1] : 1 }}
          transition={{ repeat: isRecording ? Infinity : 0, duration: 1.5 }}
          className={`w-11 h-11 rounded-full flex items-center justify-center shadow-[0_2px_16px_rgba(0,0,0,0.07)] transition-colors flex-shrink-0 ${
            isRecording
              ? "bg-[#F4A460] text-white"
              : "bg-white text-gray-500 hover:bg-gray-50"
          }`}
        >
          <Mic className="w-[18px] h-[18px]" />
        </motion.button>
      </form>
    </div>
  );
}
