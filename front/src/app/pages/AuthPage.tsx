import { useState, type FormEvent } from "react";
import { BrainCircuit, Eye, EyeOff, Lock, User } from "lucide-react";

export interface LoginInput {
  username: string;
  password: string;
}

export interface RegisterInput extends LoginInput {
  confirmPassword: string;
}

interface AuthPageProps {
  onLogin?: (input: LoginInput) => Promise<void>;
  onRegister?: (input: RegisterInput) => Promise<void>;
  onContinueAsGuest?: () => Promise<void>;
}

type AuthMode = "login" | "register";
type PendingAction = "account" | "guest" | null;

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

export function AuthPage({
  onLogin,
  onRegister,
  onContinueAsGuest,
}: AuthPageProps) {
  const [mode, setMode] = useState<AuthMode>("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [pendingAction, setPendingAction] = useState<PendingAction>(null);
  const [error, setError] = useState("");

  const switchMode = (nextMode: AuthMode) => {
    setMode(nextMode);
    setError("");
  };

  const submitAccount = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");

    const normalizedUsername = username.trim();
    if (!normalizedUsername) {
      setError("请输入用户名");
      return;
    }
    if (!password) {
      setError("请输入密码");
      return;
    }
    if (mode === "register" && password !== confirmPassword) {
      setError("两次输入的密码不一致");
      return;
    }

    setPendingAction("account");
    try {
      if (mode === "login") {
        if (!onLogin) {
          throw new Error("登录服务尚未接入");
        }
        await onLogin({ username: normalizedUsername, password });
      } else {
        if (!onRegister) {
          throw new Error("注册服务尚未接入");
        }
        await onRegister({ username: normalizedUsername, password, confirmPassword });
      }
    } catch (submitError) {
      setError(errorMessage(submitError, mode === "login" ? "登录失败" : "注册失败"));
    } finally {
      setPendingAction(null);
    }
  };

  const continueAsGuest = async () => {
    setError("");
    if (!onContinueAsGuest) {
      setError("访客身份服务尚未接入");
      return;
    }

    setPendingAction("guest");
    try {
      await onContinueAsGuest();
    } catch (guestError) {
      setError(errorMessage(guestError, "创建访客身份失败"));
    } finally {
      setPendingAction(null);
    }
  };

  const isPending = pendingAction !== null;

  return (
    <main
      className="flex min-h-dvh flex-col bg-[#F4F2ED] text-[#1C2B1E]"
      style={{ fontFamily: "'Plus Jakarta Sans', sans-serif" }}
    >
      <section className="relative flex min-h-[38dvh] flex-none flex-col items-center justify-end overflow-hidden bg-[#2E5E3E] px-8 pb-10 pt-16">
        <div className="absolute left-0 top-0 size-48 -translate-x-20 -translate-y-20 rounded-full bg-white/5" />
        <div className="absolute right-0 top-8 size-32 translate-x-12 rounded-full bg-white/5" />
        <div className="absolute bottom-0 left-1/2 size-64 -translate-x-1/2 translate-y-32 rounded-full bg-white/5" />

        <div className="relative mb-5 flex size-16 items-center justify-center rounded-2xl border border-white/20 bg-white/15 backdrop-blur-sm">
          <BrainCircuit aria-hidden="true" size={30} className="text-white" />
        </div>
        <h1 className="font-display text-center text-3xl leading-tight text-white">
          知镜
        </h1>
        <p className="mt-2 text-center text-sm text-white/70">
          让每一次输出，都成为掌握的证据
        </p>
      </section>

      <section className="relative z-10 mx-auto -mt-4 flex w-full max-w-lg flex-1 flex-col rounded-t-3xl bg-[#F4F2ED] px-6 pb-10 pt-8">
        <div
          className="mb-7 flex rounded-xl bg-[#E8EDE5] p-1"
          role="tablist"
          aria-label="账号操作"
        >
          {(["login", "register"] as const).map((item) => (
            <button
              key={item}
              type="button"
              role="tab"
              aria-selected={mode === item}
              onClick={() => switchMode(item)}
              disabled={isPending}
              className={`flex-1 rounded-lg py-2 text-sm font-semibold transition-all disabled:cursor-not-allowed disabled:opacity-60 ${
                mode === item
                  ? "bg-[#FAFAF7] text-[#1C2B1E] shadow-sm"
                  : "text-[#6B7B6D]"
              }`}
            >
              {item === "login" ? "登录" : "注册"}
            </button>
          ))}
        </div>

        <form onSubmit={submitAccount} className="flex flex-1 flex-col" noValidate>
          <div className="flex-1 space-y-4">
            <label className="block">
              <span className="mb-2 block text-xs font-semibold uppercase text-[#6B7B6D]">用户名</span>
              <span className="flex items-center gap-3 rounded-xl border border-[rgba(46,94,62,0.12)] bg-[#EEF0EA] px-4 py-3 transition-shadow focus-within:ring-1 focus-within:ring-[#7BAF8A]">
                <User aria-hidden="true" size={16} className="shrink-0 text-[#6B7B6D]" />
                <input
                  type="text"
                  autoComplete={mode === "login" ? "username" : "new-username"}
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  disabled={isPending}
                  placeholder="请输入用户名"
                  className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-[#6B7B6D] disabled:cursor-not-allowed"
                />
              </span>
            </label>

            <label className="block">
              <span className="mb-2 block text-xs font-semibold uppercase text-[#6B7B6D]">密码</span>
              <span className="flex items-center gap-3 rounded-xl border border-[rgba(46,94,62,0.12)] bg-[#EEF0EA] px-4 py-3 transition-shadow focus-within:ring-1 focus-within:ring-[#7BAF8A]">
                <Lock aria-hidden="true" size={16} className="shrink-0 text-[#6B7B6D]" />
                <input
                  type={showPassword ? "text" : "password"}
                  autoComplete={mode === "login" ? "current-password" : "new-password"}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  disabled={isPending}
                  placeholder="请输入密码"
                  className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-[#6B7B6D] disabled:cursor-not-allowed"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((visible) => !visible)}
                  disabled={isPending}
                  className="shrink-0 text-[#6B7B6D] transition-colors hover:text-[#1C2B1E] disabled:cursor-not-allowed"
                  aria-label={showPassword ? "隐藏密码" : "显示密码"}
                  title={showPassword ? "隐藏密码" : "显示密码"}
                >
                  {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                </button>
              </span>
            </label>

            {mode === "register" && (
              <label className="block">
                <span className="mb-2 block text-xs font-semibold uppercase text-[#6B7B6D]">确认密码</span>
                <span className="flex items-center gap-3 rounded-xl border border-[rgba(46,94,62,0.12)] bg-[#EEF0EA] px-4 py-3 transition-shadow focus-within:ring-1 focus-within:ring-[#7BAF8A]">
                  <Lock aria-hidden="true" size={16} className="shrink-0 text-[#6B7B6D]" />
                  <input
                    type={showPassword ? "text" : "password"}
                    autoComplete="new-password"
                    value={confirmPassword}
                    onChange={(event) => setConfirmPassword(event.target.value)}
                    disabled={isPending}
                    placeholder="再次输入密码"
                    className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-[#6B7B6D] disabled:cursor-not-allowed"
                  />
                </span>
              </label>
            )}

            {mode === "login" && (
              <div className="text-right">
                <button type="button" className="text-xs font-medium text-[#2E5E3E] hover:underline">
                  忘记密码？
                </button>
              </div>
            )}

            {error && (
              <div role="alert" className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600">
                {error}
              </div>
            )}
          </div>

          <div className="mt-8 space-y-4">
            <button
              type="submit"
              disabled={isPending}
              className="flex w-full items-center justify-center gap-2 rounded-xl bg-[#2E5E3E] py-3.5 text-sm font-semibold text-[#F4F2ED] transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-70"
            >
              {pendingAction === "account" ? (
                <span className="flex gap-1.5" aria-label="处理中">
                  {[0, 1, 2].map((item) => (
                    <span
                      key={item}
                      className="size-1.5 animate-bounce rounded-full bg-white/70"
                      style={{ animationDelay: `${item * 0.15}s` }}
                    />
                  ))}
                </span>
              ) : mode === "login" ? "登录" : "注册并开始使用"}
            </button>

            <div className="flex items-center gap-3" aria-hidden="true">
              <span className="h-px flex-1 bg-[rgba(46,94,62,0.12)]" />
              <span className="text-xs text-[#6B7B6D]">或</span>
              <span className="h-px flex-1 bg-[rgba(46,94,62,0.12)]" />
            </div>

            <button
              type="button"
              onClick={continueAsGuest}
              disabled={isPending}
              className="w-full rounded-xl bg-[#E8EDE5] py-3.5 text-sm font-semibold text-[#2E5E3E] transition-colors hover:bg-[#dfe7dc] disabled:cursor-not-allowed disabled:opacity-70"
            >
              {pendingAction === "guest" ? "正在创建访客身份..." : "以访客身份体验"}
            </button>

            <p className="text-center text-xs text-[#6B7B6D]">
              登录即代表同意
              <button type="button" className="mx-0.5 font-medium text-[#2E5E3E] hover:underline">服务条款</button>
              与
              <button type="button" className="mx-0.5 font-medium text-[#2E5E3E] hover:underline">隐私政策</button>
            </p>
          </div>
        </form>
      </section>
    </main>
  );
}
