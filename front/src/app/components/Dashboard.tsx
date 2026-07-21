import { useState } from "react";
import { motion } from "motion/react";
import {
  BookOpenCheck,
  BriefcaseBusiness,
  Check,
  ChevronLeft,
  ChevronRight,
  Code2,
  Mic2,
  Pencil,
  Plus,
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

const TODO_TYPES = [
  { label: "知识点回顾", icon: BookOpenCheck },
  { label: "费曼输出", icon: Mic2 },
  { label: "编码验证", icon: Code2 },
  { label: "模拟面试", icon: BriefcaseBusiness },
] as const;

type TodoType = (typeof TODO_TYPES)[number]["label"];

interface TodoItem {
  id: string;
  title: string;
  type: TodoType;
  duration: string;
  completed: boolean;
  icon: typeof BookOpenCheck;
}

interface TodoDraft {
  title: string;
  type: TodoType;
  durationMinutes: string;
}

interface DayProgress {
  completed: number;
  total: number;
}

const WEEKDAYS = ["日", "一", "二", "三", "四", "五", "六"];
const EMPTY_TODO_DRAFT: TodoDraft = {
  title: "",
  type: "知识点回顾",
  durationMinutes: "10",
};

const INITIAL_TODOS: TodoItem[] = [
  {
    id: "kafka-review",
    title: "复述 Kafka 积压排查链路",
    type: "知识点回顾",
    duration: "10 分钟",
    completed: true,
    icon: BookOpenCheck,
  },
  {
    id: "gc-feynman",
    title: "无提示讲解 Go GC 三色标记",
    type: "费曼输出",
    duration: "8 分钟",
    completed: true,
    icon: Mic2,
  },
  {
    id: "worker-pool",
    title: "手写一个有界 Worker Pool",
    type: "编码验证",
    duration: "30 分钟",
    completed: false,
    icon: Code2,
  },
  {
    id: "project-interview",
    title: "模拟回答一个项目难点",
    type: "模拟面试",
    duration: "15 分钟",
    completed: false,
    icon: BriefcaseBusiness,
  },
];

function dateKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function addDays(date: Date, amount: number): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + amount);
}

function buildDemoProgress(today: Date): Record<string, DayProgress> {
  const samples: Array<[number, number, number]> = [
    [-18, 1, 3],
    [-17, 3, 3],
    [-15, 2, 4],
    [-14, 4, 4],
    [-12, 1, 2],
    [-11, 0, 3],
    [-10, 3, 4],
    [-9, 2, 2],
    [-7, 1, 4],
    [-6, 3, 3],
    [-5, 2, 3],
    [-4, 4, 4],
    [-3, 1, 3],
    [-2, 2, 4],
    [-1, 3, 3],
  ];

  return Object.fromEntries(
    samples.map(([offset, completed, total]) => [
      dateKey(addDays(today, offset)),
      { completed, total },
    ]),
  );
}

function progressColor(progress?: DayProgress): string {
  if (!progress || progress.total === 0) return "bg-transparent text-[#637169]";
  return "bg-[#2E5E3E] text-white";
}

function CalendarHeatmap({
  todos,
}: {
  todos: TodoItem[];
}) {
  const today = new Date();
  const [viewMonth, setViewMonth] = useState(
    new Date(today.getFullYear(), today.getMonth(), 1),
  );
  const firstDay = new Date(viewMonth.getFullYear(), viewMonth.getMonth(), 1);
  const calendarStart = addDays(firstDay, -firstDay.getDay());
  const days = Array.from({ length: 42 }, (_, index) =>
    addDays(calendarStart, index),
  );
  const progressByDate = buildDemoProgress(today);
  progressByDate[dateKey(today)] = {
    completed: todos.filter((todo) => todo.completed).length,
    total: todos.length,
  };
  const monthLabel = new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "long",
  }).format(viewMonth);

  const changeMonth = (amount: number) => {
    setViewMonth(
      (current) => new Date(current.getFullYear(), current.getMonth() + amount, 1),
    );
  };

  return (
    <section aria-label="学习日历">
      <div className="mb-5 flex items-center justify-between px-1">
        <button
          type="button"
          onClick={() => changeMonth(-1)}
          aria-label="上个月"
          className="flex size-8 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
        >
          <ChevronLeft size={16} />
        </button>
        <div className="text-center">
          <h3 className="text-[16px] font-semibold text-foreground">{monthLabel}</h3>
        </div>
        <button
          type="button"
          onClick={() => changeMonth(1)}
          aria-label="下个月"
          className="flex size-8 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
        >
          <ChevronRight size={16} />
        </button>
      </div>

      <div className="grid grid-cols-7 gap-y-1.5">
        {WEEKDAYS.map((weekday) => (
          <div
            key={weekday}
            className="pb-2 text-center text-[10px] font-medium text-muted-foreground"
          >
            {weekday}
          </div>
        ))}
        {days.map((date) => {
          const key = dateKey(date);
          const progress = progressByDate[key];
          const outside = date.getMonth() !== viewMonth.getMonth();
          const isToday = key === dateKey(today);
          const summary = progress
            ? `${progress.completed}/${progress.total} 项完成`
            : "当天没有任务";

          if (outside) {
            return <div key={key} aria-hidden="true" className="h-8" />;
          }

          return (
            <div
              key={key}
              title={`${date.getMonth() + 1}月${date.getDate()}日，${summary}`}
              aria-label={`${date.getMonth() + 1}月${date.getDate()}日，${summary}`}
              className="flex h-8 min-w-0 items-center justify-center"
            >
              <span
                className={`flex size-7 items-center justify-center rounded-full text-[11px] font-semibold transition-colors ${progressColor(progress)} ${
                  isToday
                    ? "ring-[3px] ring-[#D59A2F] ring-offset-2 ring-offset-background shadow-[0_0_0_1px_rgba(213,154,47,0.2),0_0_10px_rgba(213,154,47,0.38)]"
                    : ""
                }`}
              >
                {date.getDate()}
              </span>
            </div>
          );
        })}
      </div>

    </section>
  );
}

interface DashboardProps {
  onClose?: () => void;
  mode?: "drawer" | "page";
}

export function Dashboard({ onClose, mode = "drawer" }: DashboardProps) {
  const [todos, setTodos] = useState(INITIAL_TODOS);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingTodoID, setEditingTodoID] = useState<string | null>(null);
  const [draft, setDraft] = useState<TodoDraft>(EMPTY_TODO_DRAFT);
  const [editorError, setEditorError] = useState("");
  const completedCount = todos.filter((todo) => todo.completed).length;
  const completionRate = Math.round((completedCount / todos.length) * 100);

  const toggleTodo = (id: string) => {
    setTodos((current) =>
      current.map((todo) =>
        todo.id === id ? { ...todo, completed: !todo.completed } : todo,
      ),
    );
  };

  const openCreateTodo = () => {
    setEditingTodoID(null);
    setDraft(EMPTY_TODO_DRAFT);
    setEditorError("");
    setEditorOpen(true);
  };

  const openEditTodo = (todo: TodoItem) => {
    setEditingTodoID(todo.id);
    setDraft({
      title: todo.title,
      type: todo.type,
      durationMinutes: todo.duration.replace(/\D/g, "") || "10",
    });
    setEditorError("");
    setEditorOpen(true);
  };

  const saveTodo = () => {
    const title = draft.title.trim();
    const durationMinutes = Number(draft.durationMinutes);
    if (!title) {
      setEditorError("请输入 Todo 内容");
      return;
    }
    if (!Number.isInteger(durationMinutes) || durationMinutes <= 0 || durationMinutes > 480) {
      setEditorError("预计时长请输入 1 到 480 之间的整数");
      return;
    }

    const selectedType = TODO_TYPES.find((item) => item.label === draft.type) ?? TODO_TYPES[0];
    if (editingTodoID) {
      setTodos((current) =>
        current.map((todo) =>
          todo.id === editingTodoID
            ? {
                ...todo,
                title,
                type: selectedType.label,
                duration: `${durationMinutes} 分钟`,
                icon: selectedType.icon,
              }
            : todo,
        ),
      );
    } else {
      setTodos((current) => [
        ...current,
        {
          id: crypto.randomUUID(),
          title,
          type: selectedType.label,
          duration: `${durationMinutes} 分钟`,
          completed: false,
          icon: selectedType.icon,
        },
      ]);
    }
    setEditorOpen(false);
  };

  return (
    <motion.aside
      key="learning-dashboard"
      initial={mode === "drawer" ? { x: "100%" } : { opacity: 0, y: 8 }}
      animate={mode === "drawer" ? { x: 0 } : { opacity: 1, y: 0 }}
      exit={mode === "drawer" ? { x: "100%" } : { opacity: 0 }}
      transition={
        mode === "drawer"
          ? { type: "spring", stiffness: 320, damping: 34 }
          : { duration: 0.2, ease: "easeOut" }
      }
      className={
        mode === "drawer"
          ? "absolute inset-y-0 right-0 z-50 flex w-[92%] max-w-[430px] flex-col overflow-hidden rounded-l-[28px] border-l border-white/80 bg-background shadow-[-18px_0_48px_rgba(24,45,32,0.18)]"
          : "relative flex min-h-0 flex-1 flex-col overflow-hidden bg-background pb-16"
      }
    >
      <div className="flex flex-shrink-0 items-center justify-between px-5 pb-4 pt-8">
        <div>
          <h2 className="text-[18px] font-semibold text-foreground">学习看板</h2>
          <p className="text-[12px] text-muted-foreground">计划、行动与完成证据</p>
        </div>
        <div className="flex items-center gap-2">
          <span className="rounded-full bg-secondary px-2.5 py-1 text-[10px] text-muted-foreground">
            演示数据
          </span>
          {mode === "drawer" && onClose && (
            <button
              type="button"
              onClick={onClose}
              aria-label="关闭学习看板"
              className="flex size-8 items-center justify-center rounded-full bg-secondary transition-colors hover:bg-accent"
            >
              <X size={15} className="text-muted-foreground" />
            </button>
          )}
        </div>
      </div>

      <div
        className={`flex-1 overflow-y-auto px-6 ${mode === "page" ? "pb-24" : "pb-8"}`}
        style={{ scrollbarWidth: "none" }}
      >
        <CalendarHeatmap todos={todos} />

        <div className="my-5 h-px bg-border" />

        <section aria-labelledby="today-todo-title">
          <div className="mb-4 flex items-end justify-between gap-3">
            <div>
              <p className="text-[11px] font-medium text-primary">今天</p>
              <h3 id="today-todo-title" className="mt-0.5 text-[17px] font-semibold text-foreground">
                今日 Todo
              </h3>
            </div>
            <div className="flex items-center gap-3">
              <div className="text-right">
                <p className="text-[13px] font-semibold text-foreground">
                  {completedCount}/{todos.length} 已完成
                </p>
                <p className="text-[10px] text-muted-foreground">完成度 {completionRate}%</p>
              </div>
              <button
                type="button"
                onClick={openCreateTodo}
                aria-label="新增 Todo"
                title="新增 Todo"
                className="flex size-8 items-center justify-center rounded-full bg-primary text-white transition-colors hover:bg-primary/90"
              >
                <Plus size={16} />
              </button>
            </div>
          </div>

          <div className="mb-4 h-1.5 overflow-hidden rounded-full bg-secondary">
            <div
              className="h-full rounded-full bg-primary transition-[width] duration-300"
              style={{ width: `${completionRate}%` }}
            />
          </div>

          <div className="space-y-2">
            {todos.map((todo) => {
              const Icon = todo.icon;
              return (
                <div
                  key={todo.id}
                  className="flex w-full items-center gap-3 rounded-lg border border-border bg-card px-3.5 py-3 text-left transition-colors hover:bg-secondary/40"
                >
                  <button
                    type="button"
                    onClick={() => toggleTodo(todo.id)}
                    aria-label={`${todo.completed ? "标记未完成" : "标记完成"}：${todo.title}`}
                    aria-pressed={todo.completed}
                    className={`flex size-6 flex-shrink-0 items-center justify-center rounded-md border transition-colors ${
                      todo.completed
                        ? "border-primary bg-primary text-white"
                        : "border-[#B9C7BD] bg-white text-transparent"
                    }`}
                  >
                    <Check size={14} strokeWidth={3} />
                  </button>
                  <span className="flex size-8 flex-shrink-0 items-center justify-center rounded-lg bg-secondary text-primary">
                    <Icon size={15} />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span
                      className={`block truncate text-[13px] font-semibold ${
                        todo.completed ? "text-muted-foreground line-through" : "text-foreground"
                      }`}
                    >
                      {todo.title}
                    </span>
                    <span className="mt-0.5 block text-[10px] text-muted-foreground">
                      {todo.type} · {todo.duration}
                    </span>
                  </span>
                  <button
                    type="button"
                    onClick={() => openEditTodo(todo)}
                    aria-label={`编辑：${todo.title}`}
                    title="编辑 Todo"
                    className="flex size-8 flex-shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-secondary hover:text-primary"
                  >
                    <Pencil size={14} />
                  </button>
                </div>
              );
            })}
          </div>
        </section>
      </div>

      <Dialog open={editorOpen} onOpenChange={setEditorOpen}>
        <DialogContent className="z-[70] max-w-[360px] gap-5 rounded-2xl border-border bg-white p-5">
          <DialogHeader className="text-left">
            <DialogTitle>{editingTodoID ? "编辑 Todo" : "新增 Todo"}</DialogTitle>
            <DialogDescription>设置今天要完成的训练任务。</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">Todo 内容</span>
              <Input
                value={draft.title}
                onChange={(event) => {
                  setDraft((current) => ({ ...current, title: event.target.value }));
                  setEditorError("");
                }}
                placeholder="例如：复述 Kafka 消息积压排查链路"
                autoFocus
              />
            </label>

            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">训练类型</span>
              <Select
                value={draft.type}
                onValueChange={(value: TodoType) =>
                  setDraft((current) => ({ ...current, type: value }))
                }
              >
                <SelectTrigger aria-label="训练类型">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent className="z-[80]">
                  {TODO_TYPES.map((item) => (
                    <SelectItem key={item.label} value={item.label}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>

            <label className="block">
              <span className="mb-1.5 block text-[12px] font-medium text-foreground">预计时长（分钟）</span>
              <Input
                type="number"
                min={1}
                max={480}
                step={1}
                value={draft.durationMinutes}
                onChange={(event) => {
                  setDraft((current) => ({ ...current, durationMinutes: event.target.value }));
                  setEditorError("");
                }}
              />
            </label>

            {editorError && (
              <p role="alert" className="text-[12px] text-destructive">
                {editorError}
              </p>
            )}
          </div>

          <DialogFooter className="flex-row justify-end">
            <button
              type="button"
              onClick={() => setEditorOpen(false)}
              className="rounded-lg px-4 py-2 text-[13px] font-medium text-muted-foreground transition-colors hover:bg-secondary"
            >
              取消
            </button>
            <button
              type="button"
              onClick={saveTodo}
              className="rounded-lg bg-primary px-4 py-2 text-[13px] font-semibold text-white transition-colors hover:bg-primary/90"
            >
              {editingTodoID ? "保存修改" : "添加 Todo"}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </motion.aside>
  );
}