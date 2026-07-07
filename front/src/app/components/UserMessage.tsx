import { motion } from "motion/react";

interface UserMessageProps {
  message: string;
}

export function UserMessage({ message }: UserMessageProps) {
  return (
    <motion.div
      initial={{ opacity: 0, x: 20 }}
      animate={{ opacity: 1, x: 0 }}
      className="flex justify-end mb-4 px-5"
    >
      <div className="bg-[#A8D5BA] text-white rounded-2xl rounded-tr-md px-4 py-3 max-w-[80%] shadow-sm">
        <p className="text-sm leading-relaxed">{message}</p>
      </div>
    </motion.div>
  );
}
