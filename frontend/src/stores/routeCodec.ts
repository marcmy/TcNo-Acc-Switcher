import type { PlatformStartup } from "../../bindings/TcNo-Acc-Switcher/internal/platform/models.js";

export type Route =
  | { page: "home" }
  | { page: "settings" }
  | { page: "preview-css" }
  | { page: "manage-platforms" }
  | { page: "platform"; platformName: string }
  | { page: "platform-settings"; platformName: string }
  | { page: "steam-advanced-clearing" }
  | { page: "steam-confirmations" }
  | { page: "steam-server-picker" }
  | { page: "steam-browser"; sessionId: string };

export function serializeRoute(r: Route): string {
  switch (r.page) {
    case "home":
      return "#/";
    case "settings":
      return "#/settings";
    case "preview-css":
      return "#/preview-css";
    case "manage-platforms":
      return "#/manage-platforms";
    case "platform":
      return "#/platform/" + encodeURIComponent(r.platformName);
    case "platform-settings":
      return "#/platform-settings/" + encodeURIComponent(r.platformName);
    case "steam-advanced-clearing":
      return "#/steam/advanced-clearing";
    case "steam-confirmations":
      return "#/steam/confirmations";
    case "steam-server-picker":
      return "#/steam/server-picker";
    case "steam-browser":
      return "#/steam/browser/" + encodeURIComponent(r.sessionId);
    default:
      return "#/";
  }
}

// Route names come from the URL. Keep dispatch explicit rather than invoking
// a handler chosen by a string lookup (including inherited JS method names).
export function parseHash(hash: string): Route | null {
  const h = (hash.startsWith("#") ? hash.slice(1) : hash).trim();
  if (!h || h === "/") return { page: "home" };

  const parts = h.split("/").filter((p) => p.length > 0);
  try {
    const head = decodeURIComponent(parts[0] || "").toLowerCase();
    switch (head) {
      case "":
      case "home":
        return { page: "home" };
      case "settings":
        return { page: "settings" };
      case "preview-css":
      case "test":
        return { page: "preview-css" };
      case "manage-platforms":
        return { page: "manage-platforms" };
      case "platform":
        return parts[1] ? { page: "platform", platformName: decodeURIComponent(parts[1]) } : null;
      case "platform-settings":
        return parts[1] ? { page: "platform-settings", platformName: decodeURIComponent(parts[1]) } : null;
      case "steam":
        switch (parts[1]?.toLowerCase()) {
          case "advanced-clearing":
            return { page: "steam-advanced-clearing" };
          case "confirmations":
            return { page: "steam-confirmations" };
          case "server-picker":
            return { page: "steam-server-picker" };
          case "browser":
            return parts[2] ? { page: "steam-browser", sessionId: decodeURIComponent(parts[2]) } : null;
          default:
            return null;
        }
      default:
        return null;
    }
  } catch {
    // Malformed percent-encoding must not crash hash-driven navigation.
    return null;
  }
}

export function validateRoute(r: Route, startup: PlatformStartup): Route {
  if (startup.platformsFileMissing) return { page: "home" };

  const disabled = new Set(startup.disabledPlatformNames || []);

  const nameOk = (name: string) => {
    const n = name.trim();
    if (!n) return false;
    if (!(startup.allPlatformNames || []).includes(n)) return false;
    if (disabled.has(n)) return false;
    return true;
  };

  switch (r.page) {
    case "platform":
    case "platform-settings":
      return nameOk(r.platformName) ? r : { page: "home" };
    case "steam-advanced-clearing":
    case "steam-confirmations":
    case "steam-server-picker":
      return nameOk("Steam") ? r : { page: "home" };
    default:
      return r;
  }
}

export function applySinglePlatformStartupRoute(r: Route, startup: PlatformStartup): Route {
  if (r.page !== "home" || startup.platformsFileMissing) return r;
  const enabled = startup.homePlatformOrder ?? [];
  if (enabled.length !== 1) return r;
  return validateRoute({ page: "platform", platformName: enabled[0] }, startup);
}
