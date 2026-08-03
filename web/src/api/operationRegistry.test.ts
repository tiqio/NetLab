import { describe, expect, it } from "vitest";
import { operationRegistry } from "./operationRegistry";
import { generatedApi } from "./index";
describe("operation registry", () => {
  it("maps every visible mutation to a generated API method", () => {
    for (const operation of Object.values(operationRegistry))
      expect(typeof generatedApi[operation.apiMethod]).toBe("function");
  });
});
