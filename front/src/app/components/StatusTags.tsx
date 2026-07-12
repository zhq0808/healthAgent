import { useState, useRef, useEffect } from "react";
import { motion, AnimatePresence } from "motion/react";
import { ChevronDown } from "lucide-react";
import { LineChart, Line, ResponsiveContainer } from "recharts";

export interface StatusTagDef {
  id: string;
  emoji: string;
  label: string;
  color: string;
  state: "active" | "pending" | "dismissed";
  sparklineData: { v: number }[];
  summary: string;
  // expandable 为 false 时该标签只作展示，不显示下拉箭头，也不弹出趋势卡片。默认可展开。
  expandable?: boolean;
}

interface StatusTagsProps {
  tags: StatusTagDef[];
}

export function StatusTags({ tags }: StatusTagsProps) {
  // 默认全部收起，用户点击标签才展开对应的小卡片。
  const [openId, setOpenId] = useState<string | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handle = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpenId(null);
      }
    };
    document.addEventListener("mousedown", handle);
    return () => document.removeEventListener("mousedown", handle);
  }, []);

  const visible = tags.filter((t) => t.state !== "dismissed");

  return (
    <div ref={ref} className="relative flex items-center gap-2 px-5 pt-6 pb-3">
      <AnimatePresence>
        {visible.map((tag, i) => {
          const canExpand = tag.expandable !== false;
          return (
          <motion.div
            key={tag.id}
            className="relative"
            initial={{ opacity: 0, y: -8, scale: 0.9 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, scale: 0.8, y: -6 }}
            transition={{ delay: i * 0.08, duration: 0.25 }}
          >
            <button
              onClick={
                canExpand
                  ? () => setOpenId(openId === tag.id ? null : tag.id)
                  : undefined
              }
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm transition-all select-none ${
                canExpand ? "" : "cursor-default"
              } ${
                tag.state === "pending"
                  ? "bg-gray-100 text-gray-400"
                  : tag.color
              }`}
            >
              <span className={tag.state === "pending" ? "opacity-40" : ""}>
                {tag.emoji}
              </span>
              <span>{tag.label}</span>
              {tag.state === "pending" && (
                <span className="text-gray-400 text-xs font-medium">?</span>
              )}
              {canExpand && (
                <ChevronDown
                  className={`w-3 h-3 opacity-40 transition-transform duration-200 ${
                    openId === tag.id ? "rotate-180" : ""
                  }`}
                />
              )}
            </button>

            {canExpand && (
              <AnimatePresence>
                {openId === tag.id && (
                  <motion.div
                    initial={{ opacity: 0, y: -6, scale: 0.94 }}
                    animate={{ opacity: 1, y: 0, scale: 1 }}
                    exit={{ opacity: 0, y: -6, scale: 0.94 }}
                    transition={{ duration: 0.16 }}
                    className="absolute top-full left-0 mt-2 z-50 w-52 bg-white rounded-2xl p-4 shadow-[0_8px_32px_rgba(0,0,0,0.11)]"
                  >
                    <div className="h-10 mb-3">
                      <ResponsiveContainer width="100%" height="100%">
                        <LineChart
                          data={tag.sparklineData}
                          margin={{ top: 2, right: 4, bottom: 2, left: 4 }}
                        >
                          <Line
                            type="monotone"
                            dataKey="v"
                            stroke={
                              tag.state === "pending" ? "#C8C8C8" : "#A8D5BA"
                            }
                            strokeWidth={1.5}
                            dot={false}
                          />
                        </LineChart>
                      </ResponsiveContainer>
                    </div>
                    <p className="text-xs text-gray-500 leading-relaxed">
                      {tag.summary}
                    </p>
                  </motion.div>
                )}
              </AnimatePresence>
            )}
          </motion.div>
          );
        })}
      </AnimatePresence>
    </div>
  );
}
