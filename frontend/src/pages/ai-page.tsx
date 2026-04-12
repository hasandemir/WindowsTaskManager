import { zodResolver } from "@hookform/resolvers/zod";
import { Bot, LoaderCircle, ShieldCheck, Sparkles } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { z } from "zod";
import { EmptyState } from "../components/shared/empty-state";
import { PageHeader } from "../components/shared/page-header";
import { PageSkeleton } from "../components/shared/page-skeleton";
import { queueRuleDraftPrefill } from "./rules-page";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { useAIAnalyzeMutation, useAIChatMutation, useAIExecuteMutation, useAIStatusQuery } from "../lib/api-client";
import type { AISuggestion } from "../types/api";

const chatSchema = z.object({
  message: z.string().min(1, "Message is required").max(1500, "Keep it under 1500 characters"),
});

const promptSuggestions = [
  "Which process looks most suspicious right now?",
  "Summarize the biggest CPU and memory hotspots.",
  "What should I investigate first from the active alerts?",
];

type ChatFormValues = z.infer<typeof chatSchema>;

export function AIPage() {
  const navigate = useNavigate();
  const { data, isLoading } = useAIStatusQuery();
  const chatMutation = useAIChatMutation();
  const analyzeMutation = useAIAnalyzeMutation();
  const executeMutation = useAIExecuteMutation();
  const [transcript, setTranscript] = useState<Array<{ role: "user" | "assistant"; text: string }>>([]);
  const [suggestions, setSuggestions] = useState<AISuggestion[]>([]);
  const form = useForm<ChatFormValues>({
    resolver: zodResolver(chatSchema),
    defaultValues: { message: "" },
    mode: "onChange",
  });

  const onSubmit = form.handleSubmit(async (values) => {
    setTranscript((current) => [...current, { role: "user", text: values.message }]);
    form.reset();
    const result = await chatMutation.mutateAsync(values.message);
    setTranscript((current) => [...current, { role: "assistant", text: result.answer }]);
    setSuggestions(result.actions ?? []);
  });

  if (isLoading) {
    return <PageSkeleton />;
  }

  if (!data) {
    return <EmptyState icon={Bot} title="Waiting for AI status" description="WTM is checking whether the local AI advisor is configured and reachable." />;
  }

  return (
    <div className="space-y-6">
      <PageHeader title="AI Advisor" description="Chat with the local advisor, inspect suggestions, and approve actions deliberately." />

      <div className="grid gap-4 sm:grid-cols-3">
        <StatusCard label="AI status" value={data.enabled ? "Enabled" : "Disabled"} variant={data.enabled ? "success" : "warning"} />
        <StatusCard
          label="Provider"
          value={data.provider ? `${data.provider}${data.model ? ` / ${data.model}` : ""}` : data.configured ? "Configured" : "Not configured"}
          variant={data.configured ? "info" : "warning"}
        />
        <StatusCard label="Messages" value={String(transcript.length)} variant={transcript.length > 0 ? "info" : "neutral"} />
      </div>

      <Card className="space-y-4">
        <div className="flex items-center gap-3">
          <div className="rounded-2xl bg-accent-muted p-3 text-accent">
            <Sparkles className="h-5 w-5" />
          </div>
          <div>
            <div className="font-semibold text-foreground">{data.enabled ? "AI enabled" : "AI disabled"}</div>
            <div className="text-sm text-secondary">{data.configured ? "Provider configured" : "Provider not configured"}</div>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border bg-background px-4 py-4">
          <div>
            <div className="text-sm font-semibold text-foreground">Provider and integration settings are in Settings again.</div>
            <div className="text-sm text-secondary">Use the dedicated settings page for provider, endpoint, API key, Telegram bot, and dashboard runtime controls.</div>
          </div>
          <Button type="button" variant="secondary" onClick={() => navigate("/settings")}>
            Open settings
          </Button>
        </div>

        <div className="flex flex-wrap gap-2">
          {promptSuggestions.map((prompt) => (
            <Button
              key={prompt}
              type="button"
              size="sm"
              variant="secondary"
              onClick={() => form.setValue("message", prompt, { shouldDirty: true, shouldTouch: true, shouldValidate: true })}
            >
              {prompt}
            </Button>
          ))}
        </div>

        <div className="flex flex-wrap items-center gap-3 rounded-2xl border border-border bg-background px-4 py-4">
          <div className="min-w-0 flex-1">
            <div className="text-sm font-semibold text-foreground">Action extraction is back.</div>
            <div className="text-sm text-secondary">Chat or analyze can propose `protect`, `ignore`, `add_rule`, `kill`, or `suspend`. Nothing runs until you approve it.</div>
          </div>
          <Button
            type="button"
            variant="secondary"
            disabled={analyzeMutation.isPending || !data.enabled}
            onClick={form.handleSubmit(async (values) => {
              const result = await analyzeMutation.mutateAsync(values.message);
              setTranscript((current) => [...current, { role: "assistant", text: result.answer }]);
              setSuggestions(result.actions ?? []);
            })}
          >
            {analyzeMutation.isPending ? "Analyzing..." : "Analyze snapshot"}
          </Button>
        </div>

        <div className="space-y-3">
          {transcript.length === 0 ? (
            <EmptyState icon={Bot} title="No conversation yet" description="Ask about noisy processes, spikes, or the safest next step." />
          ) : (
            transcript.map((entry, index) => (
              <div
                key={`${entry.role}-${index}`}
                className={entry.role === "user" ? "ml-auto max-w-3xl rounded-2xl bg-accent px-4 py-3 text-accent-foreground" : "mr-auto max-w-3xl rounded-2xl border border-border bg-background px-4 py-3 text-foreground"}
              >
                <div className="mb-1 text-xs font-semibold uppercase tracking-[0.18em] opacity-75">{entry.role}</div>
                <div className="text-sm leading-relaxed">{entry.text}</div>
              </div>
            ))
          )}
        </div>

        <div className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="text-sm font-semibold text-foreground">Suggested actions</div>
              <div className="text-sm text-secondary">Approve only the ones that make sense. Backend safety checks still apply.</div>
            </div>
            <Badge variant={suggestions.length > 0 ? "info" : "neutral"}>{suggestions.length} queued</Badge>
          </div>
          {suggestions.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-border bg-background px-4 py-4 text-sm text-secondary">
              No structured actions yet. Ask the advisor for a concrete mitigation plan or click Analyze snapshot.
            </div>
          ) : (
            <div className="space-y-3">
              {suggestions.map((suggestion) => (
                <ActionCard
                  key={suggestion.id}
                  suggestion={suggestion}
                  isPending={executeMutation.isPending}
                  onApprove={() => executeMutation.mutate(suggestion)}
                  onOpenInRules={() => {
                    if (!suggestion.rule) {
                      return;
                    }
                    queueRuleDraftPrefill({
                      name: suggestion.rule.name,
                      enabled: suggestion.rule.enabled ?? true,
                      match: suggestion.rule.match,
                      metric: suggestion.rule.metric,
                      op: suggestion.rule.op,
                      threshold: suggestion.rule.threshold,
                      for_seconds: suggestion.rule.for_seconds ?? suggestion.rule.for ?? 30,
                      action: suggestion.rule.action,
                      cooldown_seconds: suggestion.rule.cooldown_seconds ?? suggestion.rule.cooldown ?? 300,
                    });
                    navigate("/rules");
                  }}
                />
              ))}
            </div>
          )}
        </div>

        <form className="space-y-3" onSubmit={onSubmit}>
          <label className="block text-sm font-medium text-foreground" htmlFor="ai-message">
            Message
          </label>
          <textarea
            id="ai-message"
            className="min-h-32 w-full rounded-2xl border border-border bg-background px-4 py-3 text-sm text-foreground focus-visible:ring-2 focus-visible:ring-[var(--ring)] focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            placeholder="Ask the local advisor to summarize suspicious behavior or recommend your next step."
            {...form.register("message")}
          />
          {form.formState.errors.message ? <p className="text-sm text-error">{form.formState.errors.message.message}</p> : null}
          <div className="flex flex-wrap items-center gap-3">
            <Button disabled={chatMutation.isPending || !form.formState.isValid || !data.enabled} type="submit">
              {chatMutation.isPending ? (
                <>
                  <LoaderCircle className="mr-2 h-4 w-4 animate-spin motion-reduce:animate-none" />
                  Sending...
                </>
              ) : (
                "Send chat"
              )}
            </Button>
            {!data.enabled ? <Badge variant="warning">Enable AI in the backend config to chat.</Badge> : null}
          </div>
        </form>
      </Card>
    </div>
  );
}

interface ActionCardProps {
  suggestion: AISuggestion;
  isPending: boolean;
  onApprove: () => void;
  onOpenInRules: () => void;
}

function ActionCard({ suggestion, isPending, onApprove, onOpenInRules }: ActionCardProps) {
  const target =
    suggestion.type === "add_rule"
      ? suggestion.rule?.name || "rule"
      : suggestion.name || (suggestion.pid ? `PID ${suggestion.pid}` : "target");

  return (
    <div className="rounded-2xl border border-border bg-background px-4 py-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={suggestion.type === "kill" || suggestion.type === "suspend" ? "warning" : "info"}>{suggestion.type}</Badge>
            <span className="text-sm font-semibold text-foreground">{target}</span>
          </div>
          {suggestion.reason ? <div className="mt-2 text-sm text-secondary">{suggestion.reason}</div> : null}
          {suggestion.rule ? (
            <div className="mt-2 text-xs text-secondary">
              {suggestion.rule.match} {suggestion.rule.metric} {suggestion.rule.op} {suggestion.rule.threshold} {"->"} {suggestion.rule.action}
            </div>
          ) : null}
        </div>
        <Button type="button" size="sm" disabled={isPending} onClick={onApprove}>
          <ShieldCheck className="mr-2 h-4 w-4" />
          Approve
        </Button>
        {suggestion.type === "add_rule" ? (
          <Button type="button" size="sm" variant="secondary" onClick={onOpenInRules}>
            Open in Rules
          </Button>
        ) : null}
      </div>
    </div>
  );
}

interface StatusCardProps {
  label: string;
  value: string;
  variant: "neutral" | "info" | "success" | "warning";
}

function StatusCard({ label, value, variant }: StatusCardProps) {
  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</div>
          <div className="mt-3 text-2xl font-semibold tracking-tight text-foreground">{value}</div>
        </div>
        <Badge variant={variant}>{label}</Badge>
      </div>
    </Card>
  );
}
