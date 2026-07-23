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
      className="absolute inset-x-0 bottom-0 z-40 border-t border-border bg-background/95 px-5 pb-[max(8px,env(safe-area-inset-bottom))] backdrop-blur-md"
    >
      <div className="mx-auto grid h-[66px] max-w-md grid-cols-3 items-end">
        {NAV_ITEMS.map((item) => {
          const Icon = item.icon;
          const active = activeView === item.id;
          const isPractice = item.id === "practice";

          if (isPractice) {
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => onChange(item.id)}
                aria-current={active ? "page" : undefined}
                aria-label="开始练习"
                className="relative flex h-[58px] flex-col items-center justify-end pb-0.5 text-[11px] font-semibold text-primary"
              >
                <span
                  className={`absolute -top-6 flex size-14 items-center justify-center rounded-full border-[5px] border-background shadow-[0_5px_18px_rgba(46,94,62,0.22)] ${
                    active
                      ? "bg-primary text-primary-foreground"
                      : "bg-[#DDECE1] text-primary"
                  }`}
                >
                  <Icon size={23} strokeWidth={active ? 2.5 : 2.2} />
                </span>
                <span className="leading-none">{item.label}</span>
              </button>
            );
          }

          return (
            <button
              key={item.id}
              type="button"
              onClick={() => onChange(item.id)}
              aria-current={active ? "page" : undefined}
              className={`relative flex h-[58px] flex-col items-center justify-center gap-1 text-[11px] font-medium transition-colors ${
                active ? "text-primary" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <Icon size={20} strokeWidth={active ? 2.4 : 1.8} />
              <span>{item.label}</span>
              {active && (
                <span className="absolute bottom-0 h-0.5 w-5 rounded-full bg-primary" />
              )}
            </button>
          );
        })}
      </div>
    </nav>
  );
}