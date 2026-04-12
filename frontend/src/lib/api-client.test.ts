import { describe, expect, it } from "vitest";
import { extractApiErrorShape } from "./api-client";

describe("extractApiErrorShape", () => {
  it("supports the backend error envelope", () => {
    expect(extractApiErrorShape({ error: { code: "invalid_param", message: "pid must be uint32" } })).toEqual({
      code: "invalid_param",
      message: "pid must be uint32",
    });
  });

  it("keeps direct error payloads working", () => {
    expect(extractApiErrorShape({ code: "bad_request", message: "broken", details: "extra" })).toEqual({
      code: "bad_request",
      message: "broken",
      details: "extra",
    });
  });
});
