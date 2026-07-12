import { useRef, useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { Mic, Send, Plus, Square } from "lucide-react";

const PROMPTS = [
  { emoji: "🍱", label: "推荐低GI午餐" },
  { emoji: "📊", label: "看本周进度" },
  { emoji: "💧", label: "喝水打卡" },
];

interface InputDockProps {
  onSendMessage: (message: string) => void;
  onVoiceInput: () => void;
  onPhoto: (file: File) => void;
  isResponding: boolean;
  onStop: () => void;
}

export function InputDock({
  onSendMessage,
  onVoiceInput,
  onPhoto,
  isResponding,
  onStop,
}: InputDockProps) {
  const [input, setInput] = useState("");
  const [isRecording, setIsRecording] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (input.trim()) {
      onSendMessage(input.trim());
      setInput("");
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      onPhoto(file);
    }
    // 重置，允许连续选择同一文件时也能再次触发 change。
    e.target.value = "";
  };

  return (
    <div className="fixed bottom-0 left-0 right-0 bg-gradient-to-t from-[#F6F8F4] via-[#F6F8F4]/96 to-transparent pt-8 pb-7 px-5">
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
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          capture="environment"
          className="hidden"
          onChange={handleFileChange}
        />
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          aria-label="拍照 / 上传图片"
          title="拍照 / 上传图片"
          className="w-11 h-11 rounded-full flex items-center justify-center shadow-[0_2px_16px_rgba(0,0,0,0.07)] bg-white text-gray-500 hover:bg-gray-50 transition-colors flex-shrink-0"
        >
          <Plus className="w-[20px] h-[20px]" />
        </button>

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

        {isResponding ? (
          <motion.button
            type="button"
            onClick={onStop}
            aria-label="停止回答"
            title="停止回答"
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            className="w-11 h-11 rounded-full flex items-center justify-center shadow-[0_2px_16px_rgba(0,0,0,0.07)] bg-primary text-white transition-colors flex-shrink-0"
          >
            <Square className="w-4 h-4" fill="currentColor" />
          </motion.button>
        ) : (
          <motion.button
            type="button"
            onClick={() => {
              setIsRecording(!isRecording);
              onVoiceInput();
            }}
            aria-label="语音输入"
            title="语音输入"
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
        )}
      </form>
    </div>
  );
}
