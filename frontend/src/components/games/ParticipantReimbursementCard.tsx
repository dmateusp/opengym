import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { CheckCircle2, Clock, Copy, Check, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TimeDisplay } from "@/components/ui/TimeDisplay";
import { API_BASE_URL, redirectToLogin } from "@/lib/api";
import { fetchWithDemoRecovery } from "@/lib/fetchWithDemoRecovery";
import type { ReimbursementRecord } from "@/opengym/client";

interface ParticipantReimbursementCardProps {
  gameId: string;
  participantId: string;
}

export function ParticipantReimbursementCard({
  gameId,
  participantId,
}: ParticipantReimbursementCardProps) {
  const { t } = useTranslation();
  const [record, setRecord] = useState<ReimbursementRecord | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isUpdating, setIsUpdating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refCopied, setRefCopied] = useState(false);

  useEffect(() => {
    const fetchRecord = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const resp = await fetchWithDemoRecovery(
          `${API_BASE_URL}/api/games/${gameId}/reimbursements/${participantId}`,
          { credentials: "include" }
        );
        if (resp.status === 401) {
          redirectToLogin();
          return;
        }
        if (!resp.ok) {
          setError(t("reimbursements.failedToLoadSingle"));
          return;
        }
        setRecord(await resp.json());
      } catch {
        setError(t("reimbursements.failedToLoadSingle"));
      } finally {
        setIsLoading(false);
      }
    };

    fetchRecord();
  }, [gameId, participantId, t]);

  const handleMarkSent = async (value: string | null) => {
    setIsUpdating(true);
    setError(null);
    try {
      const resp = await fetchWithDemoRecovery(
        `${API_BASE_URL}/api/games/${gameId}/reimbursements`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ reimbursedAt: value }),
        }
      );
      if (resp.status === 401) {
        redirectToLogin();
        return;
      }
      if (!resp.ok) {
        setError(t("reimbursements.failedToUpdate"));
        return;
      }
      const updated: ReimbursementRecord = await resp.json();
      setRecord(updated);
    } catch {
      setError(t("reimbursements.failedToUpdate"));
    } finally {
      setIsUpdating(false);
    }
  };

  const handleCopyRef = () => {
    if (!record?.reimbursementReference) return;
    navigator.clipboard.writeText(record.reimbursementReference).then(() => {
      setRefCopied(true);
      setTimeout(() => setRefCopied(false), 2000);
    });
  };

  if (isLoading) {
    return (
      <div className="flex justify-center py-4">
        <Loader2 className="h-5 w-5 animate-spin text-primary" />
      </div>
    );
  }

  if (error && !record) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
        {error}
      </div>
    );
  }

  if (!record) return null;

  const hasSent = !!record.reimbursedAt;
  const hasReceived = !!record.reimbursementReceivedAt;

  return (
    <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 space-y-4">
      {/* Reference */}
      <div>
        <p className="text-xs font-semibold text-emerald-800 uppercase tracking-wide mb-2">
          {t("reimbursements.yourReference")}
        </p>
        <div className="flex items-center gap-3">
          <span className="inline-flex items-center justify-center rounded-lg bg-white border border-emerald-300 px-4 py-2 font-mono text-xl font-bold text-gray-900 tracking-widest">
            {record.reimbursementReference || "----"}
          </span>
          <button
            onClick={handleCopyRef}
            className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-700 hover:text-emerald-900 transition"
          >
            {refCopied ? (
              <>
                <Check className="h-3.5 w-3.5" />
                {t("common.copied")}
              </>
            ) : (
              <>
                <Copy className="h-3.5 w-3.5" />
                {t("common.copy")}
              </>
            )}
          </button>
        </div>
        <p className="mt-2 text-xs text-emerald-700">
          {t("reimbursements.includeReferenceInstruction")}
        </p>
      </div>

      {/* Status rows */}
      <div className="space-y-2 pt-2 border-t border-emerald-200">
        <div className="flex items-center gap-2 text-sm">
          {hasSent ? (
            <CheckCircle2 className="h-4 w-4 text-success flex-shrink-0" />
          ) : (
            <Clock className="h-4 w-4 text-gray-400 flex-shrink-0" />
          )}
          <span className={hasSent ? "text-gray-800" : "text-gray-500"}>
            {hasSent ? (
              <>
                {t("reimbursements.youSentAt")}{" "}
                <TimeDisplay
                  timestamp={record.reimbursedAt!}
                  displayFormat="relative"
                />
              </>
            ) : (
              t("reimbursements.notSentYet")
            )}
          </span>
        </div>

        <div className="flex items-center gap-2 text-sm">
          {hasReceived ? (
            <CheckCircle2 className="h-4 w-4 text-success flex-shrink-0" />
          ) : (
            <Clock className="h-4 w-4 text-gray-400 flex-shrink-0" />
          )}
          <span className={hasReceived ? "text-gray-800" : "text-gray-500"}>
            {hasReceived ? (
              <>
                {t("reimbursements.organizerConfirmedAt")}{" "}
                <TimeDisplay
                  timestamp={record.reimbursementReceivedAt!}
                  displayFormat="relative"
                />
              </>
            ) : (
              t("reimbursements.waitingForConfirmation")
            )}
          </span>
        </div>
      </div>

      {/* Action */}
      <div className="pt-2 border-t border-emerald-200">
        {hasSent ? (
          <Button
            variant="outline"
            size="sm"
            disabled={isUpdating}
            onClick={() => handleMarkSent(null)}
          >
            {isUpdating ? (
              <>
                <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                {t("reimbursements.updating")}
              </>
            ) : (
              t("reimbursements.clearSent")
            )}
          </Button>
        ) : (
          <Button
            size="sm"
            disabled={isUpdating}
            onClick={() => handleMarkSent(new Date().toISOString())}
          >
            {isUpdating ? (
              <>
                <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                {t("reimbursements.updating")}
              </>
            ) : (
              t("reimbursements.markAsSent")
            )}
          </Button>
        )}
        {error && <p className="mt-2 text-xs text-red-600">{error}</p>}
      </div>
    </div>
  );
}
