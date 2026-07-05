import { useState, useRef, useEffect } from "react";
import {
  User, MessageCircle, UtensilsCrossed, CheckSquare,
  Home, Send, Mic, MicOff, ChevronRight, Flame, Droplets,
  Moon, Activity, Heart, Apple, Coffee, Sun, Star,
  Edit3, Save, TrendingUp, Award, Plus, Check,
  Volume2, Sparkles, ArrowRight
} from "lucide-react";
import { sendChat } from "./api";

type Tab = "home" | "profile" | "meals" | "checkin";

interface Message {
  id: number;
  role: "user" | "ai";
  text?: string;
  card?: CardPayload;
  time: string;
}

interface CardPayload {
  type: "meal" | "checkin" | "stats";
  data: unknown;
}

// ── helpers ───────────────────────────────────────────────────────────────────
function nowStr() {
  return new Date().toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
}

// Intent matching → card or text
function resolveIntent(text: string): { reply: string; card?: CardPayload } {
  const t = text.toLowerCase();
  if (t.includes("吃") || t.includes("饭") || t.includes("饮食") || t.includes("热量") || t.includes("营养")) {
    return {
      reply: "根据你今天的活动量，我为你推荐以下膳食计划 🍽️",
      card: { type: "meal", data: null },
    };
  }
  if (t.includes("打卡") || t.includes("任务") || t.includes("完成") || t.includes("习惯")) {
    return {
      reply: "这是你今天的健康打卡进度，点击可以直接记录 ✅",
      card: { type: "checkin", data: null },
    };
  }
  if (t.includes("数据") || t.includes("步数") || t.includes("睡眠") || t.includes("今天") || t.includes("状态")) {
    return {
      reply: "来看看你今天的身体数据吧 📊",
      card: { type: "stats", data: null },
    };
  }
  const generic = [
    "根据你的档案，建议今天保持 1800 大卡摄入，蛋白质 ≥ 60g，多喝水！",
    "你今天的步数还不错！再走 2000 步就能完成今日目标，加油 💪",
    "睡眠对代谢影响很大。建议你今晚 22:30 前入睡，保证 7-8 小时。",
    "你已经连续打卡 7 天了！保持这个节奏，下周会看到明显变化的 🌱",
  ];
  return { reply: generic[Math.floor(Math.random() * generic.length)] };
}

// ── Inline cards ──────────────────────────────────────────────────────────────
function MealCard() {
  const meals = [
    { time: "早餐", name: "燕麦粥 + 水煮蛋 + 牛奶", cal: 380, emoji: "🥣", tags: ["高蛋白", "低脂"] },
    { time: "午餐", name: "清蒸鲈鱼 + 糙米饭 + 西兰花", cal: 560, emoji: "🐟", tags: ["均衡"] },
    { time: "加餐", name: "希腊酸奶 + 蓝莓", cal: 140, emoji: "🫐", tags: ["益生菌"] },
    { time: "晚餐", name: "番茄牛肉汤 + 全麦面包", cal: 480, emoji: "🍲", tags: ["饱腹感"] },
  ];
  const [logged, setLogged] = useState<Set<number>>(new Set());
  return (
    <div className="bg-card border border-border rounded-2xl overflow-hidden w-full mt-2">
      <div className="px-4 py-3 border-b border-border flex items-center justify-between">
        <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">今日膳食推荐</span>
        <span className="text-xs text-orange-500 font-semibold">共 {meals.reduce((s, m) => s + m.cal, 0)} kcal</span>
      </div>
      <div className="divide-y divide-border">
        {meals.map((m, i) => (
          <button
            key={i}
            onClick={() => setLogged(prev => { const s = new Set(prev); s.has(i) ? s.delete(i) : s.add(i); return s; })}
            className={`w-full flex items-center gap-3 px-4 py-3 text-left transition-colors ${logged.has(i) ? "bg-emerald-50/60" : "hover:bg-secondary"}`}
          >
            <span className="text-xl">{m.emoji}</span>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-1.5 mb-0.5">
                <span className="text-[10px] font-semibold text-muted-foreground uppercase">{m.time}</span>
                <span className="text-[10px] text-orange-400 font-medium">{m.cal} kcal</span>
              </div>
              <p className="text-xs font-medium truncate">{m.name}</p>
            </div>
            <div className={`w-5 h-5 rounded-full flex items-center justify-center flex-shrink-0 transition-colors ${logged.has(i) ? "bg-primary text-primary-foreground" : "border-2 border-border"}`}>
              {logged.has(i) && <Check size={10} />}
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}

function CheckInCard() {
  const items = [
    { label: "完成步数目标", icon: "🏃", done: false },
    { label: "喝够 8 杯水", icon: "💧", done: false },
    { label: "7h 以上睡眠", icon: "🌙", done: true },
    { label: "吃早饭", icon: "☀️", done: true },
    { label: "完成今日锻炼", icon: "🔥", done: false },
    { label: "晒 15 分钟太阳", icon: "🌿", done: false },
  ];
  const [done, setDone] = useState<Set<number>>(new Set(items.map((it, i) => it.done ? i : -1).filter(i => i >= 0)));
  const pct = Math.round((done.size / items.length) * 100);
  return (
    <div className="bg-card border border-border rounded-2xl overflow-hidden w-full mt-2">
      <div className="px-4 py-3 border-b border-border flex items-center justify-between">
        <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">今日打卡</span>
        <span className="text-xs text-primary font-semibold">{done.size}/{items.length} · {pct}%</span>
      </div>
      <div className="px-4 pt-3 pb-1">
        <div className="h-1.5 bg-muted rounded-full overflow-hidden mb-3">
          <div className="h-full bg-primary rounded-full transition-all duration-500" style={{ width: `${pct}%` }} />
        </div>
        <div className="grid grid-cols-2 gap-1.5 pb-3">
          {items.map((it, i) => (
            <button
              key={i}
              onClick={() => setDone(prev => { const s = new Set(prev); s.has(i) ? s.delete(i) : s.add(i); return s; })}
              className={`flex items-center gap-2 rounded-xl px-3 py-2.5 text-left transition-colors ${done.has(i) ? "bg-emerald-50 border border-emerald-200" : "bg-secondary border border-transparent hover:border-border"}`}
            >
              <span className="text-base">{it.icon}</span>
              <span className={`text-[11px] font-medium leading-tight ${done.has(i) ? "text-emerald-700 line-through" : "text-foreground"}`}>{it.label}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

function StatsCard() {
  const stats = [
    { label: "步数", value: "6,240", target: "10k", pct: 62, color: "#2E5E3E" },
    { label: "睡眠", value: "7.2h", target: "8h", pct: 90, color: "#7B5EA7" },
    { label: "饮水", value: "1.4L", target: "2L", pct: 70, color: "#3B82F6" },
    { label: "卡路里", value: "1,280", target: "1800", pct: 71, color: "#F97316" },
  ];
  return (
    <div className="bg-card border border-border rounded-2xl overflow-hidden w-full mt-2">
      <div className="px-4 py-3 border-b border-border">
        <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">今日数据总览</span>
      </div>
      <div className="grid grid-cols-2 gap-px bg-border">
        {stats.map((s) => (
          <div key={s.label} className="bg-card px-4 py-3">
            <p className="text-[10px] text-muted-foreground font-medium mb-1">{s.label}</p>
            <p className="text-base font-semibold">{s.value}</p>
            <p className="text-[10px] text-muted-foreground mb-2">/ {s.target}</p>
            <div className="h-1 bg-muted rounded-full overflow-hidden">
              <div className="h-full rounded-full" style={{ width: `${s.pct}%`, background: s.color }} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function InlineCard({ card }: { card: CardPayload }) {
  if (card.type === "meal") return <MealCard />;
  if (card.type === "checkin") return <CheckInCard />;
  if (card.type === "stats") return <StatsCard />;
  return null;
}

// ── Chat (main page) ──────────────────────────────────────────────────────────
function ChatCore({ goTo, messages, setMessages, isTyping, setIsTyping }: {
  goTo: (t: Tab) => void;
  messages: Message[];
  setMessages: React.Dispatch<React.SetStateAction<Message[]>>;
  isTyping: boolean;
  setIsTyping: React.Dispatch<React.SetStateAction<boolean>>;
}) {
  const [input, setInput] = useState("");
  const [isRecording, setIsRecording] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, isTyping]);

  function send(text: string) {
    if (!text.trim()) return;
    const userMsg: Message = { id: Date.now(), role: "user", text: text.trim(), time: nowStr() };
    setMessages(p => [...p, userMsg]);
    setInput("");
    setIsTyping(true);

    // 文本回复与卡片均由后端 /api/v1/chat 返回：前端不再做意图判断。
    sendChat(text.trim())
      .then(({ reply, card }) => {
        const aiMsg: Message = { id: Date.now() + 1, role: "ai", text: reply, time: nowStr() };
        if (card) aiMsg.card = { type: card.type, data: null };
        setMessages(p => [...p, aiMsg]);
      })
      .catch(() => {
        setMessages(p => [...p, {
          id: Date.now() + 1, role: "ai",
          text: "（连不上后端，确认 Go 服务已在 8091 启动）", time: nowStr(),
        }]);
      })
      .finally(() => setIsTyping(false));
  }

  const suggestions = ["今天吃什么", "打卡任务", "今日数据", "如何改善睡眠？", "我需要多少蛋白质？"];

  return (
    <div className="flex flex-col h-full">
      {/* Messages */}
      <div className="flex-1 overflow-y-auto space-y-5 pb-2">
        {messages.map((msg) => (
          <div key={msg.id} className={`flex gap-2.5 ${msg.role === "user" ? "flex-row-reverse" : ""}`}>
            {msg.role === "ai" && (
              <div className="w-8 h-8 rounded-full bg-primary flex-shrink-0 flex items-center justify-center text-primary-foreground mt-0.5">
                <Sparkles size={13} />
              </div>
            )}
            <div className={`max-w-[84%] flex flex-col gap-1 ${msg.role === "user" ? "items-end" : "items-start"}`}>
              {msg.text && (
                <div className={`rounded-2xl px-4 py-3 text-sm leading-relaxed whitespace-pre-line ${
                  msg.role === "user"
                    ? "bg-primary text-primary-foreground rounded-tr-sm"
                    : "bg-card border border-border rounded-tl-sm"
                }`}>
                  {msg.text}
                </div>
              )}
              {msg.card && <InlineCard card={msg.card} />}
              <span className="text-[10px] text-muted-foreground px-1">{msg.time}</span>
            </div>
          </div>
        ))}

        {isTyping && (
          <div className="flex gap-2.5 items-end">
            <div className="w-8 h-8 rounded-full bg-primary flex-shrink-0 flex items-center justify-center text-primary-foreground">
              <Sparkles size={13} />
            </div>
            <div className="bg-card border border-border rounded-2xl rounded-tl-sm px-4 py-3 flex gap-1.5 items-center">
              {[0, 1, 2].map(i => (
                <span key={i} className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce" style={{ animationDelay: `${i * 0.15}s` }} />
              ))}
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      {/* Suggestions */}
      <div className="flex gap-2 overflow-x-auto no-scrollbar py-2">
        {suggestions.map(s => (
          <button
            key={s}
            onClick={() => send(s)}
            className="flex-shrink-0 text-xs bg-secondary text-secondary-foreground px-3 py-1.5 rounded-full hover:bg-muted transition-colors font-medium border border-border"
          >
            {s}
          </button>
        ))}
      </div>

      {/* Input bar */}
      <div className="flex items-center gap-2 pt-2 border-t border-border">
        <button
          onClick={() => setIsRecording(r => !r)}
          className={`p-3 rounded-full transition-colors flex-shrink-0 ${
            isRecording ? "bg-red-500 text-white animate-pulse" : "bg-secondary text-muted-foreground hover:text-foreground"
          }`}
        >
          {isRecording ? <MicOff size={18} /> : <Mic size={18} />}
        </button>
        <input
          type="text"
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => e.key === "Enter" && send(input)}
          placeholder="问我任何健康问题…"
          className="flex-1 bg-input-background rounded-full px-4 py-2.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring border border-border"
        />
        <button
          onClick={() => send(input)}
          disabled={!input.trim()}
          className="p-3 rounded-full bg-primary text-primary-foreground disabled:opacity-40 transition-opacity hover:opacity-90 flex-shrink-0"
        >
          <Send size={18} />
        </button>
      </div>
    </div>
  );
}

// ── Other pages (lite versions) ───────────────────────────────────────────────
function ProfileTab() {
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState({ name: "李明", age: "28", height: "175", weight: "72", goal: "减脂塑形", activity: "中等活动", allergy: "无" });
  const [saved, setSaved] = useState(false);
  const bmi = (Number(form.weight) / (Number(form.height) / 100) ** 2).toFixed(1);
  const bmiLabel = Number(bmi) < 18.5 ? "偏瘦" : Number(bmi) < 24 ? "正常" : Number(bmi) < 28 ? "偏胖" : "肥胖";
  const goals = ["减脂塑形", "增肌力量", "保持健康", "控制体重", "改善睡眠"];
  const activities = ["久坐办公", "轻度活动", "中等活动", "高强度运动"];

  return (
    <div className="space-y-4">
      <div className="bg-card border border-border rounded-2xl p-5 flex items-center gap-4">
        <div className="w-16 h-16 rounded-2xl bg-primary flex items-center justify-center text-2xl text-primary-foreground font-display">
          {form.name[0]}
        </div>
        <div className="flex-1">
          <h2 className="text-lg font-semibold">{form.name}</h2>
          <p className="text-sm text-muted-foreground">{form.goal} · {form.activity}</p>
          <span className="inline-block text-xs bg-secondary text-secondary-foreground px-2 py-1 rounded-full font-medium mt-1.5">
            BMI {bmi} · {bmiLabel}
          </span>
        </div>
        <button
          onClick={() => { setEditing(e => !e); setSaved(false); }}
          className={`p-2.5 rounded-xl border transition-colors ${editing ? "bg-primary text-primary-foreground border-primary" : "border-border text-muted-foreground"}`}
        >
          <Edit3 size={16} />
        </button>
      </div>

      <div className="bg-card border border-border rounded-2xl p-5 space-y-4">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">基本信息</h3>
        {([ ["姓名","name","text",""], ["年龄","age","number","岁"], ["身高","height","number","cm"], ["体重","weight","number","kg"], ["过敏","allergy","text",""] ] as const).map(([label, key, type, suffix]) => (
          <div key={key} className="flex items-center justify-between">
            <label className="text-sm text-muted-foreground w-20">{label}</label>
            {editing
              ? <div className="flex items-center gap-1 flex-1 justify-end">
                  <input type={type} value={form[key]} onChange={e => setForm({...form, [key]: e.target.value})}
                    className="w-32 text-right bg-input-background rounded-lg px-3 py-1.5 text-sm border border-border focus:outline-none focus:ring-1 focus:ring-ring" />
                  {suffix && <span className="text-sm text-muted-foreground ml-1">{suffix}</span>}
                </div>
              : <span className="text-sm font-medium">{form[key]}{suffix ? " "+suffix : ""}</span>
            }
          </div>
        ))}
      </div>

      <div className="bg-card border border-border rounded-2xl p-5 space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">健康目标</h3>
        <div className="flex flex-wrap gap-2">
          {goals.map(g => (
            <button key={g} disabled={!editing} onClick={() => setForm({...form, goal: g})}
              className={`px-3 py-1.5 rounded-full text-sm font-medium transition-colors disabled:cursor-default ${form.goal===g ? "bg-primary text-primary-foreground" : "bg-secondary text-secondary-foreground"}`}>
              {g}
            </button>
          ))}
        </div>
      </div>

      <div className="bg-card border border-border rounded-2xl p-5 space-y-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">活动水平</h3>
        <div className="grid grid-cols-2 gap-2">
          {activities.map(a => (
            <button key={a} disabled={!editing} onClick={() => setForm({...form, activity: a})}
              className={`py-2 rounded-xl text-sm font-medium transition-colors disabled:cursor-default ${form.activity===a ? "bg-primary text-primary-foreground" : "bg-secondary text-secondary-foreground"}`}>
              {a}
            </button>
          ))}
        </div>
      </div>

      {editing && (
        <button onClick={() => { setSaved(true); setEditing(false); }}
          className="w-full bg-primary text-primary-foreground rounded-xl py-3.5 font-semibold flex items-center justify-center gap-2 hover:opacity-90">
          <Save size={16} /> 保存档案
        </button>
      )}
      {saved && !editing && (
        <div className="flex items-center gap-2 bg-emerald-50 border border-emerald-200 rounded-xl px-4 py-3 text-emerald-700 text-sm">
          <Check size={16} /> 档案已保存
        </div>
      )}
    </div>
  );
}

function MealsTab() {
  const meals = [
    { time: "早餐", name: "燕麦粥 + 水煮蛋 + 牛奶", cal: 380, emoji: "🥣", tags: ["高蛋白", "低脂"] },
    { time: "午餐", name: "清蒸鲈鱼 + 糙米饭 + 西兰花", cal: 560, emoji: "🐟", tags: ["均衡", "优质蛋白"] },
    { time: "加餐", name: "希腊酸奶 + 蓝莓", cal: 140, emoji: "🫐", tags: ["益生菌", "抗氧化"] },
    { time: "晚餐", name: "番茄牛肉汤 + 全麦面包", cal: 480, emoji: "🍲", tags: ["铁元素", "饱腹感"] },
  ];
  const [logged, setLogged] = useState(new Set<number>());
  const totalCal = [...logged].reduce((s, i) => s + meals[i].cal, 0);

  return (
    <div className="space-y-4">
      <div className="bg-card border border-border rounded-2xl p-4 flex items-center gap-4">
        <div className="relative w-16 h-16">
          <svg viewBox="0 0 64 64" className="w-16 h-16 -rotate-90">
            <circle cx="32" cy="32" r="26" fill="none" stroke="var(--muted)" strokeWidth="7" />
            <circle cx="32" cy="32" r="26" fill="none" stroke="var(--primary)" strokeWidth="7"
              strokeLinecap="round"
              strokeDasharray={`${2*Math.PI*26}`}
              strokeDashoffset={`${2*Math.PI*26*(1-totalCal/1800)}`}
              className="transition-all duration-700" />
          </svg>
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <span className="text-sm font-bold">{totalCal}</span>
            <span className="text-[8px] text-muted-foreground">kcal</span>
          </div>
        </div>
        <div className="flex-1 space-y-1">
          <div className="flex justify-between text-sm"><span className="text-muted-foreground">目标</span><span className="font-semibold">1800 kcal</span></div>
          <div className="flex justify-between text-sm"><span className="text-muted-foreground">已记录</span><span className="font-semibold text-primary">{totalCal} kcal</span></div>
          <div className="flex justify-between text-sm"><span className="text-muted-foreground">剩余</span><span className="font-semibold text-emerald-600">{1800-totalCal} kcal</span></div>
        </div>
      </div>

      <div className="space-y-2">
        {meals.map((m, i) => (
          <button key={i} onClick={() => setLogged(prev => { const s=new Set(prev); s.has(i)?s.delete(i):s.add(i); return s; })}
            className={`w-full flex items-center gap-3 bg-card border rounded-xl px-4 py-3.5 text-left transition-colors ${logged.has(i)?"border-primary/25 bg-emerald-50/40":"border-border hover:bg-secondary"}`}>
            <span className="text-2xl">{m.emoji}</span>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <span className="text-[10px] font-semibold text-muted-foreground uppercase">{m.time}</span>
                <span className="text-xs text-orange-500 font-semibold">{m.cal} kcal</span>
              </div>
              <p className="text-sm font-medium">{m.name}</p>
              <div className="flex gap-1.5 mt-1.5">
                {m.tags.map(tag => <span key={tag} className="text-[10px] bg-secondary px-2 py-0.5 rounded-full font-medium text-secondary-foreground">{tag}</span>)}
              </div>
            </div>
            <div className={`w-6 h-6 rounded-full flex items-center justify-center flex-shrink-0 transition-colors ${logged.has(i)?"bg-primary text-primary-foreground":"border-2 border-border"}`}>
              {logged.has(i) && <Check size={12} />}
            </div>
          </button>
        ))}
      </div>

      {/* Water */}
      <div className="bg-card border border-border rounded-2xl p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold">今日饮水</h3>
          <span className="text-xs text-blue-500 font-medium">4/8 杯</span>
        </div>
        <div className="flex gap-1.5">
          {Array.from({length: 8}, (_, i) => (
            <div key={i} className={`flex-1 h-8 rounded-lg ${i<4?"bg-blue-400":"bg-muted"}`} />
          ))}
        </div>
        <p className="text-xs text-muted-foreground text-center mt-2">每杯 250ml · 已喝 1000ml</p>
      </div>
    </div>
  );
}

function CheckInTab() {
  const items = [
    { label: "完成步数目标", emoji: "🏃", done: false, sub: "6,240 / 10,000 步" },
    { label: "喝够 8 杯水", emoji: "💧", done: false, sub: "4 / 8 杯" },
    { label: "7h 以上睡眠", emoji: "🌙", done: true, sub: "昨晚 7.2h" },
    { label: "吃早饭", emoji: "☀️", done: true, sub: "完成" },
    { label: "完成今日锻炼", emoji: "🔥", done: false, sub: "" },
    { label: "晒 15 分钟太阳", emoji: "🌿", done: false, sub: "" },
    { label: "冥想 5 分钟", emoji: "🧘", done: false, sub: "" },
    { label: "摄入足够蔬果", emoji: "🥦", done: true, sub: "完成" },
  ];
  const [done, setDone] = useState(new Set(items.map((it, i) => it.done ? i : -1).filter(i => i >= 0)));
  const pct = Math.round(done.size / items.length * 100);
  const moods = ["😴","😕","😐","😊","🤩"];
  const [mood, setMood] = useState<number|null>(null);
  const streakDays = ["一","二","三","四","五","六","日"];
  const streakDone = [true,true,true,true,true,true,false];

  return (
    <div className="space-y-4">
      {/* Header card */}
      <div className="bg-primary rounded-2xl p-5 text-primary-foreground">
        <div className="flex items-center justify-between mb-3">
          <div>
            <p className="text-sm opacity-70">今日完成进度</p>
            <h2 className="text-3xl font-display">{pct}%</h2>
          </div>
          <div className="text-right">
            <p className="text-xs opacity-70 mb-1">本周连续打卡</p>
            <div className="flex gap-1">
              {streakDays.map((d, i) => (
                <div key={i} className={`w-7 h-7 rounded-full flex items-center justify-center text-[10px] font-bold ${streakDone[i]?"bg-white text-primary":"bg-white/20 text-white/50"}`}>
                  {streakDone[i] ? "✓" : d}
                </div>
              ))}
            </div>
          </div>
        </div>
        <div className="h-1.5 bg-white/20 rounded-full overflow-hidden">
          <div className="h-full bg-white rounded-full transition-all duration-700" style={{ width: `${pct}%` }} />
        </div>
      </div>

      {/* Mood */}
      <div className="bg-card border border-border rounded-2xl p-4">
        <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">今日心情</p>
        <div className="flex justify-around">
          {moods.map((m, i) => (
            <button key={i} onClick={() => setMood(i)}
              className={`flex flex-col items-center gap-1 transition-transform ${mood===i?"scale-125":""}`}>
              <span className="text-2xl">{m}</span>
              {mood===i && <span className="w-1.5 h-1.5 rounded-full bg-primary" />}
            </button>
          ))}
        </div>
      </div>

      {/* Tasks */}
      <div className="grid grid-cols-1 gap-2">
        {items.map((it, i) => (
          <button key={i} onClick={() => setDone(prev => { const s=new Set(prev); s.has(i)?s.delete(i):s.add(i); return s; })}
            className={`flex items-center gap-3 rounded-xl px-4 py-3 text-left transition-all ${done.has(i)?"bg-emerald-50/60 border border-emerald-200":"bg-card border border-border hover:bg-secondary"}`}>
            <span className="text-xl">{it.emoji}</span>
            <div className="flex-1">
              <p className={`text-sm font-medium ${done.has(i)?"line-through text-muted-foreground":""}`}>{it.label}</p>
              {it.sub && <p className="text-[11px] text-muted-foreground mt-0.5">{it.sub}</p>}
            </div>
            <div className={`w-6 h-6 rounded-full flex items-center justify-center flex-shrink-0 transition-colors ${done.has(i)?"bg-primary text-primary-foreground":"border-2 border-border"}`}>
              {done.has(i) && <Check size={12} />}
            </div>
          </button>
        ))}
      </div>

      <button className="w-full bg-primary text-primary-foreground rounded-xl py-3.5 font-semibold flex items-center justify-center gap-2 hover:opacity-90">
        <TrendingUp size={18} /> 查看本周健康报告
      </button>
    </div>
  );
}

// ── Nav ────────────────────────────────────────────────────────────────────────
function BottomNav({ active, setTab }: { active: Tab; setTab: (t: Tab) => void }) {
  const sides: { id: Tab; icon: React.ReactNode; label: string }[] = [
    { id: "home", icon: <Home size={20} />, label: "主页" },
    { id: "meals", icon: <UtensilsCrossed size={20} />, label: "饮食" },
    { id: "checkin", icon: <CheckSquare size={20} />, label: "打卡" },
    { id: "profile", icon: <User size={20} />, label: "档案" },
  ];

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 bg-card border-t border-border">
      <div className="flex justify-around items-center max-w-lg mx-auto px-2">
        {/* Left two */}
        {sides.slice(0, 2).map(item => (
          <button key={item.id} onClick={() => setTab(item.id)}
            className={`flex flex-col items-center gap-1 py-3 px-4 transition-colors ${active===item.id?"text-primary":"text-muted-foreground hover:text-foreground"}`}>
            <span className={`transition-transform ${active===item.id?"scale-110":""}`}>{item.icon}</span>
            <span className="text-[10px] font-medium">{item.label}</span>
          </button>
        ))}

        {/* Center chat FAB — placeholder space */}
        <div className="w-16" />

        {/* Right two */}
        {sides.slice(2).map(item => (
          <button key={item.id} onClick={() => setTab(item.id)}
            className={`flex flex-col items-center gap-1 py-3 px-4 transition-colors ${active===item.id?"text-primary":"text-muted-foreground hover:text-foreground"}`}>
            <span className={`transition-transform ${active===item.id?"scale-110":""}`}>{item.icon}</span>
            <span className="text-[10px] font-medium">{item.label}</span>
          </button>
        ))}
      </div>

      {/* Floating center button — sits above nav */}
      <div className="absolute left-1/2 -translate-x-1/2 -top-6">
        <div className="w-14 h-14 rounded-full bg-primary shadow-lg shadow-primary/30 flex items-center justify-center border-4 border-background">
          <Sparkles size={22} className="text-primary-foreground" />
        </div>
        <p className="text-[9px] font-semibold text-primary text-center mt-0.5 tracking-wide">AI 助手</p>
      </div>
    </nav>
  );
}

// ── App root ──────────────────────────────────────────────────────────────────
export default function App() {
  const [tab, setTab] = useState<Tab | "chat">("chat");

  // 聊天状态提升到 App（跨 tab 不卸载的公共祖先），
  // 否则切走时 ChatCore 被卸载，其内部 useState 会被销毁，导致对话丢失。
  const [messages, setMessages] = useState<Message[]>([
    {
      id: 1, role: "ai", time: "09:00",
      text: "你好，我是你的健康管理助手 🌿\n\n可以问我关于饮食、运动、睡眠的任何问题，或者直接说「今天吃什么」「打卡任务」「今日数据」——我会帮你一键查看。",
    },
  ]);
  const [isTyping, setIsTyping] = useState(false);

  const titles: Record<string, string> = {
    chat: "健康助手",
    home: "健康管理",
    profile: "个人档案",
    meals: "今天吃什么",
    checkin: "今日打卡",
  };

  return (
    <div className="min-h-screen bg-background" style={{ fontFamily: "'Plus Jakarta Sans', sans-serif" }}>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=DM+Serif+Display:ital@0;1&family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap');
        .font-display { font-family: 'DM Serif Display', serif; }
        * { scrollbar-width: none; }
        *::-webkit-scrollbar { display: none; }
      `}</style>

      <div className="max-w-lg mx-auto relative flex flex-col" style={{ minHeight: "100dvh" }}>
        {/* Header */}
        <header className="sticky top-0 z-40 bg-background/90 backdrop-blur-sm border-b border-border px-4 py-4 flex items-center gap-3">
          {tab === "chat" && (
            <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground">
              <Sparkles size={14} />
            </div>
          )}
          <div className="flex-1">
            <h1 className="text-base font-semibold leading-none">{titles[tab]}</h1>
            {tab === "chat" && <p className="text-[11px] text-emerald-500 font-medium mt-0.5">● 在线</p>}
          </div>
          {tab === "chat" && (
            <button className="p-2 text-muted-foreground hover:text-foreground">
              <Volume2 size={18} />
            </button>
          )}
        </header>

        {/* Content */}
        <main className="flex-1 px-4 pt-4 pb-28 overflow-y-auto flex flex-col">
          {tab === "chat" && (
            <ChatCore
              goTo={(t) => setTab(t)}
              messages={messages}
              setMessages={setMessages}
              isTyping={isTyping}
              setIsTyping={setIsTyping}
            />
          )}
          {tab === "home" && (
            <div className="space-y-4">
              {/* Home — compact overview with chat CTA */}
              <div className="bg-primary rounded-2xl p-5 text-primary-foreground relative overflow-hidden">
                <div className="absolute top-0 right-0 w-28 h-28 bg-white/5 rounded-full -translate-y-8 translate-x-8" />
                <p className="text-sm opacity-70">2025年7月5日 · 周六</p>
                <h1 className="text-2xl font-display mt-1">早安，李明 👋</h1>
                <p className="text-sm mt-2 opacity-80">今日健康指数 82 · 连续打卡 7 天 🔥</p>
                <button onClick={() => setTab("chat")}
                  className="mt-4 flex items-center gap-2 bg-white/15 hover:bg-white/25 transition-colors rounded-full px-4 py-2 text-sm font-medium">
                  <Sparkles size={14} /> 问问健康助手 <ArrowRight size={14} />
                </button>
              </div>
              <div className="grid grid-cols-2 gap-3">
                {[
                  { label: "步数", value: "6,240", target: "10,000", pct: 62, color: "#2E5E3E", bg: "bg-emerald-50", tc: "text-emerald-700" },
                  { label: "饮水", value: "1.4L", target: "2.0L", pct: 70, color: "#3B82F6", bg: "bg-blue-50", tc: "text-blue-600" },
                  { label: "睡眠", value: "7.2h", target: "8.0h", pct: 90, color: "#7B5EA7", bg: "bg-violet-50", tc: "text-violet-600" },
                  { label: "热量", value: "1,280", target: "1,800", pct: 71, color: "#F97316", bg: "bg-orange-50", tc: "text-orange-600" },
                ].map(s => (
                  <div key={s.label} className="bg-card border border-border rounded-xl p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className={`text-xs font-semibold ${s.tc}`}>{s.label}</span>
                    </div>
                    <p className="text-xl font-bold">{s.value}</p>
                    <p className="text-xs text-muted-foreground mb-2">/ {s.target}</p>
                    <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                      <div className="h-full rounded-full" style={{ width: `${s.pct}%`, background: s.color }} />
                    </div>
                  </div>
                ))}
              </div>
              <div className="grid grid-cols-3 gap-2">
                {([["meals","今天吃什么","🍽️"],["checkin","今日打卡","✅"],["profile","我的档案","👤"]] as const).map(([t, label, icon]) => (
                  <button key={t} onClick={() => setTab(t)}
                    className="bg-card border border-border rounded-xl py-4 flex flex-col items-center gap-2 hover:bg-secondary transition-colors">
                    <span className="text-xl">{icon}</span>
                    <span className="text-xs font-medium text-muted-foreground">{label}</span>
                  </button>
                ))}
              </div>
            </div>
          )}
          {tab === "profile" && <ProfileTab />}
          {tab === "meals" && <MealsTab />}
          {tab === "checkin" && <CheckInTab />}
        </main>

        {/* Bottom nav with chat FAB */}
        <nav className="fixed bottom-0 left-0 right-0 z-50 bg-card border-t border-border">
          <div className="flex justify-around items-center max-w-lg mx-auto px-2">
            {([["home","主页",<Home size={20}/>],["meals","饮食",<UtensilsCrossed size={20}/>]] as const).map(([id, label, icon]) => (
              <button key={id} onClick={() => setTab(id)}
                className={`flex flex-col items-center gap-1 py-3 px-5 transition-colors ${tab===id?"text-primary":"text-muted-foreground"}`}>
                <span className={`transition-transform ${tab===id?"scale-110":""}`}>{icon}</span>
                <span className="text-[10px] font-medium">{label}</span>
              </button>
            ))}

            {/* FAB */}
            <div className="flex flex-col items-center pb-1">
              <button onClick={() => setTab("chat")}
                className={`w-14 h-14 rounded-full flex items-center justify-center shadow-lg transition-all -mt-6 border-4 border-background ${tab==="chat"?"bg-foreground":"bg-primary"} shadow-primary/25`}>
                <Sparkles size={22} className="text-primary-foreground" />
              </button>
              <span className={`text-[10px] font-semibold mt-0.5 ${tab==="chat"?"text-foreground":"text-primary"}`}>AI 助手</span>
            </div>

            {([["checkin","打卡",<CheckSquare size={20}/>],["profile","档案",<User size={20}/>]] as const).map(([id, label, icon]) => (
              <button key={id} onClick={() => setTab(id)}
                className={`flex flex-col items-center gap-1 py-3 px-5 transition-colors ${tab===id?"text-primary":"text-muted-foreground"}`}>
                <span className={`transition-transform ${tab===id?"scale-110":""}`}>{icon}</span>
                <span className="text-[10px] font-medium">{label}</span>
              </button>
            ))}
          </div>
        </nav>
      </div>
    </div>
  );
}
