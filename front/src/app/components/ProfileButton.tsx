import { UserRound } from "lucide-react";

interface ProfileButtonProps {
  onClick: () => void;
}

export function ProfileButton({ onClick }: ProfileButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="打开我的空间"
      title="我的空间"
      className="flex size-9 flex-shrink-0 items-center justify-center rounded-full border border-primary/15 bg-secondary text-primary transition-colors hover:bg-accent"
    >
      <UserRound size={17} strokeWidth={2} />
    </button>
  );
}