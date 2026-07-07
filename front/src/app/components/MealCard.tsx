import { useState } from "react";
import { motion } from "motion/react";
import { Check } from "lucide-react";

const MEALS = [
  { time: "早餐", name: "燕麦粥 + 水煮蛋 + 牛奶", cal: 380, emoji: "🥣", tags: ["高蛋白", "低脂"] },
  { time: "午餐", name: "清蒸鲈鱼 + 糙米饭 + 西兰花", cal: 560, emoji: "🐟", tags: ["均衡"] },
  { time: "加餐", name: "希腊酸奶 + 蓝莓", cal: 140, emoji: "🫐", tags: ["益生菌"] },
  { time: "晚餐", name: "番茄牛肉汤 + 全麦面包", cal: 480, emoji: "🍲", tags: ["饱腹感"] },
];

export function MealCard() {
  const [logged, setLogged] = useState<Set<number>>(new Set());
  const total = MEALS.reduce((s, m) => s + m.cal, 0);

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex justify-start mb-4 px-5"
    >
      <div className="bg-white border border-black/5 rounded-2xl overflow-hidden w-full max-w-[80%] shadow-[0_2px_20px_rgba(0,0,0,0.04)]">
        <div className="px-4 py-3 border-b border-black/5 flex items-center justify-between">
          <span className="text-xs font-semibold text-gray-400 uppercase tracking-wider">今日膳食推荐</span>
          <span className="text-xs text-orange-500 font-semibold">共 {total} kcal</span>
        </div>
        <div className="divide-y divide-black/5">
          {MEALS.map((m, i) => (
            <button
              key={i}
              onClick={() =>
                setLogged((prev) => {
                  const s = new Set(prev);
                  s.has(i) ? s.delete(i) : s.add(i);
                  return s;
                })
              }
              className={`w-full flex items-center gap-3 px-4 py-3 text-left transition-colors ${
                logged.has(i) ? "bg-[#EEF2E8]" : "hover:bg-gray-50"
              }`}
            >
              <span className="text-xl">{m.emoji}</span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5 mb-0.5">
                  <span className="text-[10px] font-semibold text-gray-400 uppercase">{m.time}</span>
                  <span className="text-[10px] text-orange-400 font-medium">{m.cal} kcal</span>
                </div>
                <p className="text-xs font-medium text-gray-700 truncate">{m.name}</p>
              </div>
              <div
                className={`w-5 h-5 rounded-full flex items-center justify-center flex-shrink-0 transition-colors ${
                  logged.has(i) ? "bg-[#A8D5BA] text-white" : "border-2 border-gray-200"
                }`}
              >
                {logged.has(i) && <Check size={10} />}
              </div>
            </button>
          ))}
        </div>
      </div>
    </motion.div>
  );
}
