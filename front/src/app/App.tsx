import { useState, useRef, useEffect } from "react";
import { AnimatePresence } from "motion/react";
import { StatusTags, StatusTagDef } from "./components/StatusTags";
import { MealSuggestionCard } from "./components/MealSuggestionCard";
import { ActionCard } from "./components/ActionCard";
import { WarningCard } from "./components/WarningCard";
import { ConfirmationCard } from "./components/ConfirmationCard";
import { MorningGreetingCard } from "./components/MorningGreetingCard";
import { InputDock } from "./components/InputDock";
import { UserMessage } from "./components/UserMessage";
import { AIMessage } from "./components/AIMessage";
import { MealCard } from "./components/MealCard";
import {
  createOrResumeGuest,
  ensureSessionID,
  sendChatStream,
} from "./api/chat";
import { AuthPage } from "./pages/AuthPage";

interface Message {
  id: string;
  type:
    | "user"
    | "ai"
    | "meal-card"
    | "meal-suggestion"
    | "action-card"
    | "warning"
    | "confirmation"
    | "morning-greeting";
  content: any;
}

interface ActionItem {
  id: string;
  text: string;
  completed: boolean;
}

const initialActions: ActionItem[] = [
  { id: "1", text: "餐前测血糖", completed: true },
  { id: "2", text: "饮用 500ml 温水", completed: false },
  { id: "3", text: "完成 20 分钟散步", completed: false },
];

const INITIAL_TAGS: StatusTagDef[] = [
  {
    id: "blood-sugar",
    emoji: "🩸",
    label: "关注控糖",
    color: "bg-[#EEF2E8] text-[#5A7C5C]",
    state: "active",
    sparklineData: [
      { v: 5.3 },
      { v: 5.5 },
      { v: 5.4 },
      { v: 5.2 },
      { v: 5.5 },
      { v: 5.4 },
      { v: 5.4 },
    ],
    summary: "近期血糖平稳，7 天均值 5.4",
  },
  {
    id: "energy",
    emoji: "✨",
    label: "满血复活",
    color: "bg-[#FFF5E6] text-[#A67C52]",
    state: "active",
    sparklineData: [
      { v: 4 },
      { v: 5 },
      { v: 6 },
      { v: 6 },
      { v: 7 },
      { v: 8 },
      { v: 8 },
    ],
    summary: "精力状态回升，最近保持得不错，继续加油！",
  },
  {
    id: "period",
    emoji: "🩸",
    label: "姨妈期",
    color: "bg-[#FBEAEC] text-[#B5687A]",
    state: "active",
    sparklineData: [
      { v: 3 },
      { v: 4 },
      { v: 5 },
      { v: 5 },
      { v: 4 },
      { v: 3 },
      { v: 3 },
    ],
    summary: "经期第 2 天，注意保暖、多喝温水，适度休息。",
  },
];

function HealthWorkspace() {
  const [tags, setTags] = useState<StatusTagDef[]>(INITIAL_TAGS);
  const [actions, setActions] = useState<ActionItem[]>(initialActions);
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "welcome",
      type: "ai",
      content: "太好了！精气神回来了 ✨ 中午想吃点什么，还是看看本周的总结？",
    },
    { id: "meal-card", type: "meal-card", content: null },
  ]);

  const [pendingConfirmation, setPendingConfirmation] =
    useState<Message | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({
        top: scrollRef.current.scrollHeight,
        behavior: "smooth",
      });
    }
  }, [messages, pendingConfirmation]);

  // Helpers
  const latestMealId = [...messages]
    .reverse()
    .find((m) => m.type === "meal-suggestion")?.id;
  const latestActionId = [...messages]
    .reverse()
    .find((m) => m.type === "action-card")?.id;

  const dismissMorningGreeting = () => {
    setMessages((prev) =>
      prev.filter((m) => m.id !== "morning-greeting")
    );
  };

  const handleMorningReply = (reply: "recovered" | "still-tired") => {
    dismissMorningGreeting();

    if (reply === "recovered") {
      setTags((prev) =>
        prev.map((t) =>
          t.id === "energy" ? { ...t, state: "dismissed" } : t
        )
      );
      setTimeout(() => {
        setMessages((prev) => [
          ...prev,
          {
            id: Date.now().toString(),
            type: "ai",
            content:
              "太好了！精气神回来了 ✨ 今天继续保持，身体在慢慢向好的方向走。",
          },
        ]);
      }, 400);
    } else {
      setTags((prev) =>
        prev.map((t) =>
          t.id === "energy" ? { ...t, state: "active" } : t
        )
      );
      setTimeout(() => {
        setMessages((prev) => [
          ...prev,
          {
            id: Date.now().toString(),
            type: "ai",
            content:
              "没关系，身体需要休息的信号值得被认真对待 🫧 今天可以减少剧烈运动，多喝温水。",
          },
        ]);
      }, 400);
    }
  };

  const handleSendMessage = async (text: string) => {
    const userMessage: Message = {
      id: Date.now().toString(),
      type: "user",
      content: text,
    };
    setMessages((prev) => [...prev, userMessage]);

    // 血糖录入 → 确认卡（本地演示交互，暂不走后端）。
    if (text.includes("血糖") && /\d/.test(text)) {
      const value = text.match(/\d+(\.\d+)?/)?.[0] || "0";
      const confirmation: Message = {
        id: Date.now().toString() + "-confirm",
        type: "confirmation",
        content: {
          type: "blood-sugar",
          data: { label: "今日血糖", value, unit: "mmol/L" },
        },
      };
      setPendingConfirmation(confirmation);
      return;
    }

    // 其余自由对话走真实 chat 接口。先插入占位气泡，返回后原地替换。
    const typingId = Date.now().toString() + "-typing";
    setMessages((prev) => [
      ...prev,
      { id: typingId, type: "ai", content: "正在思考…" },
    ]);

    try {
      const sessionID = await ensureSessionID();
	  const clientMessageID = crypto.randomUUID();
      let acc = "";
      await sendChatStream(sessionID, clientMessageID, text, (delta) => {
        acc += delta;
        setMessages((prev) =>
          prev.map((m) => (m.id === typingId ? { ...m, content: acc } : m))
        );
      });
    } catch {
      setMessages((prev) =>
        prev.map((m) =>
          m.id === typingId
            ? { ...m, content: "抱歉，暂时没能连上健康管家，请稍后再试。" }
            : m
        )
      );
    }
  };

  const handleAcceptMeal = () => {
    setMessages((prev) => [
      ...prev,
      {
        id: Date.now().toString(),
        type: "ai",
        content:
          "太棒了！记得慢慢咀嚼，让身体更好地吸收营养。餐后记得测一下血糖哦～",
      },
    ]);
  };

  const handleRegenerateMeal = () => {
    const newMeal: Message = {
      id: Date.now().toString(),
      type: "meal-suggestion",
      content: {
        meal: "🌅 午餐建议",
        title: "烤三文鱼配时蔬",
        emoji: "🐟",
        description: "富含 Omega-3 的三文鱼配上低碳水蔬菜，既美味又健康。",
        ingredients: ["三文鱼", "西兰花", "芦笋", "柠檬", "橄榄油"],
        benefits:
          "三文鱼中的 Omega-3 脂肪酸有助于改善胰岛素敏感性，搭配高纤维蔬菜能有效控制血糖上升速度。",
        gi: 28,
      },
    };
    setMessages((prev) => [...prev, newMeal]);
  };

  const handleToggleAction = (id: string) => {
    const updated = actions.map((a) =>
      a.id === id ? { ...a, completed: !a.completed } : a
    );
    setActions(updated);
    setMessages((prev) =>
      prev.map((msg) =>
        msg.type === "action-card" && msg.id === latestActionId
          ? { ...msg, content: updated }
          : msg
      )
    );
  };

  const handleConfirmData = () => {
    if (pendingConfirmation) {
      setMessages((prev) => [...prev, pendingConfirmation]);
      setPendingConfirmation(null);
      setTimeout(() => {
        setMessages((prev) => [
          ...prev,
          {
            id: Date.now().toString(),
            type: "ai",
            content:
              "收到！你的血糖数据已记录。目前数值在正常范围内，继续保持哦～",
          },
        ]);
      }, 500);
    }
  };

  const handleCancelConfirmation = () => {
    setPendingConfirmation(null);
  };

  return (
    <div className="size-full flex flex-col overflow-hidden bg-background">
      <StatusTags tags={tags} />

      <div ref={scrollRef} className="flex-1 overflow-y-auto pb-36">
        <div className="min-h-full flex flex-col justify-center pt-2">
          <AnimatePresence>
            {messages.map((message) => {
              switch (message.type) {
                case "morning-greeting":
                  return (
                    <MorningGreetingCard
                      key={message.id}
                      onReply={handleMorningReply}
                    />
                  );
                case "user":
                  return (
                    <UserMessage key={message.id} message={message.content} />
                  );
                case "ai":
                  return (
                    <AIMessage key={message.id} message={message.content} />
                  );
                case "meal-card":
                  return <MealCard key={message.id} />;
                case "meal-suggestion":
                  return (
                    <MealSuggestionCard
                      key={message.id}
                      {...message.content}
                      collapsed={message.id !== latestMealId}
                      onAccept={handleAcceptMeal}
                      onRegenerate={handleRegenerateMeal}
                    />
                  );
                case "action-card":
                  return (
                    <ActionCard
                      key={message.id}
                      actions={
                        message.id === latestActionId
                          ? actions
                          : message.content
                      }
                      collapsed={message.id !== latestActionId}
                      onToggle={handleToggleAction}
                    />
                  );
                case "warning":
                  return (
                    <WarningCard key={message.id} {...message.content} />
                  );
                case "confirmation":
                  return (
                    <ConfirmationCard
                      key={message.id}
                      {...message.content}
                      onConfirm={handleConfirmData}
                      onCancel={handleCancelConfirmation}
                    />
                  );
                default:
                  return null;
              }
            })}
          </AnimatePresence>

          <AnimatePresence>
            {pendingConfirmation && (
              <ConfirmationCard
                key={pendingConfirmation.id}
                {...pendingConfirmation.content}
                onConfirm={handleConfirmData}
                onCancel={handleCancelConfirmation}
              />
            )}
          </AnimatePresence>
        </div>
      </div>

      <InputDock
        onSendMessage={handleSendMessage}
        onVoiceInput={() => {}}
      />
    </div>
  );
}

export default function App() {
  const guestStartedKey = "health_agent_guest_started";
  const [authState, setAuthState] = useState<"auth" | "restoring" | "guest">(() =>
    localStorage.getItem(guestStartedKey) === "1" ? "restoring" : "auth"
  );

  useEffect(() => {
    if (authState !== "restoring") return;

    let cancelled = false;
    createOrResumeGuest()
      .then(() => ensureSessionID())
      .then(() => {
        if (!cancelled) setAuthState("guest");
      })
      .catch(() => {
        localStorage.removeItem(guestStartedKey);
        if (!cancelled) setAuthState("auth");
      });

    return () => {
      cancelled = true;
    };
  }, [authState]);

  const continueAsGuest = async () => {
    await createOrResumeGuest();
    await ensureSessionID();
    localStorage.setItem(guestStartedKey, "1");
    setAuthState("guest");
  };

  if (authState === "guest") {
    return <HealthWorkspace />;
  }

  if (authState === "restoring") {
    return (
      <main className="flex min-h-dvh items-center justify-center bg-[#F4F2ED] text-[#2E5E3E]">
        <p className="text-sm font-medium">正在恢复访客身份...</p>
      </main>
    );
  }

  return <AuthPage onContinueAsGuest={continueAsGuest} />;
}
