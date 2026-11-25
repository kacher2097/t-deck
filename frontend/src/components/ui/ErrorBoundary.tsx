import { AlertTriangle, RotateCcw } from "lucide-react";
import React from "react";
import { UI_BTN_PRIMARY } from "./uiTokens";

interface ErrorBoundaryProps {
  children: React.ReactNode;
  scope?: string;
}

interface ErrorBoundaryState {
  hasError: boolean;
  message: string;
}

export class ErrorBoundary extends React.Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, message: "" };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return {
      hasError: true,
      message: error?.message || "Đã xảy ra lỗi hiển thị giao diện.",
    };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo): void {
    console.error(`[ErrorBoundary:${this.props.scope || "root"}]`, error, info);
  }

  handleReset = () => {
    this.setState({ hasError: false, message: "" });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-[240px] h-full w-full bg-canvas text-primary flex items-center justify-center p-8">
          <div className="max-w-md w-full p-6 rounded-lg bg-surface border border-edge space-y-4 text-center">
            <AlertTriangle className="w-8 h-8 text-status-warn mx-auto" />
            <div className="space-y-1">
              <h2 className="font-heading text-base font-semibold text-primary">
                Giao diện gặp sự cố ngoài ý muốn
              </h2>
              <p className="text-xs text-muted leading-relaxed break-words">
                {this.state.message}
              </p>
            </div>
            <button
              type="button"
              onClick={this.handleReset}
              className={UI_BTN_PRIMARY}
            >
              <RotateCcw className="w-4 h-4" />
              <span>Khôi phục giao diện</span>
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
