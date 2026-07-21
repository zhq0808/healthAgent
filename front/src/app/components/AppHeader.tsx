import { BrainCircuit, PanelLeft } from "lucide-react";

interface AppHeaderProps {
  onOpenSessions: () => void;
}

// AppHeader 顶部栏：会话入口与品牌。
export function AppHeader({ onOpenSessions }: AppHeaderProps) {
  return (
    <header className="flex items-center px-5 pt-6 pb-3 bg-background/90 backdrop-blur-sm border-b border-border flex-shrink-0">
      <div className="flex items-center gap-2.5">
        <button
          type="button"
          onClick={onOpenSessions}
          aria-label="会话列表"
          title="会话列表"
          className="w-9 h-9 rounded-full bg-secondary flex items-center justify-center text-primary hover:bg-accent transition-colors active:scale-95"
        >
          <PanelLeft size={16} />
        </button>
        <div className="w-9 h-9 rounded-full bg-primary flex items-center justify-center flex-shrink-0">
          <BrainCircuit size={18} className="text-white" />
        </div>
        <div>
          <p className="text-[15px] font-semibold text-foreground leading-none">
            知镜
          </p>
          <p className="text-[11px] text-primary font-medium mt-0.5">
            AI 面试训练伙伴
          </p>
        </div>
      </div>
    </header>
  );
}
