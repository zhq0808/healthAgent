import { BrainCircuit, PanelLeft } from "lucide-react";
import { ProfileButton } from "./ProfileButton";

interface AppHeaderProps {
  onOpenSessions: () => void;
  onOpenProfile: () => void;
}

// AppHeader 顶部栏：会话入口与品牌。
export function AppHeader({ onOpenSessions, onOpenProfile }: AppHeaderProps) {
  return (
    <header className="flex flex-shrink-0 items-center justify-between border-b border-border bg-background/90 px-5 pb-3 pt-6 backdrop-blur-sm">
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
      <ProfileButton onClick={onOpenProfile} />
    </header>
  );
}
