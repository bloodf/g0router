import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { useNotificationStore } from "./notification";

describe("notificationStore", () => {
  beforeEach(() => {
    useNotificationStore.setState({ toasts: [] });
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("push auto-dismisses after duration", () => {
    const { push } = useNotificationStore.getState();
    push({ message: "hi", duration: 50 });
    expect(useNotificationStore.getState().toasts).toHaveLength(1);

    vi.advanceTimersByTime(60);
    expect(useNotificationStore.getState().toasts).toHaveLength(0);
  });
});
