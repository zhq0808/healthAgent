import { motion } from "motion/react";

interface AIMessageProps {
  message: string;
}

export function AIMessage({ message }: AIMessageProps) {
  return (
    <motion.div
      initial={{ opacity: 0, x: -20 }}
      animate={{ opacity: 1, x: 0 }}
      className="flex justify-start mb-4 px-5"
    >
      <div className="bg-white rounded-2xl rounded-tl-md px-4 py-3 max-w-[80%] shadow-[0_2px_20px_rgba(0,0,0,0.04)]">
        <p className="text-sm text-gray-700 leading-relaxed">{message}</p>
      </div>
    </motion.div>
  );
}
