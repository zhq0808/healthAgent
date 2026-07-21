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
  X,
} from "lucide-react";

interface TodoItem {
  id: string;
  title: string;
  type: string;
  duration: string;
  completed: boolean;
  icon: typeof BookOpenCheck;
}

interface DayProgress {
  completed: number;
  total: number;
}

const WEEKDAYS = ["日", "一", "二", "三", "四", "五", "六"];

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

export function Dashboard({ onClose }: { onClose: () => void }) {
  const [todos, setTodos] = useState(INITIAL_TODOS);
  const completedCount = todos.filter((todo) => todo.completed).length;
  const completionRate = Math.round((completedCount / todos.length) * 100);

  const toggleTodo = (id: string) => {
    setTodos((current) =>
      current.map((todo) =>
        todo.id === id ? { ...todo, completed: !todo.completed } : todo,
      ),
    );
  };

  return (
    <motion.aside
      key="learning-dashboard"
      initial={{ x: "100%" }}
      animate={{ x: 0 }}
      exit={{ x: "100%" }}
      transition={{ type: "spring", stiffness: 320, damping: 34 }}
      className="absolute inset-y-0 right-0 z-50 flex w-[92%] max-w-[430px] flex-col overflow-hidden rounded-l-[28px] border-l border-white/80 bg-background shadow-[-18px_0_48px_rgba(24,45,32,0.18)]"
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
          <button
            type="button"
            onClick={onClose}
            aria-label="关闭学习看板"
            className="flex size-8 items-center justify-center rounded-full bg-secondary transition-colors hover:bg-accent"
          >
            <X size={15} className="text-muted-foreground" />
          </button>
        </div>
      </div>

      <div
        className="flex-1 overflow-y-auto px-6 pb-8"
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
            <div className="text-right">
              <p className="text-[13px] font-semibold text-foreground">
                {completedCount}/{todos.length} 已完成
              </p>
              <p className="text-[10px] text-muted-foreground">完成度 {completionRate}%</p>
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
                <button
                  key={todo.id}
                  type="button"
                  onClick={() => toggleTodo(todo.id)}
                  aria-pressed={todo.completed}
                  className="flex w-full items-center gap-3 rounded-lg border border-border bg-card px-3.5 py-3 text-left transition-colors hover:bg-secondary/40"
                >
                  <span
                    className={`flex size-6 flex-shrink-0 items-center justify-center rounded-md border transition-colors ${
                      todo.completed
                        ? "border-primary bg-primary text-white"
                        : "border-[#B9C7BD] bg-white text-transparent"
                    }`}
                  >
                    <Check size={14} strokeWidth={3} />
                  </span>
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
                </button>
              );
            })}
          </div>
        </section>
      </div>
    </motion.aside>
  );
}