import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { vi } from "vitest";
import { LiveStrip } from "./live-strip";
import { testSnapshot } from "../../test/fixtures";

const mockUseSystemSnapshotQuery = vi.fn();

vi.mock("../../lib/api-client", () => ({
  useSystemSnapshotQuery: () => mockUseSystemSnapshotQuery(),
}));

describe("LiveStrip", () => {
  beforeEach(() => {
    window.localStorage.clear();
    mockUseSystemSnapshotQuery.mockReturnValue({
      data: testSnapshot,
    });
  });

  it("stays hidden on overview", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <LiveStrip />
      </MemoryRouter>,
    );

    expect(screen.queryByText("CPU")).not.toBeInTheDocument();
  });

  it("shows interface controls and updates the selected interface", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter initialEntries={["/ports"]}>
        <LiveStrip />
      </MemoryRouter>,
    );

    expect(screen.getByText("CPU")).toBeInTheDocument();
    expect(screen.getByLabelText("Select the network interface shown in the live strip")).toHaveValue("auto");
    expect(screen.getAllByText("Ethernet").length).toBeGreaterThan(0);

    await user.selectOptions(screen.getByLabelText("Select the network interface shown in the live strip"), "Wi-Fi");

    expect(screen.getByLabelText("Select the network interface shown in the live strip")).toHaveValue("Wi-Fi");
    expect(screen.getAllByText("Wi-Fi").length).toBeGreaterThan(0);
  });
});
