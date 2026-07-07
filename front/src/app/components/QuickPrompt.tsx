import { motion } from "motion/react";

interface QuickPromptProps {
  message: string;
  onDismiss: () => void;
}

export function QuickPrompt({ message, onDismiss }: QuickPromptProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      className="mb-3"
    >
      <button
        onClick={onDismiss}
        className="w-full bg-white/80 backdrop-blur-sm rounded-2xl px-4 py-3 text-left text-sm text-gray-600 hover:bg-white transition-colors shadow-sm"
      >
        {message}
      </button>
    </motion.div>
  );
}
