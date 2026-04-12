import React from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "react-router";
import { Toaster } from "sonner";
import "@fontsource-variable/inter";
import "@fontsource-variable/jetbrains-mono";
import "./index.css";
import { appRouter } from "./app/router";
import { AppProviders } from "./app/providers";

const container = document.getElementById("root");

if (!container) {
  throw new Error("Root container #root not found");
}

ReactDOM.createRoot(container).render(
  <React.StrictMode>
    <AppProviders>
      <RouterProvider router={appRouter} />
      <Toaster position="bottom-right" visibleToasts={3} richColors closeButton />
    </AppProviders>
  </React.StrictMode>,
);
