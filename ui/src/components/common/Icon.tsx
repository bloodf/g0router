import type { ReactNode } from "react";

export function Icon({
  name,
  className,
  size = 18,
}: {
  name: string;
  className?: string;
  size?: number;
}) {
  return (
    <span
      className={"material-symbols-outlined " + (className ?? "")}
      style={{ fontSize: size, lineHeight: 1 }}
      aria-hidden
    >
      {name}
    </span>
  );
}

export function IconText({
  icon,
  children,
  className,
}: {
  icon: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <span className={"inline-flex items-center gap-1.5 " + (className ?? "")}>
      <Icon name={icon} size={16} />
      {children}
    </span>
  );
}
