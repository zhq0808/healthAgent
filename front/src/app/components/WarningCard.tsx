import { motion } from "motion/react";
import { AlertCircle } from "lucide-react";

interface WarningCardProps {
  title: string;
  message: string;
  reason: string;
}

export function WarningCard({ title, message, reason }: WarningCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      className="bg-white rounded-2xl border-2 border-[#F4A460] shadow-[0_2px_20px_rgba(244,164,96,0.12)] p-5 mx-5 mb-4"
    >
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 w-10 h-10 rounded-full bg-[#FFF5E6] flex items-center justify-center">
          <AlertCircle className="w-5 h-5 text-[#F4A460]" />
        </div>
        <div className="flex-1">
          <h4 className="text-lg text-[#B8722C] mb-1">{title}</h4>
          <p className="text-gray-700 mb-3 leading-relaxed">{message}</p>
          <div className="bg-[#FFF9F0] rounded-lg p-3">
            <p className="text-sm text-[#A67C52] leading-relaxed">💡 {reason}</p>
          </div>
        </div>
      </div>
    </motion.div>
  );
}
