export const routeImports = {
  "/": () => import("../pages/dashboard-page"),
  "/processes": () => import("../pages/processes-page"),
  "/tree": () => import("../pages/tree-page"),
  "/ports": () => import("../pages/ports-page"),
  "/disks": () => import("../pages/disks-page"),
  "/alerts": () => import("../pages/alerts-page"),
  "/rules": () => import("../pages/rules-page"),
  "/ai": () => import("../pages/ai-page"),
  "/settings": () => import("../pages/settings-page"),
  "/about": () => import("../pages/about-page"),
} as const;

export type AppRoutePath = keyof typeof routeImports;

export function prefetchRoute(path: AppRoutePath) {
  void routeImports[path]();
}
