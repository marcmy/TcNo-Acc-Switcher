import { describe, expect, it } from "vitest";
import type { PlatformStartup } from "../../bindings/TcNo-Acc-Switcher/internal/platform/models.js";
import { applySinglePlatformStartupRoute, parseHash, serializeRoute, type Route } from "./routeCodec";

function startup(overrides: Partial<PlatformStartup> = {}): PlatformStartup {
  return {
    homePlatformOrder: ["Steam"],
    allPlatformNames: ["Steam"],
    disabledPlatformNames: [],
    platformsFileMissing: false,
    ...overrides,
  } as PlatformStartup;
}

describe("applySinglePlatformStartupRoute", () => {
  it("opens the only enabled platform from the startup home route", () => {
    expect(applySinglePlatformStartupRoute({ page: "home" }, startup())).toEqual({
      page: "platform",
      platformName: "Steam",
    });
  });

  it.each([
    startup({ homePlatformOrder: [] }),
    startup({
      homePlatformOrder: ["Steam", "Epic Games"],
      allPlatformNames: ["Steam", "Epic Games"],
    }),
    startup({ platformsFileMissing: true }),
  ])("keeps home when automatic navigation is not applicable", (state) => {
    expect(applySinglePlatformStartupRoute({ page: "home" }, state)).toEqual({ page: "home" });
  });

  it("preserves an explicit startup route", () => {
    const route: Route = { page: "settings" };
    expect(applySinglePlatformStartupRoute(route, startup())).toBe(route);
  });
});

describe("Steam confirmations route", () => {
  it("round-trips the singleton window route without account data in the URL", () => {
    const route: Route = { page: "steam-confirmations" };
    expect(serializeRoute(route)).toBe("#/steam/confirmations");
    expect(parseHash("#/steam/confirmations")).toEqual(route);
  });
});

describe("parseHash untrusted routes", () => {
  it.each(["constructor", "__proto__", "toString", "hasOwnProperty"])(
    "does not invoke inherited object method %s", (name) => {
      expect(parseHash("#/" + name)).toBeNull();
    },
  );

  it("keeps legitimate parameterized platform routes", () => {
    expect(parseHash("#/platform/Steam%20Beta")).toEqual({
      page: "platform",
      platformName: "Steam Beta",
    });
  });
});

describe("parseHash handles malformed URLs", () => {
  it.each(["#/%", "#/platform/%E0%A4%A", "#/steam/browser/%GG"])(
    "returns null for invalid percent escapes", (hash) => {
      expect(parseHash(hash)).toBeNull();
    },
  );
  it("preserves the legacy preview route alias", () => {
    expect(parseHash("#/test")).toEqual({ page: "preview-css" });
  });
});
