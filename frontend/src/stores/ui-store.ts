import { create } from "zustand";

interface UIState {
  sidebarOpen: boolean;
  streamConnected: boolean;
  networkBannerDismissed: boolean;
  setSidebarOpen: (open: boolean) => void;
  setStreamConnected: (connected: boolean) => void;
  dismissNetworkBanner: () => void;
  resetNetworkBanner: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  sidebarOpen: false,
  streamConnected: false,
  networkBannerDismissed: false,
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  setStreamConnected: (connected) => set({ streamConnected: connected }),
  dismissNetworkBanner: () => set({ networkBannerDismissed: true }),
  resetNetworkBanner: () => set({ networkBannerDismissed: false }),
}));
