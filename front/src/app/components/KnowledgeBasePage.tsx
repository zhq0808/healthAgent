import { useRef, useState } from "react";
import { motion } from "motion/react";
import {
  BookOpenText,
  CheckCircle2,
  Clock3,
  FileText,
  Link2,
  Pencil,
  Plus,
  Search,
  Upload,
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

type SourceType = "PDF" | "Markdown" | "网页" | "手动笔记";
type MasteryLevel = "未掌握" | "有印象" | "能讲清" | "能应用";

interface KnowledgeSource {
  id: string;
  title: string;
  type: SourceType;
  meta: string;
  topics: string[];
  summary: string;
  updatedAt: string;
}

interface MasteryItem {
  id: string;
  name: string;
  category: string;
  level: MasteryLevel;
  evidence: string;
  nextReview: string;
}

interface SourceDraft {
  title: string;
  topics: string;
  summary: string;
}

const MASTERY_LEVELS: MasteryLevel[] = ["未掌握", "有印象", "能讲清", "能应用"];

const INITIAL_SOURCES: KnowledgeSource[] = [
  {
    id: "go-concurrency",
    title: "Go 并发核心笔记",
    type: "Markdown",
    meta: "18.4 KB",
    topics: ["GMP", "Channel", "Worker Pool"],
    summary: "整理 goroutine 调度、channel 语义与有界并发的关键约束。",
    updatedAt: "今天 09:40",
  },
  {
    id: "kafka-interview",
    title: "Kafka 面试问题集",
    type: "PDF",
    meta: "1.8 MB",
    topics: ["可靠性", "消息积压", "消费幂等"],
    summary: "覆盖生产、存储、消费和故障恢复四条面试追问链路。",
    updatedAt: "昨天 21:16",
  },
  {
    id: "project-stories",
    title: "项目难点故事库",
    type: "手动笔记",
    meta: "6 个故事",
    topics: ["分布式事务", "系统设计"],
    summary: "按背景、冲突、方案、取舍和结果沉淀可验证的项目表达素材。",
    updatedAt: "7月20日",
  },
];

const INITIAL_MASTERY: MasteryItem[] = [
  {
    id: "kafka-backlog",
    name: "Kafka 消息积压排查",
    category: "消息队列",
    level: "能讲清",
    evidence: "已无提示复述完整排查链路，异常恢复细节仍需补充。",
    nextReview: "2026-07-24",
  },
  {
    id: "go-gc",
    name: "Go GC 三色标记",
    category: "Go Runtime",
    level: "有印象",
    evidence: "能说出颜色定义，但写屏障触发时机容易混淆。",
    nextReview: "2026-07-23",
  },
  {
    id: "worker-pool",
    name: "有界 Worker Pool",
    category: "Go 并发",
    level: "能应用",
    evidence: "已独立实现取消、错误收敛和有界并发测试。",
    nextReview: "2026-07-29",
  },
  {
    id: "mysql-mvcc",
    name: "MySQL MVCC",
    category: "数据库",
    level: "未掌握",
    evidence: "Read View 与可见性判断还不能脱离笔记说明。",
    nextReview: "2026-07-22",
  },
];

const LEVEL_STYLES: Record<MasteryLevel, string> = {
  未掌握: "bg-[#FCECE9] text-[#A84F44]",
  有印象: "bg-[#FFF4D9] text-[#946B16]",
  能讲清: "bg-[#E8F0F7] text-[#456986]",
  能应用: "bg-[#E4F1E8] text-[#2E6941]",
};

function sourceTypeFor(file: File): SourceType {
  const extension = file.name.split(".").pop()?.toLowerCase();
  if (extension === "pdf") return "PDF";
  return "Markdown";
}

export function KnowledgeBasePage() {
  const [sources, setSources] = useState(INITIAL_SOURCES);
  const [masteryItems, setMasteryItems] = useState(INITIAL_MASTERY);
  const [query, setQuery] = useState("");
  const [sourceEditorOpen, setSourceEditorOpen] = useState(false);
  const [editingSourceID, setEditingSourceID] = useState<string | null>(null);
  const [sourceDraft, setSourceDraft] = useState<SourceDraft>({ title: "", topics: "", summary: "" });
  const [masteryEditorOpen, setMasteryEditorOpen] = useState(false);
  const [editingMasteryID, setEditingMasteryID] = useState<string | null>(null);
  const [masteryLevel, setMasteryLevel] = useState<MasteryLevel>("有印象");
  const [masteryEvidence, setMasteryEvidence] = useState("");
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
    setSourceDraft({ title: "", topics: "", summary: "" });
    setSourceEditorOpen(true);
  };

  const openSourceEditor = (source: KnowledgeSource) => {
    setEditingSourceID(source.id);
    setSourceDraft({
      title: source.title,
      topics: source.topics.join("、"),
      summary: source.summary,
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
            ? { ...source, title, topics, summary: sourceDraft.summary.trim(), updatedAt: "刚刚" }
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
        topics: ["待整理"],
        summary: "资料已上传，等待提取知识点与生成摘要。",
        updatedAt: "刚刚",
      },
      ...current,
    ]);
  };

  const openMasteryEditor = (item: MasteryItem) => {
    setEditingMasteryID(item.id);
    setMasteryLevel(item.level);
    setMasteryEvidence(item.evidence);
    setNextReview(item.nextReview);
    setMasteryEditorOpen(true);
  };

  const saveMastery = () => {
    if (!editingMasteryID) return;
    setMasteryItems((current) =>
      current.map((item) =>
        item.id === editingMasteryID
          ? { ...item, level: masteryLevel, evidence: masteryEvidence.trim(), nextReview }
          : item,
      ),
    );
    setMasteryEditorOpen(false);
  };

  const masteredCount = masteryItems.filter(
    (item) => item.level === "能讲清" || item.level === "能应用",
  ).length;

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
          <p className="mt-0.5 text-[12px] text-muted-foreground">管理资料，也管理你真正掌握了什么</p>
        </div>
        <span className="rounded-full bg-secondary px-2.5 py-1 text-[10px] text-muted-foreground">演示数据</span>
      </header>

      <Tabs defaultValue="sources" className="min-h-0 flex-1 gap-0">
        <div className="px-5">
          <TabsList className="grid h-10 w-full grid-cols-2 rounded-lg bg-secondary p-1">
            <TabsTrigger value="sources" className="rounded-md text-[12px]">资料</TabsTrigger>
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
              placeholder="搜索资料或知识点"
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
                <p className="mt-3 text-[11px] leading-5 text-muted-foreground">{source.summary}</p>
                <div className="mt-3 flex flex-wrap gap-1.5">
                  {source.topics.map((topic) => (
                    <span key={topic} className="rounded-md bg-secondary px-2 py-1 text-[10px] text-primary">{topic}</span>
                  ))}
                </div>
              </article>
            ))}
          </div>
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
                    <CheckCircle2 size={16} />
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
                  <span className={`rounded-md px-2 py-1 text-[10px] font-medium ${LEVEL_STYLES[item.level]}`}>{item.level}</span>
                  <span className="flex items-center gap-1 text-[10px] text-muted-foreground"><Clock3 size={11} />复习 {item.nextReview}</span>
                </div>
                <p className="mt-3 border-t border-border pt-3 text-[11px] leading-5 text-muted-foreground">{item.evidence}</p>
              </article>
            ))}
          </div>
        </TabsContent>
      </Tabs>

      <Dialog open={sourceEditorOpen} onOpenChange={setSourceEditorOpen}>
        <DialogContent className="z-[70] max-w-[360px] gap-5 rounded-2xl border-border bg-white p-5">
          <DialogHeader className="text-left">
            <DialogTitle>{editingSourceID ? "编辑资料" : "新建笔记"}</DialogTitle>
            <DialogDescription>维护资料名称、知识点标签和内容摘要。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">名称</span>
              <Input value={sourceDraft.title} onChange={(event) => setSourceDraft((current) => ({ ...current, title: event.target.value }))} placeholder="资料或笔记名称" autoFocus />
            </label>
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">知识点标签</span>
              <Input value={sourceDraft.topics} onChange={(event) => setSourceDraft((current) => ({ ...current, topics: event.target.value }))} placeholder="使用逗号分隔" />
            </label>
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
            <DialogDescription>用最近一次主动输出或编码结果校准状态。</DialogDescription>
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
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">下次复习</span>
              <Input type="date" value={nextReview} onChange={(event) => setNextReview(event.target.value)} min={new Date().toISOString().slice(0, 10)} />
            </label>
          </div>
          <DialogFooter className="flex-row justify-end">
            <button type="button" onClick={() => setMasteryEditorOpen(false)} className="rounded-lg px-4 py-2 text-[13px] font-medium text-muted-foreground hover:bg-secondary">取消</button>
            <button type="button" onClick={saveMastery} className="rounded-lg bg-primary px-4 py-2 text-[13px] font-semibold text-white hover:bg-primary/90">保存修改</button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </motion.main>
  );
}