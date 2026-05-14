"use client";

const SYSTEM_COLORS: Record<string, string> = {
  brand: "#0f172a",
  account: "#3b82f6",
  files: "#10b981",
  ai: "#8b5cf6",
  chat: "#f59e0b",
  monitor: "#ef4444",
};

const SYSTEM_NAMES: Record<string, string> = {
  brand: "Opus",
  account: "Account",
  files: "Files",
  ai: "AI",
  chat: "Chat",
  monitor: "Monitor",
};

type SystemKey = keyof typeof SYSTEM_COLORS;

function OpusMark({ className, system }: { className?: string; system?: SystemKey }) {
  const color = system ? SYSTEM_COLORS[system] : SYSTEM_COLORS.brand;
  return (
    <svg viewBox="0 0 32 32" fill="none" className={className}>
      <rect width="32" height="32" rx="8" fill={color} />
      <circle cx="16" cy="16" r="8" stroke="white" strokeWidth="2.8" fill="none" />
      <rect x="20.5" y="5" width="4" height="4" rx="0.6" transform="rotate(45 22.5 7)" fill="white" />
    </svg>
  );
}

export function OpusSystemLogo({ system, className }: { system: SystemKey; className?: string }) {
  return (
    <div className={`flex items-center gap-2 ${className || ""}`}>
      <OpusMark system={system} className="h-7 w-7 shrink-0" />
      <div className="flex items-baseline gap-1.5">
        <span className="font-bold text-[15px] tracking-tight text-gray-900">Opus</span>
        <span className="text-[10px] font-bold uppercase tracking-[0.1em]" style={{ color: SYSTEM_COLORS[system] }}>
          {SYSTEM_NAMES[system]}
        </span>
      </div>
    </div>
  );
}

export function OpusBrandLogo({
  className,
  size = "md",
  system,
}: {
  className?: string;
  size?: "sm" | "md" | "lg";
  system?: SystemKey;
}) {
  const iconSizes = { sm: "h-7 w-7", md: "h-9 w-9", lg: "h-11 w-11" };
  const textSizes = { sm: "text-lg", md: "text-xl", lg: "text-2xl" };
  return (
    <div className={`flex items-center gap-2.5 ${className || ""}`}>
      <OpusMark system={system} className={`${iconSizes[size]} shrink-0`} />
      <span className={`font-extrabold tracking-tight text-gray-900 ${textSizes[size]}`}>
        OPUS
      </span>
    </div>
  );
}

export { OpusMark, SYSTEM_COLORS };
export type { SystemKey };
