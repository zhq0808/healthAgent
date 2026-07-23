import { useRef, useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { Mic, Send, Plus, Square, Sparkles, ChevronDown, Check } from "lucide-react";
import type { ModelOption } from "../api/chat";

const PROMPTS = [
  { emoji: "🧠", label: "费曼学习" },
  { emoji: "🔄", label: "知识点回顾" },
  { emoji: "💬", label: "模拟面试" },
  { emoji: "🎯", label: "JD 分析" },
];

interface InputDockProps {
  onSendMessage: (message: string) => void;
  onVoiceInput: () => void;
  onPhoto: (file: File) => void;
  isResponding: boolean;
  onStop: () => void;
  models: ModelOption[];
  selectedModelID: string;
  onSelectModel: (modelID: string) => void;
}

export function InputDock({
  onSendMessage,
  onVoiceInput,
  onPhoto,
  isResponding,
  onStop,
  models,
  selectedModelID,
  onSelectModel,
}: InputDockProps) {
  const [input, setInput] = useState("");
  const [isRecording, setIsRecording] = useState(false);
  const [modelMenuOpen, setModelMenuOpen] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const selectedModel =
    models.find((m) => m.id === selectedModelID) ?? models[0];

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
    <div className="absolute bottom-[80px] left-0 right-0 z-30 bg-gradient-to-t from-[#F6F8F4] via-[#F6F8F4]/96 to-transparent px-5 pb-4 pt-8">
      {/* 快捷提示行；模型选择器放在同一行最左侧（在滚动容器之外，避免向上弹出的菜单被裁剪）。 */}
      <div className="flex items-center gap-2 mb-3">
        {/* 模型选择。当前仅前端选择与本地记忆，后端支持后再随请求下发。 */}
        <div className="relative flex-shrink-0">
          <button
            type="button"
            onClick={() => setModelMenuOpen((v) => !v)}
            className="flex items-center gap-1.5 rounded-full bg-white px-3 py-2 text-xs text-gray-600 shadow-[0_2px_12px_rgba(0,0,0,0.05)] hover:bg-gray-50 transition-colors"
          >
            <Sparkles size={13} className="text-primary" />
            <span className="font-medium">{selectedModel?.name}</span>
            <ChevronDown
              size={13}
              className={`opacity-50 transition-transform ${
                modelMenuOpen ? "rotate-180" : ""
              }`}
            />
          </button>

          {modelMenuOpen && (
            <>
              <div
                className="fixed inset-0 z-10"
                onClick={() => setModelMenuOpen(false)}
              />
              <div className="absolute bottom-full left-0 z-20 mb-1.5 w-56 overflow-hidden rounded-2xl border border-black/5 bg-white py-1 shadow-[0_8px_32px_rgba(0,0,0,0.12)]">
                {models.map((model) => {
                  const active = model.id === selectedModelID;
                  return (
                    <button
                      key={model.id}
                      type="button"
                      onClick={() => {
                        onSelectModel(model.id);
                        setModelMenuOpen(false);
                      }}
                      className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-gray-50"
                    >
                      <span className="min-w-0 flex-1">
                        <span className="block text-[13px] font-medium text-gray-800">
                          {model.name}
                        </span>
                        <span className="block truncate text-[11px] text-gray-400">
                          {model.desc}
                        </span>
                      </span>
                      {active && (
                        <Check size={15} className="flex-shrink-0 text-primary" />
                      )}
                    </button>
                  );
                })}
              </div>
            </>
          )}
        </div>

        {/* 快捷提示（可横向滚动） */}
        <div className="flex items-center gap-2 overflow-x-auto no-scrollbar">
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
            placeholder="输入想练习的知识点或面试问题..."
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
