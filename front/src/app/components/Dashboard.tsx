import { useState } from "react";
import { motion } from "motion/react";
import {
  X,
  Flame,
  Activity,
  Scale,
  Ruler,
  Target,
  ChevronRight,
} from "lucide-react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";

// 说明：本面板为静态演示数据，尚未接入后端。文案统一中文。
const GREEN = "#2E5E3E";

const bloodSugarData = [
  { day: "周一", value: 5.1 },
  { day: "周二", value: 4.9 },
  { day: "周三", value: 5.3 },
  { day: "周四", value: 4.7 },
  { day: "周五", value: 5.0 },
  { day: "周六", value: 4.8 },
  { day: "周日", value: 4.9 },
];

const MACROS = { protein: 62, carbs: 74, fat: 48 };
const TOTAL_CALS = 1840;
const GOAL_CALS = 2200;
const CAL_PCT = Math.round((TOTAL_CALS / GOAL_CALS) * 100);

function MacroBar({
  label,
  value,
  max,
  color,
  unit,
}: {
  label: string;
  value: number;
  max: number;
  color: string;
  unit: string;
}) {
  const pct = Math.min(100, Math.round((value / max) * 100));
  return (
    <div className="flex flex-col gap-0.5">
      <div className="flex justify-between items-center">
        <span className="text-[11px] text-muted-foreground font-medium">
          {label}
        </span>
        <span className="text-[11px] font-semibold text-foreground">
          {value}
          {unit}
        </span>
      </div>
      <div className="h-1.5 rounded-full bg-secondary overflow-hidden">
        <div
          className="h-full rounded-full transition-all duration-700"
          style={{ width: `${pct}%`, backgroundColor: color }}
        />
      </div>
    </div>
  );
}

function RingChart() {
  const r = 38;
  const stroke = 8;
  const circumference = 2 * Math.PI * r;
  const filled = (CAL_PCT / 100) * circumference;

  return (
    <div className="flex items-center gap-4">
      <div className="relative flex-shrink-0">
        <svg width="96" height="96" viewBox="0 0 96 96">
          <circle cx="48" cy="48" r={r} fill="none" stroke="#EEF3EC" strokeWidth={stroke} />
          <circle
            cx="48"
            cy="48"
            r={r}
            fill="none"
            stroke={GREEN}
            strokeWidth={stroke}
            strokeDasharray={`${filled} ${circumference}`}
            strokeLinecap="round"
            transform="rotate(-90 48 48)"
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-[15px] font-semibold text-foreground leading-none">
            {TOTAL_CALS}
          </span>
          <span className="text-[10px] text-muted-foreground leading-none mt-0.5">
            千卡
          </span>
        </div>
      </div>

      <div className="flex flex-col gap-2 flex-1">
        <MacroBar label="蛋白质" value={MACROS.protein} max={120} color={GREEN} unit="克" />
        <MacroBar label="碳水" value={MACROS.carbs} max={220} color="#6BB89A" unit="克" />
        <MacroBar label="脂肪" value={MACROS.fat} max={80} color="#A8D4BC" unit="克" />
      </div>
    </div>
  );
}

function HealthScoreRing({ score }: { score: number }) {
  const r = 52;
  const stroke = 10;
  const circumference = 2 * Math.PI * r;
  const filled = (score / 100) * circumference;
  return (
    <div className="relative flex-shrink-0">
      <svg width="124" height="124" viewBox="0 0 124 124">
        <circle cx="62" cy="62" r={r} fill="none" stroke="#EEF3EC" strokeWidth={stroke} />
        <circle
          cx="62"
          cy="62"
          r={r}
          fill="none"
          stroke={GREEN}
          strokeWidth={stroke}
          strokeDasharray={`${filled} ${circumference}`}
          strokeLinecap="round"
          transform="rotate(-90 62 62)"
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-3xl font-bold text-foreground leading-none">{score}</span>
        <span className="text-[11px] text-muted-foreground mt-0.5">健康评分</span>
      </div>
    </div>
  );
}

interface StatFieldProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  unit: string;
  onChange: (v: string) => void;
}

function StatField({ icon, label, value, unit, onChange }: StatFieldProps) {
  return (
    <div className="flex items-center gap-3 bg-secondary/60 rounded-xl px-4 py-3">
      <div className="text-primary">{icon}</div>
      <div className="flex-1">
        <p className="text-[11px] text-muted-foreground font-medium">{label}</p>
        <div className="flex items-baseline gap-1">
          <input
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className="text-[15px] font-semibold text-foreground bg-transparent border-none outline-none w-16"
          />
          <span className="text-[12px] text-muted-foreground">{unit}</span>
        </div>
      </div>
      <ChevronRight size={14} className="text-muted-foreground" />
    </div>
  );
}

const DIET_TAGS = ["乳糖不耐", "生酮饮食", "无麸质", "低钠", "高蛋白"];

// Dashboard 健康仪表盘（静态演示）：健康评分、身体数据、空腹血糖趋势、饮食偏好。
export function Dashboard({ onClose }: { onClose: () => void }) {
  const [weight, setWeight] = useState("74.2");
  const [height, setHeight] = useState("178");
  const [targetCals, setTargetCals] = useState("2200");
  const [activeTags, setActiveTags] = useState<Set<string>>(
    new Set(["乳糖不耐", "生酮饮食"])
  );

  const toggleTag = (tag: string) => {
    setActiveTags((prev) => {
      const next = new Set(prev);
      if (next.has(tag)) {
        next.delete(tag);
      } else {
        next.add(tag);
      }
      return next;
    });
  };

  return (
    <motion.div
      key="dashboard"
      initial={{ y: "100%" }}
      animate={{ y: 0 }}
      exit={{ y: "100%" }}
      transition={{ type: "spring", stiffness: 300, damping: 34 }}
      className="absolute inset-0 bg-background z-50 flex flex-col rounded-t-3xl overflow-hidden"
    >
      {/* Handle */}
      <div className="flex justify-center pt-3 pb-1 flex-shrink-0">
        <div className="w-10 h-1 rounded-full bg-border" />
      </div>

      {/* Header */}
      <div className="flex items-center justify-between px-5 py-3 flex-shrink-0">
        <div>
          <h2 className="text-[18px] font-semibold text-foreground">我的健康</h2>
          <p className="text-[12px] text-muted-foreground">最近同步 · 刚刚</p>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="w-8 h-8 rounded-full bg-secondary flex items-center justify-center hover:bg-accent transition-colors"
        >
          <X size={15} className="text-muted-foreground" />
        </button>
      </div>

      <div
        className="flex-1 overflow-y-auto px-5 pb-8 space-y-5"
        style={{ scrollbarWidth: "none" }}
      >
        {/* Health Score */}
        <div className="bg-card rounded-2xl p-5 border border-border flex items-center gap-5">
          <HealthScoreRing score={82} />
          <div className="flex flex-col gap-2">
            <div>
              <p className="text-[12px] text-muted-foreground">状态</p>
              <p className="text-[14px] font-semibold text-primary">状态良好</p>
            </div>
            <div>
              <p className="text-[12px] text-muted-foreground">连续记录</p>
              <p className="text-[14px] font-semibold text-foreground">已坚持 12 天</p>
            </div>
            <div>
              <p className="text-[12px] text-muted-foreground">BMI</p>
              <p className="text-[14px] font-semibold text-foreground">23.4 · 正常</p>
            </div>
          </div>
        </div>

        {/* Intake */}
        <div className="bg-card rounded-2xl p-4 border border-border shadow-sm">
          <div className="flex items-center gap-1.5 mb-3">
            <Flame size={13} className="text-primary" />
            <span className="text-[12px] font-semibold text-primary tracking-wide">
              今日摄入
            </span>
          </div>
          <RingChart />
          <p className="text-[11px] text-muted-foreground mt-3 leading-relaxed">
            已达成每日目标 {GOAL_CALS} 千卡的 {CAL_PCT}% · 还剩 {GOAL_CALS - TOTAL_CALS} 千卡
          </p>
        </div>

        {/* Stats */}
        <div>
          <p className="text-[12px] font-semibold text-muted-foreground tracking-wider mb-2.5">
            身体数据
          </p>
          <div className="space-y-2">
            <StatField icon={<Scale size={16} />} label="体重" value={weight} unit="千克" onChange={setWeight} />
            <StatField icon={<Ruler size={16} />} label="身高" value={height} unit="厘米" onChange={setHeight} />
            <StatField icon={<Target size={16} />} label="每日热量目标" value={targetCals} unit="千卡" onChange={setTargetCals} />
          </div>
        </div>

        {/* Blood Sugar Chart */}
        <div className="bg-card rounded-2xl p-5 border border-border">
          <div className="flex items-center gap-1.5 mb-4">
            <Activity size={14} className="text-primary" />
            <p className="text-[13px] font-semibold text-foreground">空腹血糖</p>
            <span className="ml-auto text-[11px] text-muted-foreground">mmol/L · 本周</span>
          </div>
          <ResponsiveContainer width="100%" height={120}>
            <LineChart data={bloodSugarData} margin={{ top: 4, right: 4, left: -28, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(46,94,62,0.08)" vertical={false} />
              <XAxis dataKey="day" tick={{ fontSize: 10, fill: "#7A8A81" }} axisLine={false} tickLine={false} />
              <YAxis domain={[4.4, 5.6]} tick={{ fontSize: 10, fill: "#7A8A81" }} axisLine={false} tickLine={false} />
              <Tooltip
                contentStyle={{
                  background: "#fff",
                  border: "1px solid rgba(46,94,62,0.15)",
                  borderRadius: 10,
                  fontSize: 12,
                  padding: "4px 10px",
                }}
                itemStyle={{ color: GREEN }}
                labelStyle={{ color: "#1C2320" }}
              />
              <Line
                type="monotone"
                dataKey="value"
                name="血糖"
                stroke={GREEN}
                strokeWidth={2}
                dot={{ r: 3, fill: GREEN, strokeWidth: 0 }}
                activeDot={{ r: 5, fill: GREEN }}
              />
            </LineChart>
          </ResponsiveContainer>
          <div className="flex items-center justify-between mt-2">
            <span className="text-[11px] text-muted-foreground">均值：4.96 mmol/L</span>
            <span className="text-[11px] text-primary font-medium">正常范围</span>
          </div>
        </div>

        {/* Preferences */}
        <div>
          <p className="text-[12px] font-semibold text-muted-foreground tracking-wider mb-2.5">
            饮食偏好
          </p>
          <div className="flex flex-wrap gap-2">
            {DIET_TAGS.map((tag) => {
              const active = activeTags.has(tag);
              return (
                <button
                  key={tag}
                  type="button"
                  onClick={() => toggleTag(tag)}
                  className={`px-3.5 py-1.5 rounded-full text-[12px] font-medium transition-all duration-200 ${
                    active
                      ? "bg-primary text-primary-foreground"
                      : "bg-secondary text-muted-foreground border border-border"
                  }`}
                >
                  {tag}
                </button>
              );
            })}
          </div>
        </div>
      </div>
    </motion.div>
  );
}
