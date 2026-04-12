import { useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Pencil, Plus, Save, Trash2, Workflow, X } from "lucide-react";
import { ConfirmDialog } from "../components/shared/confirm-dialog";
import { DetailTile, SummaryCard } from "../components/shared/detail-tile";
import { EmptyState } from "../components/shared/empty-state";
import { PageHeader } from "../components/shared/page-header";
import { PageSkeleton } from "../components/shared/page-skeleton";
import { SearchInput } from "../components/shared/search-input";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { useDebouncedValue } from "../hooks/use-debounced-value";
import { useRulesQuery, useRulesUpdateMutation } from "../lib/api-client";
import { formatBytes } from "../lib/format";
import type { Rule } from "../types/api";

const prefillStorageKey = "wtm:rule-draft-prefill";
const defaultRuleDraft: Rule = {
  name: "",
  enabled: true,
  match: "",
  metric: "cpu_percent",
  op: ">=",
  threshold: 80,
  for_seconds: 30,
  action: "alert",
  cooldown_seconds: 300,
};

const ruleTemplates: Array<{ label: string; description: string; draft: Rule }> = [
  {
    label: "CPU guard",
    description: "Alert when one process holds CPU too long.",
    draft: {
      ...defaultRuleDraft,
      name: "High CPU sustained",
      match: "chrome.exe",
      metric: "cpu_percent",
      op: ">=",
      threshold: 80,
      for_seconds: 45,
      cooldown_seconds: 300,
      action: "alert",
    },
  },
  {
    label: "Memory guard",
    description: "Catch memory growth before the box feels bad.",
    draft: {
      ...defaultRuleDraft,
      name: "Memory leak watch",
      match: "claude.exe",
      metric: "memory_bytes",
      op: ">=",
      threshold: 2 * 1024 * 1024 * 1024,
      for_seconds: 60,
      cooldown_seconds: 600,
      action: "alert",
    },
  },
  {
    label: "Thread storm",
    description: "Detect runaway worker creation.",
    draft: {
      ...defaultRuleDraft,
      name: "Thread explosion",
      match: ".exe",
      metric: "thread_count",
      op: ">=",
      threshold: 300,
      for_seconds: 20,
      cooldown_seconds: 300,
      action: "suspend",
    },
  },
];

export function RulesPage() {
  const { data, isLoading } = useRulesQuery();
  const updateRulesMutation = useRulesUpdateMutation();
  const [searchValue, setSearchValue] = useState("");
  const [draft, setDraft] = useState<Rule>(defaultRuleDraft);
  const [draftError, setDraftError] = useState("");
  const [editingRuleName, setEditingRuleName] = useState<string | null>(null);
  const [editingDraft, setEditingDraft] = useState<Rule | null>(null);
  const [editingError, setEditingError] = useState("");
  const [deleteRuleName, setDeleteRuleName] = useState<string | null>(null);
  const debouncedSearch = useDebouncedValue(searchValue, 300);
  const rules = data?.rules ?? [];

  const filteredRules = useMemo(() => {
    const needle = debouncedSearch.trim().toLowerCase();
    if (!needle) {
      return rules;
    }
    return rules.filter((rule) => {
      return (
        rule.name.toLowerCase().includes(needle) ||
        rule.match.toLowerCase().includes(needle) ||
        rule.metric.toLowerCase().includes(needle) ||
        rule.action.toLowerCase().includes(needle)
      );
    });
  }, [debouncedSearch, rules]);

  useEffect(() => {
    const raw = window.sessionStorage.getItem(prefillStorageKey);
    if (!raw) {
      return;
    }
    try {
      const parsed = JSON.parse(raw) as Partial<Rule>;
      setDraft(normalizeRuleDraft({ ...defaultRuleDraft, ...parsed }));
      setDraftError("");
    } finally {
      window.sessionStorage.removeItem(prefillStorageKey);
    }
  }, []);

  if (isLoading) {
    return <PageSkeleton />;
  }

  const enabledRules = rules.filter((rule) => rule.enabled).length;
  const actionRules = rules.filter((rule) => rule.action !== "alert").length;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Rules"
        description="Create, review, and tune backend automation rules without losing control of what they do."
        eyebrow="Automation"
        icon={Workflow}
        meta={
          <>
            <Badge variant="info">{rules.length} total rules</Badge>
            <Badge variant={enabledRules > 0 ? "success" : "neutral"}>{enabledRules} enabled</Badge>
          </>
        }
        actions={
          <SearchInput
            ariaLabel="Search rules by name, match, metric, or action"
            placeholder="Search rules"
            value={searchValue}
            onChange={setSearchValue}
          />
        }
      />

      <div className="grid gap-4 sm:grid-cols-3">
        <SummaryCard label="Rules" value={String(rules.length)} accent={<Badge variant="info">Configured</Badge>} />
        <SummaryCard label="Enabled" value={String(enabledRules)} accent={<Badge variant={enabledRules > 0 ? "success" : "neutral"}>Live</Badge>} />
        <SummaryCard label="Actions beyond alerts" value={String(actionRules)} accent={<Badge variant="warning">Suspend / kill</Badge>} />
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.45fr)_22rem]">
        <Card className="space-y-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold tracking-tight text-foreground">Add rule</h2>
              <p className="text-sm text-secondary">Define what to watch, how long it must hold, and what WTM should do when it trips.</p>
            </div>
            <Badge variant="info">Automation</Badge>
          </div>

          <div className="grid gap-3 lg:grid-cols-3">
            {ruleTemplates.map((template) => (
              <button
                key={template.label}
                type="button"
                className="rounded-lg border border-border bg-background px-4 py-3 text-left transition-colors hover:bg-background-muted"
                onClick={() => {
                  setDraft(template.draft);
                  setDraftError("");
                }}
              >
                <div className="text-sm font-semibold text-foreground">{template.label}</div>
                <div className="mt-2 text-sm text-secondary">{template.description}</div>
              </button>
            ))}
          </div>

          <RuleForm draft={draft} onChange={setDraft} />

          <div className="grid gap-3 sm:grid-cols-2">
            <RulePreview draft={draft} />
            <div className="soft-panel">
              <div className="metric-label">What happens</div>
              <div className="mt-2 text-sm leading-relaxed text-secondary">
                Alert-only rules are safest. Suspend and kill are stronger actions and are best for high-confidence, noisy patterns.
              </div>
            </div>
          </div>

          {draftError ? <ErrorBanner>{draftError}</ErrorBanner> : null}

          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="button"
              disabled={updateRulesMutation.isPending}
              onClick={() => {
                const error = validateRuleDraft(draft, rules);
                if (error) {
                  setDraftError(error);
                  return;
                }

                setDraftError("");
                updateRulesMutation.mutate([...rules, normalizeRuleDraft(draft)], {
                  onSuccess: () => {
                    setDraft(defaultRuleDraft);
                  },
                });
              }}
            >
              <Plus className="mr-2 h-4 w-4" />
              Add rule
            </Button>
            <span className="text-sm text-secondary">Rules are saved to the backend config immediately.</span>
          </div>
        </Card>

        <Card className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="section-title">Operator guide</h2>
            <Badge variant="neutral">Tips</Badge>
          </div>
          <GuideRow title="Start broad, then tighten" description="Match on a process family first, then narrow thresholds after you observe a few cycles." />
          <GuideRow title="Use cooldowns" description="Cooldown protects you from repeated triggers and keeps alerts or actions from thrashing." />
          <GuideRow title="Prefer alert before action" description="If you are not fully sure, ship an alert-only rule first and promote it later." />
        </Card>
      </div>

      {filteredRules.length === 0 ? (
        <EmptyState
          icon={Workflow}
          title={rules.length === 0 ? "No rules defined" : "No rules match"}
          description={rules.length === 0 ? "Add your first automation rule above." : "Try a different rule, match, metric, or action filter."}
        />
      ) : null}

      <div className="grid gap-4">
        {filteredRules.map((rule) => {
          const isEditing = editingRuleName === rule.name && editingDraft;

          return (
            <Card key={rule.name} className="space-y-4">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="eyebrow">Saved rule</div>
                  <h2 className="mt-2 text-lg font-semibold tracking-tight text-foreground">{rule.name}</h2>
                  <p className="mt-1 text-sm text-secondary">{describeRule(rule)}</p>
                </div>
                <div className="flex flex-wrap items-center justify-end gap-2">
                  <Badge variant={rule.enabled ? "success" : "neutral"}>{rule.enabled ? "Enabled" : "Disabled"}</Badge>
                  {!isEditing ? (
                    <>
                      <Button
                        type="button"
                        size="sm"
                        variant="secondary"
                        disabled={updateRulesMutation.isPending}
                        onClick={() => {
                          setEditingRuleName(rule.name);
                          setEditingDraft({ ...rule });
                          setEditingError("");
                        }}
                      >
                        <Pencil className="mr-2 h-4 w-4" />
                        Edit
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="secondary"
                        disabled={updateRulesMutation.isPending}
                        onClick={() =>
                          updateRulesMutation.mutate(rules.map((entry) => (entry.name === rule.name ? { ...entry, enabled: !entry.enabled } : entry)))
                        }
                      >
                        {rule.enabled ? "Disable" : "Enable"}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="secondary"
                        disabled={updateRulesMutation.isPending}
                        onClick={() => setDeleteRuleName(rule.name)}
                      >
                        <Trash2 className="mr-2 h-4 w-4" />
                        Delete
                      </Button>
                    </>
                  ) : (
                    <>
                      <Button
                        type="button"
                        size="sm"
                        disabled={updateRulesMutation.isPending}
                        onClick={() => {
                          const error = validateRuleDraft(editingDraft, rules, rule.name);
                          if (error) {
                            setEditingError(error);
                            return;
                          }

                          setEditingError("");
                          updateRulesMutation.mutate(
                            rules.map((entry) => (entry.name === rule.name ? normalizeRuleDraft(editingDraft) : entry)),
                            {
                              onSuccess: () => {
                                setEditingRuleName(null);
                                setEditingDraft(null);
                              },
                            },
                          );
                        }}
                      >
                        <Save className="mr-2 h-4 w-4" />
                        Save
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="secondary"
                        onClick={() => {
                          setEditingRuleName(null);
                          setEditingDraft(null);
                          setEditingError("");
                        }}
                      >
                        <X className="mr-2 h-4 w-4" />
                        Cancel
                      </Button>
                    </>
                  )}
                </div>
              </div>

              {isEditing ? (
                <>
                  <RuleForm draft={editingDraft} onChange={setEditingDraft} />
                  <RulePreview draft={editingDraft} />
                  {editingError ? <ErrorBanner>{editingError}</ErrorBanner> : null}
                </>
              ) : (
                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <DetailTile label="Match" value={rule.match} valueClassName="break-words" />
                  <DetailTile label="Metric" value={formatRuleMetric(rule)} valueClassName="break-words" />
                  <DetailTile label="Action" value={rule.action} valueClassName="break-words" />
                  <DetailTile label="Window" value={`${rule.for_seconds}s for / ${rule.cooldown_seconds}s cooldown`} valueClassName="break-words" />
                </div>
              )}
            </Card>
          );
        })}
      </div>

      <ConfirmDialog
        open={deleteRuleName !== null}
        title="Delete rule?"
        description={
          deleteRuleName
            ? `This will remove "${deleteRuleName}" from the backend config immediately.`
            : "This will remove the selected rule from the backend config immediately."
        }
        confirmLabel="Delete rule"
        isPending={updateRulesMutation.isPending}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteRuleName(null);
          }
        }}
        onConfirm={() => {
          if (!deleteRuleName) {
            return;
          }
          updateRulesMutation.mutate(rules.filter((entry) => entry.name !== deleteRuleName), {
            onSuccess: () => {
              setDeleteRuleName(null);
            },
          });
        }}
      />
    </div>
  );
}

interface RuleFormProps {
  draft: Rule;
  onChange: (value: Rule) => void;
}

function RuleForm({ draft, onChange }: RuleFormProps) {
  const isByteMetric = draft.metric === "memory_bytes" || draft.metric === "private_bytes";
  const [thresholdUnit, setThresholdUnit] = useState<ThresholdUnit>(() => detectThresholdUnit(draft.metric, draft.threshold));

  useEffect(() => {
    setThresholdUnit((current) => {
      if (draft.metric !== "memory_bytes" && draft.metric !== "private_bytes") {
        return "raw";
      }
      return current === "raw" ? detectThresholdUnit(draft.metric, draft.threshold) : current;
    });
  }, [draft.metric, draft.threshold]);

  return (
    <>
      <div className="grid gap-4 lg:grid-cols-2">
        <Field label="Rule name">
          <input
            aria-label="Rule name"
            className="form-control w-full"
            placeholder="High CPU browser tabs"
            value={draft.name}
            onChange={(event) => onChange({ ...draft, name: event.target.value })}
          />
        </Field>
        <Field label="Rule match">
          <input
            aria-label="Rule match"
            className="form-control w-full"
            placeholder="chrome.exe or .exe"
            value={draft.match}
            onChange={(event) => onChange({ ...draft, match: event.target.value })}
          />
        </Field>
        <Field label="Rule metric">
          <select
            aria-label="Rule metric"
            className="form-control w-full"
            value={draft.metric}
            onChange={(event) => onChange({ ...draft, metric: event.target.value })}
          >
            <option value="cpu_percent">CPU percent</option>
            <option value="memory_bytes">Memory bytes</option>
            <option value="private_bytes">Private bytes</option>
            <option value="thread_count">Thread count</option>
          </select>
        </Field>
        <Field label="Rule operator">
          <select
            aria-label="Rule operator"
            className="form-control w-full"
            value={draft.op}
            onChange={(event) => onChange({ ...draft, op: event.target.value })}
          >
            <option value=">=">{">="}</option>
            <option value=">">{">"}</option>
            <option value="<=">{"<="}</option>
            <option value="<">{"<"}</option>
          </select>
        </Field>
        <Field label="Rule threshold">
          {isByteMetric ? (
            <div className="flex gap-2">
              <input
                aria-label="Rule threshold"
                type="number"
                min={0}
                step="any"
                className="form-control min-w-0 flex-1"
                value={formatThresholdForInput(draft.threshold, thresholdUnit)}
                onChange={(event) => onChange({ ...draft, threshold: parseThresholdInput(event.target.value, thresholdUnit) })}
              />
              <select
                aria-label="Rule threshold unit"
                className="form-control"
                value={thresholdUnit}
                onChange={(event) => setThresholdUnit(event.target.value as ThresholdUnit)}
              >
                <option value="mb">MB</option>
                <option value="gb">GB</option>
                <option value="tb">TB</option>
                <option value="raw">Bytes</option>
              </select>
            </div>
          ) : (
            <input
              aria-label="Rule threshold"
              type="number"
              className="form-control w-full"
              value={draft.threshold}
              onChange={(event) => onChange({ ...draft, threshold: Number(event.target.value) || 0 })}
            />
          )}
        </Field>
        <Field label="Rule action">
          <select
            aria-label="Rule action"
            className="form-control w-full"
            value={draft.action}
            onChange={(event) => onChange({ ...draft, action: event.target.value })}
          >
            <option value="alert">Alert only</option>
            <option value="suspend">Suspend process</option>
            <option value="kill">Kill process</option>
          </select>
        </Field>
        <Field label="Rule for seconds">
          <input
            aria-label="Rule for seconds"
            type="number"
            min={0}
            max={86400}
            className="form-control w-full"
            value={draft.for_seconds}
            onChange={(event) => onChange({ ...draft, for_seconds: Number(event.target.value) || 0 })}
          />
        </Field>
        <Field label="Rule cooldown seconds">
          <input
            aria-label="Rule cooldown seconds"
            type="number"
            min={0}
            className="form-control w-full"
            value={draft.cooldown_seconds}
            onChange={(event) => onChange({ ...draft, cooldown_seconds: Number(event.target.value) || 0 })}
          />
        </Field>
      </div>

      <label className="flex min-h-10 items-center justify-between rounded-lg border border-border bg-background px-4 py-3 text-sm text-foreground">
        <span>Rule enabled immediately</span>
        <input aria-label="Rule enabled" type="checkbox" checked={draft.enabled} onChange={(event) => onChange({ ...draft, enabled: event.target.checked })} />
      </label>
    </>
  );
}

interface FieldProps {
  label: string;
  children: ReactNode;
}

function Field({ label, children }: FieldProps) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium text-foreground">{label}</span>
      {children}
    </label>
  );
}

function RulePreview({ draft }: { draft: Rule }) {
  return (
    <div className="soft-panel">
      <div className="metric-label">Preview</div>
      <div className="mt-2 text-sm text-foreground">{describeRule(draft)}</div>
    </div>
  );
}

function ErrorBanner({ children }: { children: ReactNode }) {
  return <div className="rounded-lg border border-error bg-[color:var(--error-bg)] px-4 py-3 text-sm text-error">{children}</div>;
}

interface GuideRowProps {
  title: string;
  description: string;
}

function GuideRow({ title, description }: GuideRowProps) {
  return (
    <div className="soft-panel">
      <div className="text-sm font-semibold text-foreground">{title}</div>
      <div className="mt-1 text-sm leading-relaxed text-secondary">{description}</div>
    </div>
  );
}

function validateRuleDraft(draft: Rule, existingRules: Rule[], originalName?: string) {
  const name = draft.name.trim();
  const match = draft.match.trim();

  if (!name) {
    return "Rule name is required.";
  }
  if (!match) {
    return "Process match is required.";
  }
  if (!Number.isFinite(draft.threshold)) {
    return "Threshold must be a valid number.";
  }
  if (draft.threshold < 0) {
    return "Threshold cannot be negative.";
  }
  if (draft.for_seconds < 0 || draft.for_seconds > 86400) {
    return "For seconds must stay between 0 and 86400.";
  }
  if (draft.cooldown_seconds < 0) {
    return "Cooldown seconds cannot be negative.";
  }
  if (existingRules.some((rule) => rule.name.toLowerCase() === name.toLowerCase() && rule.name !== originalName)) {
    return "A rule with that name already exists.";
  }

  return "";
}

function normalizeRuleDraft(draft: Rule): Rule {
  return {
    ...draft,
    name: draft.name.trim(),
    match: draft.match.trim(),
  };
}

type ThresholdUnit = "raw" | "mb" | "gb" | "tb";

function detectThresholdUnit(metric: string, threshold: number): ThresholdUnit {
  if (metric !== "memory_bytes" && metric !== "private_bytes") {
    return "raw";
  }
  if (threshold >= 1024 ** 4) {
    return "tb";
  }
  if (threshold >= 1024 ** 3) {
    return "gb";
  }
  return "mb";
}

function thresholdDivisor(unit: ThresholdUnit) {
  switch (unit) {
    case "mb":
      return 1024 ** 2;
    case "gb":
      return 1024 ** 3;
    case "tb":
      return 1024 ** 4;
    default:
      return 1;
  }
}

function formatThresholdForInput(value: number, unit: ThresholdUnit) {
  if (!Number.isFinite(value) || value <= 0) {
    return "0";
  }
  const converted = value / thresholdDivisor(unit);
  return converted >= 10 ? converted.toFixed(0) : converted.toFixed(1);
}

function parseThresholdInput(raw: string, unit: ThresholdUnit) {
  const parsed = Number(raw);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 0;
  }
  return parsed * thresholdDivisor(unit);
}

export function queueRuleDraftPrefill(rule: Partial<Rule>) {
  window.sessionStorage.setItem(prefillStorageKey, JSON.stringify(rule));
}

function describeRule(rule: Rule) {
  const actionLabel =
    rule.action === "kill"
      ? "kill the process"
      : rule.action === "suspend"
        ? "suspend the process"
        : "raise an alert";

  return `When "${rule.match}" has ${formatRuleMetric(rule)} for ${rule.for_seconds}s, ${actionLabel}. Cooldown ${rule.cooldown_seconds}s.`;
}

function formatRuleMetric(rule: Rule) {
  const threshold =
    rule.metric === "memory_bytes" || rule.metric === "private_bytes" ? formatBytes(rule.threshold) : String(rule.threshold);
  const metricLabel =
    rule.metric === "cpu_percent"
      ? "CPU"
      : rule.metric === "memory_bytes"
        ? "memory"
        : rule.metric === "private_bytes"
          ? "private bytes"
          : "thread count";

  return `${metricLabel} ${rule.op} ${threshold}`;
}
