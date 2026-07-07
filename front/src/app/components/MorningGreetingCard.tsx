import { motion } from "motion/react";

interface MorningGreetingCardProps {
  onReply: (reply: "recovered" | "still-tired") => void;
}

export function MorningGreetingCard({ onReply }: MorningGreetingCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.95, y: -8 }}
      transition={{ duration: 0.3 }}
      className="mx-5 mb-4 p-5 rounded-3xl bg-gradient-to-br from-[#EFF7F3] to-[#FEF8F0] shadow-[0_2px_20px_rgba(0,0,0,0.04)]"
    >
      <div className="text-[10px] text-[#A8C4B4] mb-3 font-medium tracking-widest uppercase">
        清晨问候
      </div>

      <p className="text-[#3D5A4A] leading-relaxed mb-5 text-[15px]">
        早安！昨天看你比较累，今天有缓解一些吗？☀️
      </p>

      <div className="flex gap-2.5">
        <button
          onClick={() => onReply("recovered")}
          className="flex-1 py-2.5 px-3 rounded-2xl bg-[#A8D5BA]/25 text-[#3D6B52] text-sm hover:bg-[#A8D5BA]/45 transition-colors font-medium"
        >
          ✨ 满血复活
        </button>
        <button
          onClick={() => onReply("still-tired")}
          className="flex-1 py-2.5 px-3 rounded-2xl bg-white/60 text-[#7A6654] text-sm hover:bg-white/90 transition-colors border border-[#EAE2D8] font-medium"
        >
          🫠 还是有点累
        </button>
      </div>
    </motion.div>
  );
}
