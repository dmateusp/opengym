import { useParams, useNavigate } from "react-router-dom";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import { ArrowLeft, Loader2, CheckCircle2, XCircle } from "lucide-react";
import { API_BASE_URL, redirectToLogin } from "@/lib/api";
import { fetchWithDemoRecovery } from "@/lib/fetchWithDemoRecovery";
import { TimeDisplay } from "@/components/ui/TimeDisplay";
import UserProfileMenu from "@/components/auth/UserProfileMenu";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import type { GameReimbursementEntry, User } from "@/opengym/client";
import { useCurrentUser } from "@/hooks/useCurrentUser";

const getInitials = (name?: string | null, email?: string) => {
  if (name) {
    const parts = name.split(" ").filter((p) => p.length > 0);
    if (parts.length > 0) {
      return parts
        .slice(0, 3)
        .map((p) => p[0].toUpperCase())
        .join("");
    }
    return name.slice(0, 2).toUpperCase();
  }
  if (email) {
    return email.slice(0, 2).toUpperCase();
  }
  return "??";
};

const formatCents = (amountCents: number) => (amountCents / 100).toFixed(2);

export default function ReimbursementsPage() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { user, setUser } = useCurrentUser();

  const [entries, setEntries] = useState<GameReimbursementEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isFreezeRequiredError, setIsFreezeRequiredError] = useState(false);
  const [isOrganizerOnlyError, setIsOrganizerOnlyError] = useState(false);
  const [updating, setUpdating] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;

    const fetchReimbursements = async () => {
      try {
        setIsLoading(true);
        setError(null);
        setIsFreezeRequiredError(false);
        setIsOrganizerOnlyError(false);
        const resp = await fetchWithDemoRecovery(
          `${API_BASE_URL}/api/games/${id}/reimbursements`,
          { credentials: "include" }
        );
        if (resp.status === 401) {
          redirectToLogin();
          return;
        }
        if (!resp.ok) {
          if (resp.status === 403) {
            setIsOrganizerOnlyError(true);
            setError(t("reimbursements.organizerOnly"));
            return;
          }
          const errorText = (await resp.text()).toLowerCase();
          if (resp.status === 400 && errorText.includes("frozen")) {
            setIsFreezeRequiredError(true);
            setError(t("reimbursements.freezeRequired"));
            return;
          }
          setError(t("reimbursements.failedToLoad"));
          return;
        }
        const data: GameReimbursementEntry[] = await resp.json();
        setEntries(data);
      } catch {
        setError(t("reimbursements.failedToLoad"));
      } finally {
        setIsLoading(false);
      }
    };

    fetchReimbursements();
  }, [id, t]);

  const handleSetReceivedAt = async (participantId: string, value: string | null) => {
    if (!id) return;
    setUpdating(participantId);
    try {
      setError(null);
      setIsFreezeRequiredError(false);
      setIsOrganizerOnlyError(false);
      const result = await fetchWithDemoRecovery(
        `${API_BASE_URL}/api/games/${id}/reimbursements`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({
            participantId,
            reimbursement_received_at: value,
          }),
        }
      );
      if (result.status === 401) {
        redirectToLogin();
        return;
      }
      if (!result.ok) {
        if (result.status === 403) {
          setIsOrganizerOnlyError(true);
          setError(t("reimbursements.organizerOnly"));
          return;
        }
        const errorText = (await result.text()).toLowerCase();
        if (result.status === 400 && errorText.includes("frozen")) {
          setIsFreezeRequiredError(true);
          setError(t("reimbursements.freezeRequired"));
          return;
        }
        setError(t("reimbursements.failedToUpdate"));
        return;
      }

      // Refresh entries
      const resp = await fetchWithDemoRecovery(
        `${API_BASE_URL}/api/games/${id}/reimbursements`,
        { credentials: "include" }
      );
      if (resp.ok) {
        setEntries(await resp.json());
      }
    } catch {
      setError(t("reimbursements.failedToUpdate"));
    } finally {
      setUpdating(null);
    }
  };

  const handleUserChange = (updatedUser: User | null) => {
    setUser(updatedUser);
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50">
      <div className="container mx-auto px-4 py-8 max-w-4xl">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <Button
            variant="ghost"
            onClick={() => navigate(`/games/${id}`)}
            className="text-gray-600 hover:text-gray-900"
          >
            <ArrowLeft className="mr-2 h-5 w-5" />
            {t("reimbursements.backToGame")}
          </Button>
          <div className="flex items-center gap-2">
            <LanguageSwitcher />
            <UserProfileMenu user={user} onUserChange={handleUserChange} />
          </div>
        </div>

        <Card className="overflow-hidden border-l-8 border-l-primary">
          <div className="p-8">
            <h1 className="text-2xl font-bold text-gray-900 mb-6">
              {t("reimbursements.title")}
            </h1>

            {isLoading ? (
              <div className="flex justify-center py-12">
                <Loader2 className="h-8 w-8 animate-spin text-primary" />
              </div>
            ) : error ? (
              isFreezeRequiredError ? (
                <div className="bg-yellow-50 border border-yellow-200 rounded-xl p-6 text-yellow-900">
                  <p className="font-semibold text-base mb-2">
                    {t("reimbursements.freezeRequired")}
                  </p>
                  <p className="text-sm text-yellow-800 mb-4">
                    {t("reimbursements.freezeRequiredDescription")}
                  </p>
                  <Button variant="outline" onClick={() => navigate(`/games/${id}`)}>
                    {t("reimbursements.backToGame")}
                  </Button>
                </div>
              ) : isOrganizerOnlyError ? (
                <div className="bg-blue-50 border border-blue-200 rounded-xl p-6 text-blue-900">
                  <p className="font-semibold text-base mb-2">
                    {t("reimbursements.organizerOnly")}
                  </p>
                  <p className="text-sm text-blue-800 mb-4">
                    {t("reimbursements.organizerOnlyDescription")}
                  </p>
                  <Button variant="outline" onClick={() => navigate(`/games/${id}`)}>
                    {t("reimbursements.backToGame")}
                  </Button>
                </div>
              ) : (
                <div className="bg-red-50 border border-red-200 rounded-xl p-4 text-red-700 text-sm">
                  {error}
                </div>
              )
            ) : entries.length === 0 ? (
              <p className="text-gray-500 text-center py-12">
                {t("reimbursements.noParticipants")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 text-left text-gray-500 text-xs uppercase tracking-wider">
                      <th className="pb-3 pr-4">{t("reimbursements.reference")}</th>
                      <th className="pb-3 pr-4 text-right">{t("reimbursements.amountOwed")}</th>
                      <th className="pb-3 pr-4">{t("reimbursements.participant")}</th>
                      <th className="pb-3 pr-4">{t("reimbursements.sentAt")}</th>
                      <th className="pb-3 pr-4">{t("reimbursements.receivedAt")}</th>
                      <th className="pb-3">{t("reimbursements.actions")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {entries.map((entry) => {
                      const p = entry.participant;
                      const isUpdating = updating === p.id;
                      const hasReceived = !!entry.reimbursementReceivedAt;
                      return (
                        <tr
                          key={p.id}
                          className="border-b border-gray-100 last:border-0"
                        >
                          <td className="py-4 pr-4">
                            <span className="inline-flex min-w-12 justify-center rounded-md bg-gray-100 px-2 py-1 font-mono text-xs font-semibold text-gray-700">
                              {entry.reimbursementReference || "----"}
                            </span>
                          </td>

                          <td className="py-4 pr-4 text-right">
                            <span className="inline-flex min-w-16 justify-end rounded-md bg-emerald-50 px-2 py-1 font-mono text-xs font-semibold text-emerald-700">
                              {formatCents(entry.amountOwedCents)}
                            </span>
                          </td>

                          {/* Participant */}
                          <td className="py-4 pr-4">
                            <div className="flex items-center gap-3">
                              <Avatar className="h-9 w-9">
                                {p.picture && (
                                  <AvatarImage
                                    src={p.picture}
                                    alt={p.name ?? p.email}
                                  />
                                )}
                                <AvatarFallback className="text-xs bg-primary/10 text-primary">
                                  {getInitials(p.name, p.email)}
                                </AvatarFallback>
                              </Avatar>
                              <div>
                                <div className="flex items-center gap-2">
                                  <p className="font-medium text-gray-900">
                                    {p.name ?? p.email}
                                  </p>
                                  {entry.guests > 0 && (
                                    <span
                                      className="inline-flex items-center justify-center rounded-full bg-secondary px-2 py-0.5 text-[11px] font-bold leading-none text-white"
                                      title={`+${entry.guests}`}
                                    >
                                      +{entry.guests}
                                    </span>
                                  )}
                                </div>
                                {p.name && (
                                  <p className="text-xs text-gray-400">{p.email}</p>
                                )}
                              </div>
                            </div>
                          </td>

                          {/* Sent at (participant-set) */}
                          <td className="py-4 pr-4 text-gray-600">
                            {entry.reimbursedAt ? (
                              <div className="flex items-center gap-1.5 text-success">
                                <CheckCircle2 className="h-4 w-4 flex-shrink-0" />
                                <TimeDisplay
                                  timestamp={entry.reimbursedAt}
                                  displayFormat="relative"
                                />
                              </div>
                            ) : (
                              <div className="flex items-center gap-1.5 text-gray-400">
                                <XCircle className="h-4 w-4 flex-shrink-0" />
                                <span>{t("reimbursements.notSent")}</span>
                              </div>
                            )}
                          </td>

                          {/* Received at (organizer-set) */}
                          <td className="py-4 pr-4 text-gray-600">
                            {entry.reimbursementReceivedAt ? (
                              <div className="flex items-center gap-1.5 text-success">
                                <CheckCircle2 className="h-4 w-4 flex-shrink-0" />
                                <TimeDisplay
                                  timestamp={entry.reimbursementReceivedAt}
                                  displayFormat="relative"
                                />
                              </div>
                            ) : (
                              <div className="flex items-center gap-1.5 text-gray-400">
                                <XCircle className="h-4 w-4 flex-shrink-0" />
                                <span>{t("reimbursements.notReceived")}</span>
                              </div>
                            )}
                          </td>

                          {/* Organizer action */}
                          <td className="py-4">
                            {hasReceived ? (
                              <Button
                                variant="outline"
                                size="sm"
                                disabled={isUpdating}
                                onClick={() =>
                                  handleSetReceivedAt(p.id, null)
                                }
                              >
                                {isUpdating ? (
                                  <Loader2 className="h-3 w-3 animate-spin" />
                                ) : (
                                  t("reimbursements.clearReceived")
                                )}
                              </Button>
                            ) : (
                              <Button
                                size="sm"
                                disabled={isUpdating}
                                onClick={() =>
                                  handleSetReceivedAt(
                                    p.id,
                                    new Date().toISOString()
                                  )
                                }
                              >
                                {isUpdating ? (
                                  <Loader2 className="h-3 w-3 animate-spin" />
                                ) : (
                                  t("reimbursements.markReceived")
                                )}
                              </Button>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}
