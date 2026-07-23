import { useRef, useState } from "react";
import { motion } from "motion/react";
import {
  Archive,
  BookOpenText,
  BriefcaseBusiness,
  Check,
  CheckCircle2,
  Clock3,
  FileText,
  FlaskConical,
  Link2,
  Pencil,
  Plus,
  Search,
  ShieldCheck,
  Sparkles,
  Upload,
  X,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { Input } from "./ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs";
import { Textarea } from "./ui/textarea";
import { ProfileButton } from "./ProfileButton";

type SourceType = "PDF" | "Markdown" | "网页" | "手动笔记";
type SourceStatus = "待识别" | "待确认" | "已整理" | "仅归档";
type SourceOrigin = "用户笔记" | "AI 整理" | "外部资料" | "来源待确认";
type SourceKind = "学习笔记" | "学习 Todo" | "技术资料" | "目标 JD" | "项目事实" | "面试复盘" | "其他";
type SourcePurpose = "供我学习" | "供 AI 检索" | "生成计划" | "事实参考" | "仅归档";
type CandidateKind = "知识点候选" | "目标要求候选" | "事实候选";
type MasteryLevel = "暂无证据" | "仅接触" | "能讲清" | "可独立实现" | "完整验证";
type EvidenceScope = "学习练习" | "个人 Demo" | "模拟场景" | "生产实践";

interface KnowledgeSource {
  id: string;
  title: string;
  type: SourceType;
  meta: string;
  topics: string[];
  summary: string;
  updatedAt: string;
  status: SourceStatus;
  origin: SourceOrigin;
  kind: SourceKind;
  purposes: SourcePurpose[];
}

interface ContentCandidate {
  id: string;
  kind: CandidateKind;
  title: string;
  source: string;
  excerpt: string;
  suggestion: string;
}

interface MasteryItem {
  id: string;
  name: string;
  category: string;
  level: MasteryLevel;
  evidence: string;
  evidenceScope?: EvidenceScope;
  nextReview: string;
}

interface SourceDraft {
  title: string;
  topics: string;
  summary: string;
  origin: SourceOrigin;
  kind: SourceKind;
  purposes: SourcePurpose[];
}

const MASTERY_LEVELS: MasteryLevel[] = ["暂无证据", "仅接触", "能讲清", "可独立实现", "完整验证"];
const EVIDENCE_SCOPES: EvidenceScope[] = ["学习练习", "个人 Demo", "模拟场景", "生产实践"];
const SOURCE_ORIGINS: SourceOrigin[] = ["用户笔记", "AI 整理", "外部资料", "来源待确认"];
const SOURCE_KINDS: SourceKind[] = ["学习笔记", "学习 Todo", "技术资料", "目标 JD", "项目事实", "面试复盘", "其他"];
const SOURCE_PURPOSES: SourcePurpose[] = ["供我学习", "供 AI 检索", "生成计划", "事实参考", "仅归档"];

const INITIAL_SOURCES: KnowledgeSource[] = [
  {
    id: "go-concurrency",
    title: "Go 并发核心笔记",
    type: "Markdown",
    meta: "18.4 KB",
    topics: ["GMP", "Channel", "Worker Pool"],
    summary: "整理 goroutine 调度、channel 语义与有界并发的关键约束。",
    updatedAt: "今天 09:40",
    status: "待确认",
    origin: "用户笔记",
    kind: "学习笔记",
    purposes: ["供我学习", "供 AI 检索"],
  },
  {
    id: "kafka-interview",
    title: "Kafka 面试问题集",
    type: "PDF",
    meta: "1.8 MB",
    topics: ["可靠性", "消息积压", "消费幂等"],
    summary: "覆盖生产、存储、消费和故障恢复四条面试追问链路。",
    updatedAt: "昨天 21:16",
    status: "已整理",
    origin: "外部资料",
    kind: "技术资料",
    purposes: ["供我学习", "供 AI 检索"],
  },
  {
    id: "project-stories",
    title: "项目难点故事库",
    type: "手动笔记",
    meta: "6 个故事",
    topics: ["分布式事务", "系统设计"],
    summary: "按背景、冲突、方案、取舍和结果沉淀可验证的项目表达素材。",
    updatedAt: "7月20日",
    status: "待确认",
    origin: "用户笔记",
    kind: "项目事实",
    purposes: ["供我学习", "事实参考"],
  },
  {
    id: "target-jd",
    title: "Go / AI Agent 后端岗位 JD",
    type: "网页",
    meta: "网页快照",
    topics: ["岗位要求", "Agent 基建"],
    summary: "保存岗位职责与任职要求，待确认哪些要求需要关联到当前目标。",
    updatedAt: "7月19日",
    status: "待确认",
    origin: "外部资料",
    kind: "目标 JD",
    purposes: ["生成计划", "供 AI 检索"],
  },
  {
    id: "previous-study-todo",
    title: "上一阶段学习 Todo",
    type: "Markdown",
    meta: "12 个任务",
    topics: ["Go 并发", "Kafka", "待复习"],
    summary: "用户以前维护的学习任务，部分任务可迁入计划，技术描述可选择供 AI 检索。",
    updatedAt: "7月18日",
    status: "待确认",
    origin: "用户笔记",
    kind: "学习 Todo",
    purposes: ["生成计划"],
  },
  {
    id: "ai-study-notes",
    title: "AI 整理的 Agent 评测笔记",
    type: "Markdown",
    meta: "9.2 KB",
    topics: ["离线评测", "回归测试", "待核实"],
    summary: "AI 根据对话整理的笔记，允许用于学习或检索，但引用前仍需核实来源与结论。",
    updatedAt: "7月17日",
    status: "待确认",
    origin: "AI 整理",
    kind: "学习笔记",
    purposes: ["供我学习", "供 AI 检索"],
  },
];

const INITIAL_CANDIDATES: ContentCandidate[] = [
  {
    id: "scheduler-queues",
    kind: "知识点候选",
    title: "Go 调度器的本地队列与全局队列",
    source: "Go 并发核心笔记",
    excerpt: "P 优先从本地运行队列获取 G，本地为空时再按顺序尝试其他来源……",
    suggestion: "建议新建知识点；它是理解 work stealing 和调度公平性的前置概念。",
  },
  {
    id: "agent-evaluation",
    kind: "目标要求候选",
    title: "具备 Agent 评测与效果迭代经验",
    source: "Go / AI Agent 后端岗位 JD",
    excerpt: "负责 Agent 服务质量评估、Prompt 版本管理与线上效果持续优化。",
    suggestion: "建议关联当前目标 JD，再由差距分析决定是否进入学习计划。",
  },
  {
    id: "outbox-fact",
    kind: "事实候选",
    title: "在借贷链路中参与 Outbox 方案落地",
    source: "项目难点故事库",
    excerpt: "业务写入与事件记录在同一事务提交，后台任务负责补偿发送……",
    suggestion: "仅保存为待核实事实；不能自动成为生产证据或提高掌握等级。",
  },
];

const INITIAL_MASTERY: MasteryItem[] = [
  {
    id: "kafka-backlog",
    name: "Kafka 消息积压排查",
    category: "消息队列",
    level: "能讲清",
    evidence: "已无提示复述完整排查链路，异常恢复细节仍需补充。",
    evidenceScope: "学习练习",
    nextReview: "2026-07-24",
  },
  {
    id: "go-gc",
    name: "Go GC 三色标记",
    category: "Go Runtime",
    level: "仅接触",
    evidence: "能说出颜色定义，但写屏障触发时机容易混淆。",
    evidenceScope: "学习练习",
    nextReview: "2026-07-23",
  },
  {
    id: "worker-pool",
    name: "有界 Worker Pool",
    category: "Go 并发",
    level: "完整验证",
    evidence: "已独立实现取消、错误收敛和有界并发测试。",
    evidenceScope: "个人 Demo",
    nextReview: "2026-07-29",
  },
  {
    id: "mysql-mvcc",
    name: "MySQL MVCC",
    category: "数据库",
    level: "暂无证据",
    evidence: "已决定追踪这个知识点，尚未产生接触或主动输出证据。",
    nextReview: "2026-07-22",
  },
];

const LEVEL_STYLES: Record<MasteryLevel, string> = {
  暂无证据: "bg-[#F1EEEA] text-[#756B61]",
  仅接触: "bg-[#FFF4D9] text-[#946B16]",
  能讲清: "bg-[#E8F0F7] text-[#456986]",
  可独立实现: "bg-[#E4F1E8] text-[#2E6941]",
  完整验证: "bg-[#DCE9DF] text-[#245438]",
};

const STATUS_STYLES: Record<SourceStatus, string> = {
  待识别: "bg-[#F1EEEA] text-[#756B61]",
  待确认: "bg-[#FFF4D9] text-[#946B16]",
  已整理: "bg-[#E4F1E8] text-[#2E6941]",
  仅归档: "bg-[#E8F0F7] text-[#456986]",
};

const CANDIDATE_STYLES: Record<CandidateKind, string> = {
  知识点候选: "bg-[#E8F0F7] text-[#456986]",
  目标要求候选: "bg-[#F7ECDD] text-[#8B5D28]",
  事实候选: "bg-[#E4F1E8] text-[#2E6941]",
};

function sourceTypeFor(file: File): SourceType {
  const extension = file.name.split(".").pop()?.toLowerCase();
  if (extension === "pdf") return "PDF";
  return "Markdown";
}

interface KnowledgeBasePageProps {
  onOpenProfile: () => void;
}

export function KnowledgeBasePage({ onOpenProfile }: KnowledgeBasePageProps) {
  const [sources, setSources] = useState(INITIAL_SOURCES);
  const [candidates, setCandidates] = useState(INITIAL_CANDIDATES);
  const [masteryItems, setMasteryItems] = useState(INITIAL_MASTERY);
  const [query, setQuery] = useState("");
  const [sourceEditorOpen, setSourceEditorOpen] = useState(false);
  const [editingSourceID, setEditingSourceID] = useState<string | null>(null);
  const [sourceDraft, setSourceDraft] = useState<SourceDraft>({
    title: "",
    topics: "",
    summary: "",
    origin: "用户笔记",
    kind: "学习笔记",
    purposes: ["仅归档"],
  });
  const [masteryEditorOpen, setMasteryEditorOpen] = useState(false);
  const [editingMasteryID, setEditingMasteryID] = useState<string | null>(null);
  const [masteryLevel, setMasteryLevel] = useState<MasteryLevel>("仅接触");
  const [masteryEvidence, setMasteryEvidence] = useState("");
  const [evidenceScope, setEvidenceScope] = useState<EvidenceScope>("学习练习");
  const [productionConfirmed, setProductionConfirmed] = useState(false);
  const [nextReview, setNextReview] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);

  const filteredSources = sources.filter((source) => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [source.title, source.summary, ...source.topics]
      .join(" ")
      .toLowerCase()
      .includes(normalized);
  });

  const openNewNote = () => {
    setEditingSourceID(null);
    setSourceDraft({ title: "", topics: "", summary: "", origin: "用户笔记", kind: "学习笔记", purposes: ["仅归档"] });
    setSourceEditorOpen(true);
  };

  const openSourceEditor = (source: KnowledgeSource) => {
    setEditingSourceID(source.id);
    setSourceDraft({
      title: source.title,
      topics: source.topics.join("、"),
      summary: source.summary,
      origin: source.origin,
      kind: source.kind,
      purposes: source.purposes,
    });
    setSourceEditorOpen(true);
  };

  const saveSource = () => {
    const title = sourceDraft.title.trim();
    if (!title) return;
    const topics = sourceDraft.topics
      .split(/[、,，]/)
      .map((topic) => topic.trim())
      .filter(Boolean);

    if (editingSourceID) {
      setSources((current) =>
        current.map((source) =>
          source.id === editingSourceID
            ? {
                ...source,
                title,
                topics,
                summary: sourceDraft.summary.trim(),
                origin: sourceDraft.origin,
                kind: sourceDraft.kind,
                purposes: sourceDraft.purposes,
                updatedAt: "刚刚",
              }
            : source,
        ),
      );
    } else {
      setSources((current) => [
        {
          id: crypto.randomUUID(),
          title,
          type: "手动笔记",
          meta: "新建笔记",
          topics,
          summary: sourceDraft.summary.trim(),
          updatedAt: "刚刚",
          status: "仅归档",
          origin: sourceDraft.origin,
          kind: sourceDraft.kind,
          purposes: sourceDraft.purposes,
        },
        ...current,
      ]);
    }
    setSourceEditorOpen(false);
  };

  const handleUpload = (file: File) => {
    const title = file.name.replace(/\.[^.]+$/, "");
    setSources((current) => [
      {
        id: crypto.randomUUID(),
        title,
        type: sourceTypeFor(file),
        meta: `${Math.max(file.size / 1024, 0.1).toFixed(1)} KB`,
        topics: ["尚未分类"],
        summary: "资料已上传，等待识别用途与提取候选内容；不会改变掌握状态。",
        updatedAt: "刚刚",
        status: "待识别",
        origin: "来源待确认",
        kind: "其他",
        purposes: [],
      },
      ...current,
    ]);
  };

  const openMasteryEditor = (item: MasteryItem) => {
    setEditingMasteryID(item.id);
    setMasteryLevel(item.level);
    setMasteryEvidence(item.evidence);
    setEvidenceScope(item.evidenceScope ?? "学习练习");
    setProductionConfirmed(item.evidenceScope === "生产实践");
    setNextReview(item.nextReview);
    setMasteryEditorOpen(true);
  };

  const saveMastery = () => {
    if (!editingMasteryID) return;
    if (masteryLevel !== "暂无证据" && !masteryEvidence.trim()) return;
    if (masteryLevel !== "暂无证据" && evidenceScope === "生产实践" && !productionConfirmed) return;
    setMasteryItems((current) =>
      current.map((item) =>
        item.id === editingMasteryID
          ? {
              ...item,
              level: masteryLevel,
              evidence: masteryEvidence.trim(),
              evidenceScope: masteryLevel === "暂无证据" ? undefined : evidenceScope,
              nextReview,
            }
          : item,
      ),
    );
    setMasteryEditorOpen(false);
  };

  const masteredCount = masteryItems.filter(
    (item) => item.level === "能讲清" || item.level === "可独立实现" || item.level === "完整验证",
  ).length;

  const verifiedCount = masteryItems.filter((item) => item.level === "完整验证").length;

  const resolveCandidate = (candidate: ContentCandidate, resolution: "confirm" | "archive" | "reject") => {
    if (resolution === "confirm" && candidate.kind === "知识点候选") {
      setMasteryItems((current) => {
        if (current.some((item) => item.name === candidate.title)) return current;
        return [
          {
            id: candidate.id,
            name: candidate.title,
            category: "待分类",
            level: "暂无证据",
            evidence: "已确认纳入知识库，尚未产生掌握证据。",
            nextReview: "待安排",
          },
          ...current,
        ];
      });
    }
    setCandidates((current) => current.filter((item) => item.id !== candidate.id));
  };

  const candidateActionLabel = (kind: CandidateKind) => {
    if (kind === "知识点候选") return "纳入知识库";
    if (kind === "目标要求候选") return "关联目标 JD";
    return "保存待核实事实";
  };

  const toggleSourcePurpose = (purpose: SourcePurpose) => {
    setSourceDraft((current) => {
      const selected = current.purposes.includes(purpose);
      const purposes = selected
        ? current.purposes.filter((item) => item !== purpose)
        : [...current.purposes.filter((item) => item !== "仅归档"), purpose];

      if (purpose === "仅归档" && !selected) {
        return { ...current, purposes: ["仅归档"] };
      }
      return { ...current, purposes };
    });
  };

  return (
    <motion.main
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className="flex min-h-0 flex-1 flex-col overflow-hidden bg-background pb-16"
    >
      <header className="flex flex-shrink-0 items-start justify-between gap-4 px-5 pb-4 pt-8">
        <div>
          <h1 className="text-[20px] font-semibold text-foreground">知识库</h1>
          <p className="mt-0.5 text-[12px] text-muted-foreground">资料先确认用途，掌握只看真实证据</p>
        </div>
        <div className="flex items-center gap-2">
          <span className="rounded-full bg-secondary px-2.5 py-1 text-[10px] text-muted-foreground">演示数据</span>
          <ProfileButton onClick={onOpenProfile} />
        </div>
      </header>

      <Tabs defaultValue="sources" className="min-h-0 flex-1 gap-0">
        <div className="px-5">
          <TabsList className="grid h-10 w-full grid-cols-3 rounded-lg bg-secondary p-1">
            <TabsTrigger value="sources" className="rounded-md text-[12px]">资料</TabsTrigger>
            <TabsTrigger value="candidates" className="rounded-md text-[12px]">
              待确认
              {candidates.length > 0 && <span className="ml-1 text-[9px] text-[#946B16]">{candidates.length}</span>}
            </TabsTrigger>
            <TabsTrigger value="mastery" className="rounded-md text-[12px]">掌握情况</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="sources" className="min-h-0 overflow-y-auto px-5 pb-24 pt-5">
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="flex h-11 items-center justify-center gap-2 rounded-lg bg-primary text-[12px] font-semibold text-white transition-colors hover:bg-primary/90"
            >
              <Upload size={15} />
              上传资料
            </button>
            <button
              type="button"
              onClick={openNewNote}
              className="flex h-11 items-center justify-center gap-2 rounded-lg border border-border bg-white text-[12px] font-semibold text-foreground transition-colors hover:bg-secondary"
            >
              <Plus size={15} />
              新建笔记
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".pdf,.md,.markdown,.txt,.doc,.docx"
              className="hidden"
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) handleUpload(file);
                event.target.value = "";
              }}
            />
          </div>

          <div className="relative mt-4">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索资料或内容线索"
              className="h-10 bg-white pl-9 text-[12px]"
            />
          </div>

          <div className="mb-3 mt-5 flex items-center justify-between">
            <h2 className="text-[14px] font-semibold text-foreground">全部资料</h2>
            <span className="text-[11px] text-muted-foreground">{filteredSources.length} 份</span>
          </div>

          <div className="space-y-2.5">
            {filteredSources.map((source) => (
              <article key={source.id} className="rounded-lg border border-border bg-white p-3.5">
                <div className="flex items-start gap-3">
                  <span className="flex size-9 flex-shrink-0 items-center justify-center rounded-lg bg-secondary text-primary">
                    {source.type === "网页" ? <Link2 size={16} /> : source.type === "手动笔记" ? <BookOpenText size={16} /> : <FileText size={16} />}
                  </span>
                  <div className="min-w-0 flex-1">
                    <h3 className="truncate text-[13px] font-semibold text-foreground">{source.title}</h3>
                    <p className="mt-0.5 text-[10px] text-muted-foreground">{source.type} · {source.meta} · {source.updatedAt}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => openSourceEditor(source)}
                    aria-label={`编辑资料：${source.title}`}
                    title="编辑资料"
                    className="flex size-8 flex-shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-secondary hover:text-primary"
                  >
                    <Pencil size={14} />
                  </button>
                </div>
                <div className="mt-3 flex items-center justify-between gap-3">
                  <span className={`rounded-md px-2 py-1 text-[10px] font-medium ${STATUS_STYLES[source.status]}`}>{source.status}</span>
                  <span className="text-[10px] text-muted-foreground">内容线索，不计入掌握</span>
                </div>
                <p className="mt-3 text-[11px] leading-5 text-muted-foreground">{source.summary}</p>
                <div className="mt-3 flex flex-wrap items-center gap-1.5 border-t border-border pt-3">
                  <span className="rounded-md border border-border px-2 py-1 text-[10px] text-muted-foreground">来源：{source.origin}</span>
                  <span className="rounded-md border border-border px-2 py-1 text-[10px] text-muted-foreground">类别：{source.kind}</span>
                  {source.purposes.length > 0 ? source.purposes.map((purpose) => (
                    <span key={purpose} className="rounded-md bg-[#E8F0E8] px-2 py-1 text-[10px] text-[#2E6941]">{purpose}</span>
                  )) : (
                    <span className="rounded-md bg-[#FFF4D9] px-2 py-1 text-[10px] text-[#946B16]">用途待确认</span>
                  )}
                </div>
                <div className="mt-3 flex flex-wrap gap-1.5">
                  {source.topics.map((topic) => (
                    <span key={topic} className="rounded-md bg-secondary px-2 py-1 text-[10px] text-primary">{topic}</span>
                  ))}
                </div>
              </article>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="candidates" className="min-h-0 overflow-y-auto px-5 pb-24 pt-5">
          <section className="border-l-2 border-[#C7944A] bg-[#FFF9EC] px-3.5 py-3" aria-label="候选内容说明">
            <div className="flex items-start gap-2.5">
              <Sparkles size={15} className="mt-0.5 flex-shrink-0 text-[#946B16]" />
              <div>
                <h2 className="text-[12px] font-semibold text-foreground">AI 只提出候选，不替你决定</h2>
                <p className="mt-1 text-[10px] leading-4 text-muted-foreground">确认用途后才会进入知识库、目标 JD 或事实边界；也可以只归档。</p>
              </div>
            </div>
          </section>

          <div className="mb-3 mt-5 flex items-center justify-between">
            <h2 className="text-[14px] font-semibold text-foreground">待处理内容</h2>
            <span className="text-[11px] text-muted-foreground">{candidates.length} 项</span>
          </div>

          {candidates.length === 0 ? (
            <div className="border-y border-border py-10 text-center">
              <Check size={22} className="mx-auto text-primary" />
              <p className="mt-2 text-[13px] font-medium text-foreground">候选内容已处理完</p>
              <p className="mt-1 text-[10px] text-muted-foreground">新资料解析后会在这里等待确认用途。</p>
            </div>
          ) : (
            <div className="space-y-2.5">
              {candidates.map((candidate) => (
                <article key={candidate.id} className="rounded-lg border border-border bg-white p-3.5">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <span className={`inline-flex rounded-md px-2 py-1 text-[10px] font-medium ${CANDIDATE_STYLES[candidate.kind]}`}>{candidate.kind}</span>
                      <h3 className="mt-2 text-[13px] font-semibold leading-5 text-foreground">{candidate.title}</h3>
                    </div>
                    <button
                      type="button"
                      onClick={() => resolveCandidate(candidate, "reject")}
                      aria-label={`忽略候选：${candidate.title}`}
                      title="忽略"
                      className="flex size-8 flex-shrink-0 items-center justify-center rounded-full text-muted-foreground hover:bg-secondary hover:text-foreground"
                    >
                      <X size={14} />
                    </button>
                  </div>

                  <div className="mt-3 border-l-2 border-border pl-3">
                    <p className="text-[10px] font-medium text-muted-foreground">来自：{candidate.source}</p>
                    <p className="mt-1 text-[11px] leading-5 text-foreground">“{candidate.excerpt}”</p>
                  </div>

                  <p className="mt-3 text-[10px] leading-4 text-muted-foreground">{candidate.suggestion}</p>

                  <div className="mt-3 flex items-center justify-end gap-2 border-t border-border pt-3">
                    <button
                      type="button"
                      onClick={() => resolveCandidate(candidate, "archive")}
                      className="flex h-8 items-center gap-1.5 rounded-md px-2.5 text-[11px] font-medium text-muted-foreground hover:bg-secondary"
                    >
                      <Archive size={13} />
                      仅归档
                    </button>
                    <button
                      type="button"
                      onClick={() => resolveCandidate(candidate, "confirm")}
                      className="flex h-8 items-center gap-1.5 rounded-md bg-primary px-3 text-[11px] font-semibold text-white hover:bg-primary/90"
                    >
                      <Check size={13} />
                      {candidateActionLabel(candidate.kind)}
                    </button>
                  </div>
                </article>
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="mastery" className="min-h-0 overflow-y-auto px-5 pb-24 pt-5">
          <section className="border-b border-border pb-5" aria-label="掌握情况摘要">
            <div className="flex items-end justify-between">
              <div>
                <p className="text-[11px] font-medium text-primary">已形成输出证据</p>
                <p className="mt-1 text-[26px] font-semibold text-foreground">{masteredCount}/{masteryItems.length}</p>
              </div>
              <div className="text-right">
                <p className="text-[13px] font-semibold text-foreground">{Math.round((masteredCount / masteryItems.length) * 100)}%</p>
                <p className="text-[10px] text-muted-foreground">达到“能讲清”以上</p>
              </div>
            </div>
            <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-secondary">
              <div className="h-full rounded-full bg-primary" style={{ width: `${(masteredCount / masteryItems.length) * 100}%` }} />
            </div>
            <div className="mt-3 flex items-center justify-between text-[10px] text-muted-foreground">
              <span>完整验证 {verifiedCount} 项</span>
              <span>生产实践是证据来源，不是等级</span>
            </div>
          </section>

          <div className="mb-3 mt-5 flex items-center justify-between">
            <div>
              <h2 className="text-[14px] font-semibold text-foreground">知识点</h2>
              <p className="mt-0.5 text-[10px] text-muted-foreground">掌握等级应由最近一次输出证据校准</p>
            </div>
          </div>

          <div className="space-y-2.5">
            {masteryItems.map((item) => (
              <article key={item.id} className="rounded-lg border border-border bg-white p-3.5">
                <div className="flex items-start gap-3">
                  <span className="flex size-9 flex-shrink-0 items-center justify-center rounded-lg bg-secondary text-primary">
                    {item.level === "完整验证" ? <ShieldCheck size={16} /> : <CheckCircle2 size={16} />}
                  </span>
                  <div className="min-w-0 flex-1">
                    <h3 className="text-[13px] font-semibold text-foreground">{item.name}</h3>
                    <p className="mt-0.5 text-[10px] text-muted-foreground">{item.category}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => openMasteryEditor(item)}
                    aria-label={`编辑掌握情况：${item.name}`}
                    title="编辑掌握情况"
                    className="flex size-8 flex-shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-secondary hover:text-primary"
                  >
                    <Pencil size={14} />
                  </button>
                </div>
                <div className="mt-3 flex items-center justify-between gap-3">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <span className={`rounded-md px-2 py-1 text-[10px] font-medium ${LEVEL_STYLES[item.level]}`}>{item.level}</span>
                    {item.evidenceScope && (
                      <span className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[10px] text-muted-foreground">
                        {item.evidenceScope === "生产实践" ? <BriefcaseBusiness size={10} /> : <FlaskConical size={10} />}
                        {item.evidenceScope}
                      </span>
                    )}
                  </div>
                  <span className="flex items-center gap-1 text-[10px] text-muted-foreground"><Clock3 size={11} />复习 {item.nextReview}</span>
                </div>
                <p className="mt-3 border-t border-border pt-3 text-[11px] leading-5 text-muted-foreground">{item.evidence}</p>
              </article>
            ))}
          </div>
        </TabsContent>
      </Tabs>

      <Dialog open={sourceEditorOpen} onOpenChange={setSourceEditorOpen}>
        <DialogContent className="z-[70] max-h-[calc(100vh-2rem)] max-w-[360px] gap-5 overflow-y-auto rounded-2xl border-border bg-white p-5">
          <DialogHeader className="text-left">
            <DialogTitle>{editingSourceID ? "编辑资料" : "新建笔记"}</DialogTitle>
            <DialogDescription>维护资料本身；内容线索不等于正式知识点。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">名称</span>
              <Input value={sourceDraft.title} onChange={(event) => setSourceDraft((current) => ({ ...current, title: event.target.value }))} placeholder="资料或笔记名称" autoFocus />
            </label>
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">内容线索</span>
              <Input value={sourceDraft.topics} onChange={(event) => setSourceDraft((current) => ({ ...current, topics: event.target.value }))} placeholder="仅用于整理和搜索，使用逗号分隔" />
            </label>
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">内容来源</span>
              <Select value={sourceDraft.origin} onValueChange={(value: SourceOrigin) => setSourceDraft((current) => ({ ...current, origin: value }))}>
                <SelectTrigger aria-label="内容来源"><SelectValue /></SelectTrigger>
                <SelectContent className="z-[80]">
                  {SOURCE_ORIGINS.map((origin) => <SelectItem key={origin} value={origin}>{origin}</SelectItem>)}
                </SelectContent>
              </Select>
            </label>
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">内容类别</span>
              <Select value={sourceDraft.kind} onValueChange={(value: SourceKind) => setSourceDraft((current) => ({ ...current, kind: value }))}>
                <SelectTrigger aria-label="内容类别"><SelectValue /></SelectTrigger>
                <SelectContent className="z-[80]">
                  {SOURCE_KINDS.map((kind) => <SelectItem key={kind} value={kind}>{kind}</SelectItem>)}
                </SelectContent>
              </Select>
            </label>
            <fieldset>
              <legend className="mb-1.5 text-[12px] font-medium text-foreground">资料用途（可多选）</legend>
              <div className="grid grid-cols-2 gap-2">
                {SOURCE_PURPOSES.map((purpose) => (
                  <label key={purpose} className="flex cursor-pointer items-center gap-2 rounded-md border border-border px-2.5 py-2 text-[11px] text-foreground">
                    <input
                      type="checkbox"
                      checked={sourceDraft.purposes.includes(purpose)}
                      onChange={() => toggleSourcePurpose(purpose)}
                      className="size-3.5 accent-[#28573A]"
                    />
                    {purpose}
                  </label>
                ))}
              </div>
              <p className="mt-1.5 text-[10px] leading-4 text-muted-foreground">供 AI 检索只表示可作为回答来源，不代表你已经学习或掌握。</p>
            </fieldset>
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">摘要或笔记内容</span>
              <Textarea value={sourceDraft.summary} onChange={(event) => setSourceDraft((current) => ({ ...current, summary: event.target.value }))} placeholder="记录关键结论、来源或待验证问题" className="min-h-24" />
            </label>
          </div>
          <DialogFooter className="flex-row justify-end">
            <button type="button" onClick={() => setSourceEditorOpen(false)} className="rounded-lg px-4 py-2 text-[13px] font-medium text-muted-foreground hover:bg-secondary">取消</button>
            <button type="button" onClick={saveSource} disabled={!sourceDraft.title.trim()} className="rounded-lg bg-primary px-4 py-2 text-[13px] font-semibold text-white hover:bg-primary/90 disabled:opacity-40">保存</button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={masteryEditorOpen} onOpenChange={setMasteryEditorOpen}>
        <DialogContent className="z-[70] max-w-[360px] gap-5 rounded-2xl border-border bg-white p-5">
          <DialogHeader className="text-left">
            <DialogTitle>编辑掌握情况</DialogTitle>
            <DialogDescription>能力等级与证据来源分开记录，变更会保留依据。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">掌握等级</span>
              <Select value={masteryLevel} onValueChange={(value: MasteryLevel) => setMasteryLevel(value)}>
                <SelectTrigger aria-label="掌握等级"><SelectValue /></SelectTrigger>
                <SelectContent className="z-[80]">
                  {MASTERY_LEVELS.map((level) => <SelectItem key={level} value={level}>{level}</SelectItem>)}
                </SelectContent>
              </Select>
            </label>
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">掌握证据</span>
              <Textarea value={masteryEvidence} onChange={(event) => setMasteryEvidence(event.target.value)} placeholder="例如：无提示讲了 5 分钟，但遗漏故障恢复" className="min-h-24" />
            </label>
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">证据来源</span>
              <Select value={evidenceScope} onValueChange={(value: EvidenceScope) => {
                setEvidenceScope(value);
                if (value !== "生产实践") setProductionConfirmed(false);
              }} disabled={masteryLevel === "暂无证据"}>
                <SelectTrigger aria-label="证据来源"><SelectValue /></SelectTrigger>
                <SelectContent className="z-[80]">
                  {EVIDENCE_SCOPES.map((scope) => <SelectItem key={scope} value={scope}>{scope}</SelectItem>)}
                </SelectContent>
              </Select>
            </label>
            {evidenceScope === "生产实践" && masteryLevel !== "暂无证据" && (
              <label className="flex cursor-pointer items-start gap-2.5 border-l-2 border-[#C7944A] bg-[#FFF9EC] px-3 py-2.5">
                <input
                  type="checkbox"
                  checked={productionConfirmed}
                  onChange={(event) => setProductionConfirmed(event.target.checked)}
                  className="mt-0.5 size-4 accent-[#28573A]"
                />
                <span className="text-[10px] leading-4 text-muted-foreground">我确认该证据来自真实工作事实。AI 不会自动完成这项确认，也不会因此自动提高能力等级。</span>
              </label>
            )}
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">下次复习</span>
              <Input type="date" value={nextReview} onChange={(event) => setNextReview(event.target.value)} min={new Date().toISOString().slice(0, 10)} />
            </label>
          </div>
          <DialogFooter className="flex-row justify-end">
            <button type="button" onClick={() => setMasteryEditorOpen(false)} className="rounded-lg px-4 py-2 text-[13px] font-medium text-muted-foreground hover:bg-secondary">取消</button>
            <button
              type="button"
              onClick={saveMastery}
              disabled={
                (masteryLevel !== "暂无证据" && !masteryEvidence.trim())
                || (evidenceScope === "生产实践" && masteryLevel !== "暂无证据" && !productionConfirmed)
              }
              className="rounded-lg bg-primary px-4 py-2 text-[13px] font-semibold text-white hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-40"
            >
              提交变更
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </motion.main>
  );
}