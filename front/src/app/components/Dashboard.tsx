import { motion } from "motion/react";
import {
  BookOpenCheck,
  BrainCircuit,
  CalendarCheck,
  CheckCircle2,
  MessageSquareText,
  X,
} from "lucide-react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

const GREEN = "#2E5E3E";

const outputData = [
  { day: "周一", attempts: 1 },
  { day: "周二", attempts: 2 },
  { day: "周三", attempts: 0 },
  { day: "周四", attempts: 1 },
  { day: "周五", attempts: 3 },
  { day: "周六", attempts: 1 },
  { day: "周日", attempts: 2 },
];

const evidenceStages = [
  { label: "已接触", value: 12, max: 12, color: "#A8D4BC" },
  { label: "能讲清", value: 6, max: 12, color: "#6BB89A" },
  { label: "已验证", value: 2, max: 12, color: GREEN },
];

const topics = [
  {
    title: "Kafka 消息积压",
    detail: "今天 · 知识点回顾",
    status: "待回顾",
    icon: CalendarCheck,
  },
  {
    title: "Go GC 三色标记",
    detail: "下一步 · 费曼输出",
    status: "待输出",
    icon: MessageSquareText,
  },
  {
    title: "分布式事务",
    detail: "已完成 · 故障实验",
    status: "已验证",
    icon: CheckCircle2,
  },
];

function EvidenceProgress() {
  return (
    <div className="space-y-3">
      {evidenceStages.map((stage) => (
        <div key={stage.label}>
          <div className="mb-1.5 flex items-center justify-between text-[12px]">
            <span className="font-medium text-foreground">{stage.label}</span>
            <span className="text-muted-foreground">{stage.value} 个知识点</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-secondary">
            <div
              className="h-full rounded-full"
              style={{
                width: `${Math.round((stage.value / stage.max) * 100)}%`,
                backgroundColor: stage.color,
              }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

export function Dashboard({ onClose }: { onClose: () => void }) {
  return (
    <motion.div
      key="dashboard"
      initial={{ y: "100%" }}
      animate={{ y: 0 }}
      exit={{ y: "100%" }}
      transition={{ type: "spring", stiffness: 300, damping: 34 }}
      className="absolute inset-0 z-50 flex flex-col overflow-hidden rounded-t-3xl bg-background"
    >
      <div className="flex flex-shrink-0 justify-center pb-1 pt-3">
        <div className="h-1 w-10 rounded-full bg-border" />
      </div>

      <div className="flex flex-shrink-0 items-center justify-between px-5 py-3">
        <div>
          <h2 className="text-[18px] font-semibold text-foreground">学习看板</h2>
          <p className="text-[12px] text-muted-foreground">基于主动输出与验证证据</p>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="关闭学习看板"
          className="flex size-8 items-center justify-center rounded-full bg-secondary transition-colors hover:bg-accent"
        >
          <X size={15} className="text-muted-foreground" />
        </button>
      </div>

      <div
        className="flex-1 space-y-5 overflow-y-auto px-5 pb-8"
        style={{ scrollbarWidth: "none" }}
      >
        <section className="rounded-2xl border border-border bg-card p-5">
          <div className="mb-4 flex items-start justify-between gap-4">
            <div>
              <div className="mb-1 flex items-center gap-1.5 text-primary">
                <BrainCircuit size={15} />
                <span className="text-[12px] font-semibold">掌握证据</span>
              </div>
              <p className="text-[11px] leading-relaxed text-muted-foreground">
                导入不等于掌握，只有主动输出和验证才会推进阶段。
              </p>
            </div>
            <span className="flex-shrink-0 rounded-full bg-secondary px-2.5 py-1 text-[10px] text-muted-foreground">
              演示数据
            </span>
          </div>
          <EvidenceProgress />
        </section>

        <section className="rounded-2xl border border-border bg-card p-5">
          <div className="mb-4 flex items-center gap-1.5">
            <MessageSquareText size={14} className="text-primary" />
            <p className="text-[13px] font-semibold text-foreground">本周主动输出</p>
            <span className="ml-auto text-[11px] text-muted-foreground">共 10 次</span>
          </div>
          <ResponsiveContainer width="100%" height={130}>
            <LineChart data={outputData} margin={{ top: 4, right: 4, left: -34, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(46,94,62,0.08)" vertical={false} />
              <XAxis dataKey="day" tick={{ fontSize: 10, fill: "#7A8A81" }} axisLine={false} tickLine={false} />
              <YAxis domain={[0, 3]} ticks={[0, 1, 2, 3]} tick={{ fontSize: 10, fill: "#7A8A81" }} axisLine={false} tickLine={false} />
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
                dataKey="attempts"
                name="主动输出"
                stroke={GREEN}
                strokeWidth={2}
                dot={{ r: 3, fill: GREEN, strokeWidth: 0 }}
                activeDot={{ r: 5, fill: GREEN }}
              />
            </LineChart>
          </ResponsiveContainer>
        </section>

        <section>
          <div className="mb-2.5 flex items-center gap-1.5">
            <BookOpenCheck size={14} className="text-primary" />
            <p className="text-[12px] font-semibold tracking-wide text-muted-foreground">当前主题</p>
          </div>
          <div className="space-y-2">
            {topics.map((topic) => {
              const Icon = topic.icon;
              return (
                <div
                  key={topic.title}
                  className="flex items-center gap-3 rounded-xl border border-border bg-card px-4 py-3"
                >
                  <div className="flex size-9 flex-shrink-0 items-center justify-center rounded-lg bg-secondary text-primary">
                    <Icon size={16} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[13px] font-semibold text-foreground">{topic.title}</p>
                    <p className="mt-0.5 text-[11px] text-muted-foreground">{topic.detail}</p>
                  </div>
                  <span className="flex-shrink-0 rounded-full bg-secondary px-2.5 py-1 text-[10px] font-medium text-primary">
                    {topic.status}
                  </span>
                </div>
              );
            })}
          </div>
        </section>
      </div>
    </motion.div>
  );
}