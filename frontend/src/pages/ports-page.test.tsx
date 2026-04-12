import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { PortsPage } from "./ports-page";
import { testSnapshot } from "../test/fixtures";

const mockUseSystemSnapshotQuery = vi.fn();

vi.mock("../lib/api-client", () => ({
  useSystemSnapshotQuery: () => mockUseSystemSnapshotQuery(),
}));

describe("PortsPage", () => {
  beforeEach(() => {
    mockUseSystemSnapshotQuery.mockReturnValue({
      data: {
        ...testSnapshot,
        port_bindings: [
          {
            protocol: "tcp",
            local_addr: "127.0.0.1",
            local_port: 3000,
            remote_addr: "",
            remote_port: 0,
            state: "LISTEN",
            pid: 101,
            process: "chrome.exe",
            label: "Local dev",
          },
          {
            protocol: "udp",
            local_addr: "0.0.0.0",
            local_port: 5353,
            remote_addr: "224.0.0.251",
            remote_port: 5353,
            state: "",
            pid: 202,
            process: "mdns.exe",
            label: "mDNS",
          },
        ],
      },
      isLoading: false,
    });
  });

  it("filters bindings by protocol chip", async () => {
    const user = userEvent.setup();
    render(<PortsPage />);

    await user.click(screen.getByRole("button", { name: "TCP" }));

    await waitFor(() => {
      expect(screen.queryByText("mdns.exe")).not.toBeInTheDocument();
    });
    expect(screen.getAllByText("chrome.exe").length).toBeGreaterThan(0);
  });

  it("shows a detail panel for the selected binding", async () => {
    const user = userEvent.setup();
    render(<PortsPage />);

    await user.click(screen.getAllByRole("button", { name: "Details" })[0]!);

    expect(screen.getByText("Local endpoint")).toBeInTheDocument();
    expect(screen.getAllByText(/127.0.0.1:3000/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Waiting for inbound connections/).length).toBeGreaterThan(0);
  });
});
