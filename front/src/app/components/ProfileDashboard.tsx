import {
  ArrowLeft,
  BookOpenCheck,
  BriefcaseBusiness,
  CheckCircle2,
  CircleDashed,
  Code2,
  FlaskConical,
  Mic2,
  ShieldCheck,
} from "lucide-react";
import { Progress } from "./ui/progress";

interface ProfileDashboardProps {
  onBack: () => void;
}

const MASTERY_DISTRIBUTION = [
  { label: "完整验证", count: 5, color: "bg-[#4D8060]" },
  { label: "可独立实现", count: 7, color: "bg-[#7EAA8B]" },
  { label: "能讲清", count: 6, color: "bg-[#AAC9B2]" },
  { label: "仅接触", count: 7, color: "bg-[#D9E4D9]" },
  { label: "暂无证据", count: 7, color: "bg-[#E9D9C5]" },
] as const;

const PRIORITY_GAPS = [
  {
    name: "Go 有界并发与错误收敛",
    category: "Go 并发",
    state: "可实现，待完整验证",
    icon: Code2,
  },
  {
    name: "Kafka Rebalance 与顺序提交",
    category: "消息队列",
    state: "能讲清，缺故障证据",
    icon: ShieldCheck,
  },
  {
    name: "Agent 评测与退化检测",
    category: "AI 工程",
    state: "仅接触",
    icon: FlaskConical,
  },
  {
    name: "MySQL 索引与事务实战",
    category: "数据存储",
    state: "目标 JD 未覆盖",
    icon: CircleDashed,
  },
] as const;

const EVIDENCE_SUMMARY = [
  { label: "主动讲解", value: 12, icon: Mic2 },
  { label: "编码验证", value: 8, icon: Code2 },
  { label: "故障实验", value: 3, icon: FlaskConical },
  { label: "生产事实", value: 6, icon: BriefcaseBusiness },
] as const;

export function ProfileDashboard({ onBack }: ProfileDashboardProps) {
  return (
    <main className="flex min-h-0 flex-1 flex-col overflow-hidden bg-background">
      <header className="flex flex-shrink-0 items-center justify-between border-b border-border px-5 pb-4 pt-6">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={onBack}
            aria-label="返回"
            className="flex size-9 items-center justify-center rounded-full bg-secondary text-primary transition-colors hover:bg-accent"
          >
            <ArrowLeft size={18} />
          </button>
          <div>
            <h1 className="text-[20px] font-semibold leading-tight text-foreground">我的空间</h1>
            <p className="mt-0.5 text-[12px] text-muted-foreground">目标、掌握程度与证据总览</p>
          </div>
        </div>
        <span className="rounded-full bg-secondary px-2.5 py-1 text-[10px] text-muted-foreground">演示数据</span>
      </header>

      <div className="flex-1 overflow-y-auto px-5 pb-8 pt-5" style={{ scrollbarWidth: "none" }}>
        <section aria-labelledby="target-jd-title" className="rounded-lg bg-[#E8F0E8] p-5">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0">
              <div className="mb-2 flex items-center gap-1.5 text-[11px] font-semibold text-primary">
                <BriefcaseBusiness size={14} />
                当前目标 JD
              </div>
              <h2 id="target-jd-title" className="text-[18px] font-semibold text-foreground">
                Go / AI Agent 后端工程师
              </h2>
              <p className="mt-1 text-[12px] text-muted-foreground">高并发服务 · Agent 基建 · 分布式系统</p>
            </div>
            <div className="flex-shrink-0 text-right">
              <p className="text-[26px] font-semibold leading-none text-primary">56%</p>
              <p className="mt-1 text-[10px] text-muted-foreground">证据覆盖率</p>
            </div>
          </div>
          <Progress value={56} className="mt-5 h-2.5 bg-white/80" />
          <div className="mt-3 flex items-center justify-between text-[11px]">
            <span className="font-medium text-foreground">18 个要求已有可靠证据</span>
            <span className="text-muted-foreground">还差 14 个</span>
          </div>
        </section>

        <section aria-labelledby="mastery-title" className="mt-7">
          <div className="flex items-end justify-between gap-3">
            <div>
              <p className="text-[11px] font-medium text-primary">全部 32 个 JD 知识要求</p>
              <h2 id="mastery-title" className="mt-0.5 text-[17px] font-semibold text-foreground">掌握分布</h2>
            </div>
            <span className="text-[11px] text-muted-foreground">按最高证据等级</span>
          </div>

          <div className="mt-4 flex h-3 w-full overflow-hidden rounded-full bg-muted">
            {MASTERY_DISTRIBUTION.map((item) => (
              <span
                key={item.label}
                className={item.color}
                style={{ width: `${(item.count / 32) * 100}%` }}
                title={`${item.label} ${item.count}`}
              />
            ))}
          </div>

          <div className="mt-4 grid grid-cols-2 gap-x-5 gap-y-3 sm:grid-cols-3">
            {MASTERY_DISTRIBUTION.map((item) => (
              <div key={item.label} className="flex items-center justify-between gap-2 border-b border-border pb-2">
                <span className="flex items-center gap-2 text-[12px] text-muted-foreground">
                  <span className={`size-2 rounded-full ${item.color}`} />
                  {item.label}
                </span>
                <strong className="text-[13px] font-semibold text-foreground">{item.count}</strong>
              </div>
            ))}
          </div>
        </section>

        <section aria-labelledby="gaps-title" className="mt-7">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-[11px] font-medium text-[#A56A2A]">影响目标匹配度</p>
              <h2 id="gaps-title" className="mt-0.5 text-[17px] font-semibold text-foreground">优先补齐的缺口</h2>
            </div>
            <span className="rounded-full bg-[#F7ECDD] px-2.5 py-1 text-[10px] font-medium text-[#8B5D28]">4 项优先</span>
          </div>

          <div className="mt-3 divide-y divide-border border-y border-border">
            {PRIORITY_GAPS.map((gap) => {
              const Icon = gap.icon;
              return (
                <div key={gap.name} className="flex items-center gap-3 py-3.5">
                  <span className="flex size-9 flex-shrink-0 items-center justify-center rounded-md bg-secondary text-primary">
                    <Icon size={17} />
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="text-[13px] font-medium text-foreground">{gap.name}</p>
                    <p className="mt-0.5 text-[11px] text-muted-foreground">{gap.category} · {gap.state}</p>
                  </div>
                </div>
              );
            })}
          </div>
        </section>

        <section aria-labelledby="evidence-title" className="mt-7">
          <div className="flex items-center gap-2">
            <BookOpenCheck size={16} className="text-primary" />
            <h2 id="evidence-title" className="text-[17px] font-semibold text-foreground">掌握证据</h2>
          </div>
          <div className="mt-3 grid grid-cols-2 gap-2">
            {EVIDENCE_SUMMARY.map((item) => {
              const Icon = item.icon;
              return (
                <div key={item.label} className="rounded-lg border border-border bg-white p-3.5">
                  <div className="flex items-center justify-between">
                    <Icon size={16} className="text-primary" />
                    <span className="text-[20px] font-semibold leading-none text-foreground">{item.value}</span>
                  </div>
                  <p className="mt-3 text-[11px] text-muted-foreground">{item.label}</p>
                </div>
              );
            })}
          </div>
          <p className="mt-3 flex items-center gap-1.5 text-[11px] text-muted-foreground">
            <CheckCircle2 size={13} className="text-primary" />
            只统计有来源、时间和确认记录的证据
          </p>
        </section>
      </div>
    </main>
  );
}