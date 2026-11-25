import type React from "react";

interface FormFieldProps {
  label: string;
  hint?: string;
  children: React.ReactNode;
}

export function FormField({ label, hint, children }: FormFieldProps) {
  return (
    <div className="min-w-0">
      <div className="flex items-center justify-between gap-2 mb-1.5 min-h-[20px]">
        <label className="text-[13px] font-medium text-muted leading-5 truncate">
          {label}
        </label>
        {hint && (
          <span className="text-[11px] text-muted shrink-0 whitespace-nowrap">
            {hint}
          </span>
        )}
      </div>
      {children}
    </div>
  );
}
