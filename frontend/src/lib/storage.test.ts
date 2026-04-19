import { describe, it, expect, beforeEach } from "vitest";
import { getItem, setItem, removeItem } from "./storage";

describe("storage", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("returns fallback when key is absent", () => {
    expect(getItem("missing", 42)).toBe(42);
  });

  it("stores and retrieves a string", () => {
    setItem("name", "test");
    expect(getItem("name", "")).toBe("test");
  });

  it("stores and retrieves an object", () => {
    setItem("config", { a: 1 });
    expect(getItem("config", {})).toEqual({ a: 1 });
  });

  it("uses ds: prefix in localStorage", () => {
    setItem("key", "val");
    expect(localStorage.getItem("ds:key")).toBe('"val"');
  });

  it("removes item", () => {
    setItem("key", "val");
    removeItem("key");
    expect(getItem("key", "default")).toBe("default");
  });

  it("returns fallback on corrupt JSON", () => {
    localStorage.setItem("ds:broken", "not json{");
    expect(getItem("broken", "safe")).toBe("safe");
  });
});
