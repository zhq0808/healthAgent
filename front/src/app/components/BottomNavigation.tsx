import { BookOpenText, CalendarCheck2, MessageCircleMore } from "lucide-react";

export type AppView = "practice" | "plan" | "knowledge";

const NAV_ITEMS = [
  { id: "knowledge", label: "知识库", icon: BookOpenText },
  { id: "practice", label: "练习", icon: MessageCircleMore },
  { id: "plan", label: "计划", icon: CalendarCheck2 },
] as const;

interface BottomNavigationProps {
  activeView: AppView;
  onChange: (view: AppView) => void;
}

export function BottomNavigation({ activeView, onChange }: BottomNavigationProps) {
  return (
    <nav
      aria-label="主导航"
      className="absolute inset-x-0 bottom-0 z-40 border-t border-border bg-background/95 px-5 pb-[max(10px,env(safe-area-inset-bottom))] pt-2 backdrop-blur-md"
    >
      <div className="mx-auto grid max-w-md grid-cols-3">
        {NAV_ITEMS.map((item) => {
          const Icon = item.icon;
          const active = activeView === item.id;
          return (
            <button
              key={item.id}
              type="button"
              onClick={() => onChange(item.id)}
              aria-current={active ? "page" : undefined}
              className={`flex h-12 flex-col items-center justify-center gap-1 text-[11px] font-medium transition-colors ${
                active ? "text-primary" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <Icon size={20} strokeWidth={active ? 2.4 : 1.8} />
              <span>{item.label}</span>
            </button>
          );
        })}
      </div>
    </nav>
  );
}