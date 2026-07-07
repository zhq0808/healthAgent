import { motion } from "motion/react";
import { X, Check } from "lucide-react";

interface ConfirmationCardProps {
  type: "blood-sugar" | "meal" | "exercise";
  data: {
    label: string;
    value: string;
    unit?: string;
  };
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmationCard({
  type,
  data,
  onConfirm,
  onCancel,
}: ConfirmationCardProps) {
  const getIcon = () => {
    switch (type) {
      case "blood-sugar":
        return "🩸";
      case "meal":
        return "🍽️";
      case "exercise":
        return "🏃";
      default:
        return "📝";
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.95 }}
      className="bg-[#F8F9FA] rounded-2xl p-4 mx-5 mb-4 border border-gray-200"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 flex-1">
          <span className="text-2xl">{getIcon()}</span>
          <div>
            <p className="text-sm text-gray-500">{data.label}</p>
            <p className="text-lg">
              <span className="font-medium">{data.value}</span>
              {data.unit && <span className="text-gray-500 ml-1">{data.unit}</span>}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onConfirm}
            className="w-8 h-8 rounded-full bg-[#A8D5BA] text-white flex items-center justify-center hover:bg-[#95C4A8] transition-colors"
          >
            <Check className="w-4 h-4" />
          </button>
          <button
            onClick={onCancel}
            className="w-8 h-8 rounded-full bg-gray-300 text-gray-600 flex items-center justify-center hover:bg-gray-400 transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>
    </motion.div>
  );
}
