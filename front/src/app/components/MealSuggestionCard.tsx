import { motion } from "motion/react";
import { Check, RefreshCw } from "lucide-react";

interface MealSuggestionCardProps {
  meal: string;
  title: string;
  emoji: string;
  description: string;
  ingredients: string[];
  benefits: string;
  gi?: number;
  collapsed?: boolean;
  onAccept: () => void;
  onRegenerate: () => void;
}

export function MealSuggestionCard({
  meal,
  title,
  emoji,
  description,
  ingredients,
  benefits,
  gi,
  collapsed,
  onAccept,
  onRegenerate,
}: MealSuggestionCardProps) {
  if (collapsed) {
    return (
      <motion.div
        initial={{ opacity: 0, height: 80 }}
        animate={{ opacity: 1, height: "auto" }}
        className="mx-5 mb-2.5 px-4 py-2.5 rounded-2xl bg-white/50 border border-[#ECE8DF] flex items-center gap-2.5"
      >
        <span className="text-xl">{emoji}</span>
        <span className="text-sm text-gray-400 flex-1 truncate">
          {meal} · {title}
        </span>
        <span className="text-xs text-gray-300 flex-shrink-0">已折叠</span>
      </motion.div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-white rounded-3xl shadow-[0_2px_20px_rgba(0,0,0,0.04)] p-5 mx-5 mb-4"
    >
      <div className="flex items-start gap-4 mb-4">
        <div className="w-14 h-14 rounded-2xl bg-[#F5F7F3] flex items-center justify-center text-3xl flex-shrink-0">
          {emoji}
        </div>
        <div className="flex-1 min-w-0 pt-0.5">
          <div className="text-xs text-gray-400 mb-1">{meal}</div>
          <h3 className="text-[17px] font-medium text-gray-800 leading-snug mb-2">
            {title}
          </h3>
          {gi !== undefined && (
            <span className="inline-flex items-center px-2 py-0.5 rounded-full bg-[#EEF9F4] text-[#3A6F5A] text-xs">
              GI · {gi}
            </span>
          )}
        </div>
      </div>

      <p className="text-gray-500 text-sm leading-relaxed mb-4">{description}</p>

      <div className="flex flex-wrap gap-1.5 mb-4">
        {ingredients.map((ing, i) => (
          <span
            key={i}
            className="px-2.5 py-1 bg-[#F5F7F3] rounded-lg text-sm text-gray-600"
          >
            {ing}
          </span>
        ))}
      </div>

      <div className="mb-5 p-3.5 bg-[#EEF9F4] rounded-2xl">
        <p className="text-sm text-[#3A6F5A] leading-relaxed">{benefits}</p>
      </div>

      <div className="flex gap-2.5">
        <button
          onClick={onAccept}
          className="flex-1 flex items-center justify-center gap-2 bg-[#A8D5BA] text-white px-4 py-3 rounded-xl hover:bg-[#95C4A8] transition-colors text-sm"
        >
          <Check className="w-4 h-4" />
          已按建议吃
        </button>
        <button
          onClick={onRegenerate}
          className="flex items-center justify-center gap-2 bg-gray-100 text-gray-600 px-4 py-3 rounded-xl hover:bg-gray-200 transition-colors text-sm"
        >
          <RefreshCw className="w-4 h-4" />
          换一个
        </button>
      </div>
    </motion.div>
  );
}
