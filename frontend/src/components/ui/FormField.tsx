import React, { useId } from "react";

interface FormFieldProps {
  label: string;
  hint?: string;
  htmlFor?: string;
  children: React.ReactNode;
}

const CONTROL_TAGS = new Set(["input", "select", "textarea"]);

function bindFirstControlId(
  node: React.ReactNode,
  fallbackId: string,
  state: { foundId?: string },
  depth = 0
): React.ReactNode {
  if (state.foundId || depth > 6 || node == null || typeof node !== "object") {
    return node;
  }

  if (Array.isArray(node)) {
    let changed = false;
    const mapped = React.Children.map(node, (item) => {
      if (state.foundId) return item;
      const updated = bindFirstControlId(item, fallbackId, state, depth);
      if (updated !== item) changed = true;
      return updated;
    });
    return changed ? mapped : node;
  }

  if (!React.isValidElement(node)) {
    return node;
  }

  const el = node as React.ReactElement<{
    id?: string;
    name?: string;
    children?: React.ReactNode;
  }>;

  if (typeof el.type === "string" && CONTROL_TAGS.has(el.type)) {
    const resolvedId = el.props.id || fallbackId;
    state.foundId = resolvedId;
    return React.cloneElement(el, {
      id: resolvedId,
      name: el.props.name || resolvedId,
    });
  }

  if (el.props?.children !== undefined) {
    let changed = false;
    const nextChildren = React.Children.map(el.props.children, (child) => {
      if (state.foundId) return child;
      const updated = bindFirstControlId(child, fallbackId, state, depth + 1);
      if (updated !== child) changed = true;
      return updated;
    });
    if (changed) {
      return React.cloneElement(el, undefined, nextChildren);
    }
  }

  return node;
}

export function FormField({ label, hint, htmlFor, children }: FormFieldProps) {
  const autoId = useId();
  const state: { foundId?: string } = {};
  const boundChildren = bindFirstControlId(children, htmlFor || autoId, state);
  const labelFor = htmlFor || state.foundId;

  return (
    <div className="min-w-0">
      <div className="flex items-center justify-between gap-2 mb-1.5 min-h-[20px]">
        <label
          htmlFor={labelFor}
          className="text-[13px] font-medium text-muted leading-5 truncate"
        >
          {label}
        </label>
        {hint && (
          <span className="text-[11px] text-muted shrink-0 whitespace-nowrap">
            {hint}
          </span>
        )}
      </div>
      {boundChildren}
    </div>
  );
}
