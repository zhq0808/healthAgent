import { motion } from "motion/react";
import { CheckCircle2, Circle } from "lucide-react";

interface ActionItem {
  id: string;
  text: string;
  completed: boolean;
}

interface ActionCardProps {
  actions: ActionItem[];
  collapsed?: boolean;
  onToggle: (id: string) => void;
}

export function ActionCard({ actions, collapsed, onToggle }: ActionCardProps) {
  const completedCount = actions.filter((a) => a.completed).length;
  const allDone = completedCount === actions.length;
  const progress = (completedCount / actions.length) * 100;

  if (collapsed) {
    return (
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="mx-5 mb-2.5 px-4 py-2.5 rounded-2xl bg-white/50 border border-[#ECE8DF] flex items-center gap-2.5"
      >
        <span className="text-base">{allDone ? "✅" : "📋"}</span>
        <span className="text-sm text-gray-400 flex-1">
          今日行动清单已打卡 {completedCount}/{actions.length}
        </span>
      </motion.div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-white rounded-3xl shadow-[0_2px_20px_rgba(0,0,0,0.04)] p-5 mx-5 mb-4"
    >
      <div className="flex items-baseline justify-between mb-1">
        <h3 className="text-[17px] font-medium text-gray-800">今日行动清单</h3>
        <span className="text-sm text-gray-400">
          {completedCount}/{actions.length}
        </span>
      </div>

      <div className="mb-5 mt-3">
        <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
          <motion.div
            initial={{ width: 0 }}
            animate={{ width: `${progress}%` }}
            transition={{ duration: 0.5, ease: "easeOut" }}
            className="h-full rounded-full bg-gradient-to-r from-[#A8D5BA] to-[#8BC9A8]"
          />
        </div>
      </div>

      <div className="space-y-1">
        {actions.map((action) => (
          <button
            key={action.id}
            onClick={() => onToggle(action.id)}
            className="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-gray-50 transition-colors text-left"
          >
            {action.completed ? (
              <CheckCircle2 className="w-5 h-5 text-[#A8D5BA] flex-shrink-0" />
            ) : (
              <Circle className="w-5 h-5 text-gray-200 flex-shrink-0" />
            )}
            <span
              className={`text-sm flex-1 ${
                action.completed ? "text-gray-400 line-through" : "text-gray-700"
              }`}
            >
              {action.text}
            </span>
          </button>
        ))}
      </div>
    </motion.div>
  );
}
