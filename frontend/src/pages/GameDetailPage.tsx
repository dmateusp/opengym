import { useParams, useNavigate } from "react-router-dom";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { API_BASE_URL, redirectToLogin } from "@/lib/api";
import { fetchWithDemoRecovery } from "@/lib/fetchWithDemoRecovery";
import {
  ArrowLeft,
  Loader2,
  CheckCircle2,
  Clock,
  Users,
  Crown,
  XCircle,
  Rocket,
  Copy,
  Check,
} from "lucide-react";
import { Input } from "@/components/ui/input";
import { MarkdownRenderer } from "@/components/ui/MarkdownRenderer";
import { PriceDisplay } from "@/components/games/PriceDisplay";
import {
  Popover,
  PopoverContent,
  PopoverAnchor,
} from "@/components/ui/popover";
import { TimeDisplay } from "@/components/ui/TimeDisplay";
import UserProfileMenu from "@/components/auth/UserProfileMenu";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import { NumberLimitEditor } from "@/components/ui/NumberLimitEditor";
import { ParticipantGrid } from "@/components/games/ParticipantGrid";
import { GameStatusBadge } from "@/components/games/GameStatusBadge";
import { DateTimeEditor } from "@/components/ui/DateTimeEditor";
import { enUS, pt } from "date-fns/locale";

interface Game {
  id: string;
  name: string;
  organizerId: number;
  description?: string;
  location?: string;
  startsAt?: string;
  durationMinutes?: number;
  maxPlayers?: number;
  maxWaitlistSize?: number;
  maxGuestsPerPlayer?: number;
  totalPriceCents?: number;
  gameSpotsLeft?: number;
  waitlistSpotsLeft?: number;
  createdAt: string;
  updatedAt: string;
  publishedAt?: string | null;
}

interface AuthUser {
  id: string;
  email: string;
  name?: string;
  picture?: string;
  isDemo: boolean;
}

interface Participant {
  status: "going" | "not_going" | "waitlisted";
  user: AuthUser;
  guests: number;
  createdAt: string;
  updatedAt: string;
}

export default function GameDetailPage() {
  const { t, i18n } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  // Helper function to get initials from name
  const getInitials = (name?: string, email?: string) => {
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

  const formatSpotsLeft = (value?: number) => {
    if (value === undefined || value === null) return "—";
    return value.toString();
  };

  const [game, setGame] = useState<Game | null>(null);
  const [organizer, setOrganizer] = useState<AuthUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [organizerHintOpen, setOrganizerHintOpen] = useState(false);
  const [isPublishing, setIsPublishing] = useState(false);
  const [publishError, setPublishError] = useState<string | null>(null);
  const [publishAtInput, setPublishAtInput] = useState("");
  const [nowTs, setNowTs] = useState(() => Date.now());
  const [isEditingSchedule, setIsEditingSchedule] = useState(false);

  // Participants state
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [isLoadingParticipants, setIsLoadingParticipants] = useState(false);
  const [participantsError, setParticipantsError] = useState<string | null>(
    null
  );
  const [isUpdatingParticipation, setIsUpdatingParticipation] = useState(false);
  const [showUngoingConfirmation, setShowUngoingConfirmation] = useState(false);
  const [showOrganizerWarning, setShowOrganizerWarning] = useState(false);
  const [guestCountInput, setGuestCountInput] = useState("0");

  // Share state
  const [shareUrlCopied, setShareUrlCopied] = useState(false);

  // Editing state
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<string>("");

  const datePickerLocale = useMemo(
    () => (i18n.language === "pt-PT" ? pt : enUS),
    [i18n.language]
  );

  // Autosave status per field
  const saveTimersRef = useRef<Record<string, number | undefined>>({});
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement>(null);

  useEffect(() => {
    const fetchAll = async () => {
      try {
        setIsLoading(true);
        setError(null);

        const response = await fetchWithDemoRecovery(
          `${API_BASE_URL}/api/games/${id}`,
          {
            credentials: "include",
          }
        );

        if (!response.ok) {
          if (response.status === 401) {
            redirectToLogin();
            return;
          }
          if (response.status === 404) {
            throw new Error(t("game.gameNotFound"));
          }
          throw new Error(t("game.errorLoadingGame"));
        }

        const gameData = await response.json();
        setGame(gameData.game);
        setOrganizer(gameData.organizer);

        // Fetch authenticated user (optional if unauthenticated)
        try {
          const meResp = await fetchWithDemoRecovery(
            `${API_BASE_URL}/api/auth/me`,
            {
              credentials: "include",
            }
          );
          if (meResp.ok) {
            const me = await meResp.json();
            setUser(me);
          }
        } catch {
          // ignore user fetch errors, treat as not logged in
        }
      } catch (err) {
        setError(
          err instanceof Error ? err.message : t("errors.somethingWentWrong")
        );
      } finally {
        setIsLoading(false);
      }
    };

    if (id) {
      fetchAll();
    }
  }, [id]);

  useEffect(() => {
    const timer = window.setInterval(() => setNowTs(Date.now()), 30000);
    return () => window.clearInterval(timer);
  }, []);

  const refreshGame = async () => {
    if (!id) return;
    try {
      const response = await fetchWithDemoRecovery(
        `${API_BASE_URL}/api/games/${id}`,
        {
          credentials: "include",
        }
      );
      if (response.ok) {
        const gameData = await response.json();
        setGame(gameData.game);
        setOrganizer(gameData.organizer);
      }
    } catch (err) {
      console.error("Error refreshing game:", err);
    }
  };

  const fetchParticipants = async () => {
    if (!id) return;
    try {
      setIsLoadingParticipants(true);
      setParticipantsError(null);
      const response = await fetchWithDemoRecovery(
        `${API_BASE_URL}/api/games/${id}/participants`,
        {
          credentials: "include",
        }
      );
      if (response.ok) {
        const data = await response.json();
        setParticipants(data);
      } else if (response.status === 401) {
        redirectToLogin();
      } else {
        throw new Error(t("game.failedToLoadParticipants"));
      }
    } catch (err) {
      setParticipantsError(
        err instanceof Error ? err.message : t("game.failedToLoadParticipants")
      );
    } finally {
      setIsLoadingParticipants(false);
    }
  };

  useEffect(() => {
    if (game?.publishedAt) {
      fetchParticipants();
    }
  }, [game?.publishedAt, id]);

  const handleUserChange = (newUser: AuthUser) => {
    setUser(newUser);
    // Refetch game when user changes
    refreshGame();
    if (game?.publishedAt) {
      fetchParticipants();
    }
  };

  const updateParticipation = async (
    status: "going" | "not_going",
    guestCount?: number
  ) => {
    if (!id || !user) return;
    try {
      setIsUpdatingParticipation(true);
      const body: { status: string; guests?: number } = { status };
      if (guestCount !== undefined) {
        body.guests = guestCount;
      }
      const response = await fetchWithDemoRecovery(
        `${API_BASE_URL}/api/games/${id}/participants`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify(body),
        }
      );
      if (response.ok) {
        await Promise.all([refreshGame(), fetchParticipants()]);
      } else if (response.status === 401) {
        redirectToLogin();
      } else {
        const txt = await response.text();
        throw new Error(txt || t("game.failedToUpdateParticipation"));
      }
    } catch (err) {
      setParticipantsError(
        err instanceof Error
          ? err.message
          : t("game.failedToUpdateParticipation")
      );
    } finally {
      setIsUpdatingParticipation(false);
    }
  };

  const isOrganizer = useMemo(() => {
    if (!game || !user) return false;
    const userIdNum = Number.parseInt(user.id, 10);
    if (Number.isNaN(userIdNum)) return false;
    return userIdNum === game.organizerId;
  }, [game, user]);

  const currentUserParticipation = useMemo(() => {
    if (!user) return null;
    return participants.find((p) => p.user.id === user.id);
  }, [participants, user]);

  const participantCounts = useMemo(() => {
    const going = participants.reduce(
      (sum, p) => sum + (p.status === "going" ? 1 + p.guests : 0),
      0
    );
    const waitlisted = participants.reduce(
      (sum, p) => sum + (p.status === "waitlisted" ? 1 + p.guests : 0),
      0
    );
    const notGoing = participants.filter(
      (p) => p.status === "not_going"
    ).length;
    return { going, waitlisted, notGoing };
  }, [participants]);

  const isGameFull = useMemo(() => {
    if (!game) return false;

    const noGameSpots =
      typeof game.gameSpotsLeft === "number" &&
      game.gameSpotsLeft <= 0;

    const noWaitlistSpots =
      typeof game.waitlistSpotsLeft === "number" &&
      game.waitlistSpotsLeft <= 0;

    return noGameSpots && noWaitlistSpots;
  }, [game]);

  const organizerJoiningWouldDisplace = useMemo(() => {
    if (!game || !isOrganizer) return false;
    const guestCount = parseInt(guestCountInput, 10) || 0;
    const totalSpotsNeeded = 1 + guestCount;
    
    const availableSpots = game.gameSpotsLeft ?? 0;
    
    return availableSpots < totalSpotsNeeded;
  }, [game, isOrganizer, guestCountInput]);

  const joinButtonDisabled = useMemo(() => {
    return isUpdatingParticipation || (!isOrganizer && isGameFull);
  }, [isUpdatingParticipation, isOrganizer, isGameFull]);
  const publishRequirements = useMemo(() => {
    if (!game) return [];
    return [
      {
        label: t("game.locationSet"),
        met: !!game.location,
        field: "location",
      },
      {
        label: t("game.startTimeSet"),
        met: !!game.startsAt,
        field: "startsAt",
      },
      {
        label: t("game.durationSet"),
        met:
          typeof game.durationMinutes === "number" && game.durationMinutes > 0,
        field: "durationMinutes",
      },
      {
        label: t("game.maxPlayersSet"),
        met: typeof game.maxPlayers === "number" && (game.maxPlayers > 0 || game.maxPlayers === -1),
        field: "maxPlayers",
      },
      {
        label: t("game.waitlistSizeSet"),
        met: typeof game.maxWaitlistSize === "number",
        field: "maxWaitlistSize",
      },
      {
        label: t("game.guestsPerPlayerSet"),
        met: typeof game.maxGuestsPerPlayer === "number",
        field: "maxGuestsPerPlayer",
      },
      {
        label: t("game.pricingSet"),
        met:
          typeof game.totalPriceCents === "number" && game.totalPriceCents >= 0,
        field: "totalPriceCents",
      },
    ];
  }, [game, t]);

  const canPublish = useMemo(() => {
    return publishRequirements.every((req) => req.met);
  }, [publishRequirements]);

  const publishedAtDate = useMemo(() => {
    if (!game?.publishedAt) return null;
    const d = new Date(game.publishedAt);
    if (Number.isNaN(d.getTime())) return null;
    return d;
  }, [game?.publishedAt]);

  const isScheduled = useMemo(() => {
    if (!publishedAtDate) return false;
    return publishedAtDate.getTime() > nowTs;
  }, [publishedAtDate, nowTs]);

  const isPublished = useMemo(() => {
    if (!publishedAtDate) return false;
    // Treat as published if the timestamp is in the past, or within 60 seconds of now
    // (to handle server/client clock differences when publishing "now")
    return publishedAtDate.getTime() <= nowTs + 60000;
  }, [publishedAtDate, nowTs]);

  // Total slots to display in the participant grids.
  // ParticipantGrid uses this to calculate empty slots: totalSlots - participants.length
  // This should be the game's capacity (maxPlayers/maxWaitlistSize), not remaining spots.
  const maxGoingCount = game?.maxPlayers;
  const maxWaitlistCount = game?.maxWaitlistSize;

  useEffect(() => {
    if (game?.publishedAt) {
      setPublishAtInput(toLocalInputValue(game.publishedAt));
      setIsEditingSchedule(false);
    } else {
      setPublishAtInput("");
    }
  }, [game?.publishedAt]);

  // Focus input when editing starts
  useEffect(() => {
    if (editingField && inputRef.current) {
      inputRef.current.focus();
    }
  }, [editingField]);

  function startEditing(field: string, currentValue: unknown) {
    if (!isOrganizer) return;
    setEditingField(field);
    if (field === "totalPriceCents" && typeof currentValue === "number") {
      setEditValue(formatCentsAsDollars(currentValue));
    } else if (field === "startsAt" && typeof currentValue === "string") {
      setEditValue(toLocalInputValue(currentValue));
    } else if (typeof currentValue === "number") {
      setEditValue(String(currentValue));
    } else if (typeof currentValue === "string") {
      setEditValue(currentValue);
    } else {
      setEditValue("");
    }
  }

  function cancelEditing() {
    setEditingField(null);
    setEditValue("");
    cancelAllDebounces();
  }

  function cancelAllDebounces() {
    Object.keys(saveTimersRef.current).forEach((field) => {
      const timerId = saveTimersRef.current[field];
      if (timerId !== undefined) {
        clearTimeout(timerId);
        saveTimersRef.current[field] = undefined;
      }
    });
  }

  async function updatePublishTime(
    publishedAtValue: string | null,
    requireReady = true
  ) {
    if (!isOrganizer || !id) return false;
    if (requireReady && !canPublish) {
      setPublishError(t("game.completeFieldsBeforePublishing"));
      return false;
    }

    setIsPublishing(true);
    setPublishError(null);

    try {
      const resp = await fetch(`${API_BASE_URL}/api/games/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ publishedAt: publishedAtValue }),
      });

      if (!resp.ok) {
        if (resp.status === 401) {
          redirectToLogin();
          return false;
        }
        const txt = await resp.text();
        throw new Error(txt || t("game.failedToUpdatePublishTime"));
      }

      const updated = await resp.json();
      setGame(updated.game);
      setOrganizer(updated.organizer);
      return true;
    } catch (e) {
      setPublishError(
        e instanceof Error ? e.message : t("game.failedToUpdatePublishTime")
      );
      return false;
    } finally {
      setIsPublishing(false);
    }
  }

  async function handlePublishNow() {
    const ok = await updatePublishTime(new Date().toISOString());
    if (ok) {
      setIsEditingSchedule(false);
    }
  }

  async function handleSchedulePublish() {
    if (!publishAtInput) {
      setPublishError(t("game.selectDateTimeToSchedule"));
      return;
    }
    const iso = fromLocalInputValue(publishAtInput);
    if (!iso) {
      setPublishError(t("game.invalidDateTime"));
      return;
    }
    const ok = await updatePublishTime(iso);
    if (ok) setIsEditingSchedule(false);
  }

  async function saveField(field: string, value: unknown) {
    if (!isOrganizer || !id) return;
    // Skip sending empty strings to avoid accidental clearing until supported
    if (typeof value === "string" && value.trim() === "") return;
    if (value === undefined) return;

    try {
      const resp = await fetch(`${API_BASE_URL}/api/games/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ [field]: value }),
      });
      if (!resp.ok) {
        if (resp.status === 401) {
          redirectToLogin();
          return;
        }
        const txt = await resp.text();
        throw new Error(txt || t("game.failedToSave"));
      }
      const updated = await resp.json();
      setGame(updated.game);
      setOrganizer(updated.organizer);
      setEditingField(null);
    } catch (e) {
      console.error("Failed to save field:", field, e);
    }
  }

  function handleBlur(field: string) {
    if (editingField !== field) return;
    cancelDebouncedSave(field);

    let valueToSave: unknown = editValue;
    if (field === "totalPriceCents") {
      // Convert dollars.cents format to cents
      const dollars = parseFloat(editValue);
      if (isNaN(dollars)) {
        valueToSave = undefined;
      } else {
        // Round to ensure we have whole cents
        valueToSave = Math.round(dollars * 100);
      }
    } else if (field === "maxPlayers" || field === "durationMinutes") {
      const num = Number(editValue);
      valueToSave = isNaN(num) ? undefined : num;
    } else if (field === "startsAt") {
      valueToSave = fromLocalInputValue(editValue);
    }

    if (valueToSave !== undefined && valueToSave !== "") {
      saveField(field, valueToSave);
    } else {
      cancelEditing();
    }
  }

  function cancelDebouncedSave(field: string) {
    const timers = saveTimersRef.current;
    const existing = timers[field];
    if (existing !== undefined) {
      clearTimeout(existing);
      timers[field] = undefined;
    }
  }

  function handleCopyShareLink() {
    const shareUrl = window.location.href;
    navigator.clipboard.writeText(shareUrl).then(() => {
      setShareUrlCopied(true);
      setTimeout(() => setShareUrlCopied(false), 2000);
    });
  }

  async function handleJoinWithGuests() {
    const guestCount = parseInt(guestCountInput, 10);
    if (isNaN(guestCount) || guestCount < 0) {
      return;
    }
    const maxGuests = game?.maxGuestsPerPlayer ?? 0;
    if (maxGuests !== -1 && guestCount > maxGuests) {
      return;
    }
    
    console.log('handleJoinWithGuests:', { 
      isOrganizer, 
      organizerJoiningWouldDisplace,
      gameSpotsLeft: game?.gameSpotsLeft,
      guestCount,
      totalNeeded: 1 + guestCount
    });
    
    // Check if organizer joining would displace others
    if (isOrganizer && organizerJoiningWouldDisplace) {
      console.log('Showing organizer warning');
      setShowOrganizerWarning(true);
      return;
    }
    
    console.log('Calling updateParticipation');
    await updateParticipation("going", guestCount);
  }

  async function confirmOrganizerJoin() {
    const guestCount = parseInt(guestCountInput, 10) || 0;
    setShowOrganizerWarning(false);
    await updateParticipation("going", guestCount);
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50 flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50">
        <div className="container mx-auto px-4 py-8">
          <Button
            variant="ghost"
            onClick={() => navigate("/")}
            className="mb-8"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t("game.backToGames")}
          </Button>

          <div className="bg-red-50 border border-red-200 rounded-xl p-8 text-center">
            <h1 className="text-2xl font-bold text-red-900 mb-2">
              {t("common.error")}
            </h1>
            <p className="text-red-700">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  if (!game) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50">
        <div className="container mx-auto px-4 py-8">
          <Button
            variant="ghost"
            onClick={() => navigate("/")}
            className="mb-8"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t("game.backToGames")}
          </Button>

          <div className="text-center py-20">
            <p className="text-gray-600">{t("game.gameNotFound")}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50">
      <div className="container mx-auto px-4 py-8 max-w-5xl">
        {/* Header with Back Button */}
        <div className="flex items-center justify-between mb-8">
          <Button
            variant="ghost"
            onClick={() => navigate("/")}
            className="text-gray-600 hover:text-gray-900"
          >
            <ArrowLeft className="mr-2 h-5 w-5" />
            {t("common.back")}
          </Button>
          <div className="flex items-center gap-2">
            <LanguageSwitcher />
            <UserProfileMenu user={user} onUserChange={handleUserChange} />
          </div>
        </div>

        {/* Main Game Card */}
        <Card className="overflow-hidden border-l-8 border-l-primary mb-8">
          {/* Hero Section with Status Badge */}
          <div className="relative p-8 pb-6">
            <div className="flex justify-between items-start gap-4 mb-4">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-3">
                  {isOrganizer && (
                    <div
                      onMouseEnter={() => setOrganizerHintOpen(true)}
                      onMouseLeave={() => setOrganizerHintOpen(false)}
                    >
                      <Popover open={organizerHintOpen}>
                        <PopoverAnchor asChild>
                          <span className="inline-flex items-center bg-secondary/20 rounded-full p-1.5 cursor-help">
                            <Crown className="h-5 w-5 text-primary" />
                          </span>
                        </PopoverAnchor>
                        <PopoverContent
                          side="right"
                          className="text-sm rounded-xl"
                        >
                          <p className="font-semibold mb-1">
                            {t("game.youAreOrganizing")}
                          </p>
                          <p className="text-gray-600">
                            {t("game.youCanEditEverything")}
                          </p>
                        </PopoverContent>
                      </Popover>
                    </div>
                  )}
                  {editingField === "name" && isOrganizer ? (
                    <div>
                      <Input
                        value={editValue}
                        onChange={(e) => setEditValue(e.target.value)}
                        onBlur={() => handleBlur("name")}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") handleBlur("name");
                          if (e.key === "Escape") setEditingField(null);
                        }}
                        maxLength={100}
                        className={`text-3xl font-bold ${
                          editValue.length >= 100 ? "border-red-500" : ""
                        }`}
                      />
                      <div className="mt-1 flex justify-end">
                        <span
                          className={`text-xs ${
                            editValue.length >= 100
                              ? "text-red-600"
                              : "text-gray-500"
                          }`}
                        >
                          {editValue.length}/100
                        </span>
                      </div>
                    </div>
                  ) : (
                    <h1
                      className="text-4xl font-bold text-gray-900 cursor-text transition"
                      onClick={() => startEditing("name", game?.name || "")}
                    >
                      {game?.name || t("common.game")}
                    </h1>
                  )}
                </div>

                {/* Status Badge */}
                <div className="flex flex-wrap gap-2 items-center">
                  <GameStatusBadge
                    state={
                      isPublished
                        ? "published"
                        : isScheduled
                        ? "scheduled"
                        : "draft"
                    }
                    publishedAt={publishedAtDate ?? undefined}
                  />
                  {isPublished && (
                    <button
                      onClick={handleCopyShareLink}
                      className="inline-flex items-center gap-2 text-sm font-semibold px-3 py-1.5 rounded-full bg-gray-100 text-gray-700 hover:bg-gray-200 transition"
                      title={t("common.copyGameLink")}
                    >
                      {shareUrlCopied ? (
                        <>
                          <Check className="h-4 w-4" />
                          {t("common.copied")}
                        </>
                      ) : (
                        <>
                          <Copy className="h-4 w-4" />
                          {t("common.share")}
                        </>
                      )}
                    </button>
                  )}
                </div>
                {isOrganizer && isPublished && (
                  <div className="bg-blue-50 border border-blue-200 rounded-lg p-3 text-sm text-blue-900 mt-2">
                    <span className="font-semibold">
                      {t("game.thisGameIsPublished")}
                    </span>{" "}
                    {t("game.anyoneWithLink")}
                  </div>
                )}
              </div>
            </div>

            {/* Quick Stats */}
            {isPublished && (
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4 mt-6 pt-6">
                <div className="text-center">
                  {/* Organizer Info */}
                  {organizer && (
                    <div className="flex items-center gap-3 mt-2">
                      <Avatar className="h-8 w-8 flex-shrink-0">
                        <AvatarImage
                          src={organizer.picture || undefined}
                          alt={organizer.name || organizer.email}
                        />
                        <AvatarFallback className="text-xs">
                          {getInitials(organizer.name, organizer.email)}
                        </AvatarFallback>
                      </Avatar>
                      <div>
                        <p className="text-xs text-gray-500 font-medium">
                          {t("game.organizer")}
                        </p>
                        <p className="text-sm font-semibold text-gray-900">
                          {organizer.name || organizer.email}
                        </p>
                      </div>
                    </div>
                  )}
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-primary mb-1">
                    {formatSpotsLeft(game?.gameSpotsLeft)}
                  </div>
                  <div className="text-xs text-gray-600 font-medium">
                    {t("game.spotsLeft")}
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-accent mb-1">
                    {formatSpotsLeft(game?.waitlistSpotsLeft)}
                  </div>
                  <div className="text-xs text-gray-600 font-medium">
                    {t("game.waitlistSpotsLeft")}
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Game Details */}
          <div className="grid md:grid-cols-2 gap-6 p-8 border-t border-gray-100">
            {/* Left Column */}
            <div className="space-y-6">
              {/* Location */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">
                  {t("game.location")}
                </label>
                {editingField === "location" && isOrganizer ? (
                  <Input
                    type="text"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur("location")}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleBlur("location");
                      if (e.key === "Escape") setEditingField(null);
                    }}
                    placeholder={t("game.locationPlaceholder")}
                  />
                ) : (
                  <div
                    onClick={() =>
                      startEditing("location", game?.location || "")
                    }
                    className={`text-lg font-semibold cursor-text transition ${
                      game?.location ? "text-gray-900" : "text-gray-400"
                    }`}
                  >
                    {game?.location ||
                      (isOrganizer ? t("game.clickToAddLocation") : "—")}
                  </div>
                )}
              </div>

              {/* Date & Time */}
              <div>
                {editingField === "startsAt" && isOrganizer ? (
                  <DateTimeEditor
                    value={editValue || toLocalInputValue(game?.startsAt || "")}
                    onChange={setEditValue}
                    locale={datePickerLocale}
                    dateLabel={t("game.date", { defaultValue: "Date" })}
                    timeLabel={t("game.time", { defaultValue: "Time" })}
                    onBlur={() => handleBlur("startsAt")}
                  />
                ) : (
                  <div>
                    <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">
                      {t("game.when")}
                    </label>
                    <div
                      onClick={() =>
                        startEditing("startsAt", game?.startsAt || "")
                      }
                      className={`text-lg font-semibold cursor-text transition ${
                        game?.startsAt ? "text-gray-900" : "text-gray-400"
                      }`}
                    >
                      {game?.startsAt ? (
                        <TimeDisplay
                          timestamp={game.startsAt}
                          displayFormat="friendly"
                          className="text-gray-900"
                        />
                      ) : isOrganizer ? (
                        t("game.clickToSetTime")
                      ) : (
                        "—"
                      )}
                    </div>
                  </div>
                )}
              </div>

              {/* Duration */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">
                  {t("game.duration")}
                </label>
                {editingField === "durationMinutes" && isOrganizer ? (
                  <Input
                    type="number"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur("durationMinutes")}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleBlur("durationMinutes");
                      if (e.key === "Escape") setEditingField(null);
                    }}
                    placeholder={t("common.minutesPlaceholder")}
                  />
                ) : (
                  <div
                    onClick={() =>
                      startEditing(
                        "durationMinutes",
                        game?.durationMinutes || ""
                      )
                    }
                    className={`text-lg font-semibold cursor-text transition ${
                      game?.durationMinutes ? "text-gray-900" : "text-gray-400"
                    }`}
                  >
                    {game?.durationMinutes
                      ? `${game.durationMinutes} min`
                      : isOrganizer
                      ? t("game.clickToAddDuration")
                      : "—"}
                  </div>
                )}
              </div>
            </div>

            {/* Right Column */}
            <div className="space-y-6">
              {/* Max Players */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">
                  {t("common.players")}
                </label>
                {editingField === "maxPlayers" && isOrganizer ? (
                  <NumberLimitEditor
                    value={game?.maxPlayers}
                    onSave={(value) => {
                      if (value !== undefined) {
                        saveField("maxPlayers", value);
                      } else {
                        cancelEditing();
                      }
                    }}
                    onCancel={cancelEditing}
                    showDisabledOption={false}
                    placeholder={t("common.enterMaxPlayers")}
                    label={{
                      limited: t("common.setMaximum"),
                    }}
                  />
                ) : (
                  <div
                    onClick={() =>
                      startEditing("maxPlayers", game?.maxPlayers || "")
                    }
                    className={`text-lg font-semibold cursor-text transition ${
                      game?.maxPlayers ? "text-gray-900" : "text-gray-400"
                    }`}
                  >
                    {game?.maxPlayers
                      ? `${t("common.upTo")} ${game.maxPlayers}`
                      : isOrganizer
                      ? t("common.clickToSet")
                      : "—"}
                  </div>
                )}
              </div>

              {/* Waitlist Size */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">
                  {t("common.waitlist")}
                </label>
                {editingField === "maxWaitlistSize" && isOrganizer ? (
                  <NumberLimitEditor
                    value={game?.maxWaitlistSize}
                    onSave={(value) => {
                      if (value !== undefined) {
                        saveField("maxWaitlistSize", value);
                      } else {
                        cancelEditing();
                      }
                    }}
                    onCancel={cancelEditing}
                    placeholder={t("common.enterMaxWaitlistSize")}
                  />
                ) : (
                  <div
                    onClick={() =>
                      startEditing(
                        "maxWaitlistSize",
                        game?.maxWaitlistSize ?? ""
                      )
                    }
                    className={`text-lg font-semibold cursor-text transition ${
                      typeof game?.maxWaitlistSize === "number"
                        ? "text-gray-900"
                        : "text-gray-400"
                    }`}
                  >
                    {typeof game?.maxWaitlistSize === "number"
                      ? game.maxWaitlistSize === 0
                        ? t("common.disabled")
                        : `${t("common.upTo")} ${game.maxWaitlistSize}`
                      : isOrganizer
                      ? t("common.clickToSet")
                      : "—"}
                  </div>
                )}
              </div>

              {/* Guests Per Player */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">
                  {t("common.guestsPerPlayer")}
                </label>
                {editingField === "maxGuestsPerPlayer" && isOrganizer ? (
                  <NumberLimitEditor
                    value={game?.maxGuestsPerPlayer}
                    onSave={(value) => {
                      if (value !== undefined) {
                        saveField("maxGuestsPerPlayer", value);
                      } else {
                        cancelEditing();
                      }
                    }}
                    onCancel={cancelEditing}
                    placeholder={t("common.enterMaxGuests")}
                  />
                ) : (
                  <div
                    onClick={() =>
                      startEditing(
                        "maxGuestsPerPlayer",
                        game?.maxGuestsPerPlayer ?? ""
                      )
                    }
                    className={`text-lg font-semibold cursor-text transition ${
                      typeof game?.maxGuestsPerPlayer === "number"
                        ? "text-gray-900"
                        : "text-gray-400"
                    }`}
                  >
                    {typeof game?.maxGuestsPerPlayer === "number"
                      ? game.maxGuestsPerPlayer === 0
                        ? t("common.disabled")
                        : `${t("common.upTo")} ${game.maxGuestsPerPlayer}`
                      : isOrganizer
                      ? t("common.clickToSet")
                      : "—"}
                  </div>
                )}
              </div>

              {/* Price */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">
                  {t("game.price")}
                </label>
                {editingField === "totalPriceCents" && isOrganizer ? (
                  <Input
                    type="number"
                    step="0.01"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur("totalPriceCents")}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleBlur("totalPriceCents");
                      if (e.key === "Escape") setEditingField(null);
                    }}
                    placeholder={t("common.pricePlaceholder")}
                  />
                ) : (
                  <div
                    onClick={() =>
                      startEditing(
                        "totalPriceCents",
                        game?.totalPriceCents || ""
                      )
                    }
                    className={`text-lg font-semibold cursor-text transition ${
                      game?.totalPriceCents !== undefined &&
                      game.totalPriceCents >= 0
                        ? "text-gray-900"
                        : "text-gray-400"
                    }`}
                  >
                    {game?.totalPriceCents !== undefined &&
                    game.totalPriceCents >= 0 ? (
                      <PriceDisplay
                        totalPriceCents={game.totalPriceCents}
                        maxPlayers={game.maxPlayers}
                      />
                    ) : isOrganizer ? (
                      t("game.clickToAddPrice")
                    ) : (
                      "—"
                    )}
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Description */}
          {(game?.description || isOrganizer) && (
            <div className="px-8 py-6 border-t border-gray-100">
              <div className="flex items-center justify-between mb-3">
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide block">
                  {t("game.description")}
                </label>
                <span className="text-xs text-gray-400">
                  {t("common.markdownSupported")}
                </span>
              </div>
              {editingField === "description" && isOrganizer ? (
                <div>
                  <textarea
                    ref={
                      inputRef as unknown as React.RefObject<HTMLTextAreaElement>
                    }
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur("description")}
                    maxLength={1000}
                    className={`w-full p-3 border-2 rounded-xl resize-none focus:outline-none ${
                      editValue.length >= 1000
                        ? "border-red-500"
                        : "border-primary"
                    }`}
                    rows={3}
                    placeholder={t("common.tellPeopleAboutGame")}
                  />
                  <div className="mt-1 flex justify-end">
                    <span
                      className={`text-xs ${
                        editValue.length >= 1000
                          ? "text-red-600"
                          : "text-gray-500"
                      }`}
                    >
                      {editValue.length}/1000
                    </span>
                  </div>
                </div>
              ) : (
                <div
                  onClick={() =>
                    startEditing("description", game?.description || "")
                  }
                  className={`cursor-text transition text-sm leading-relaxed ${
                    game?.description ? "text-gray-700" : "text-gray-400"
                  }`}
                >
                  {game?.description ? (
                    <MarkdownRenderer value={game.description} />
                  ) : isOrganizer ? (
                    t("game.clickToAddDescription")
                  ) : (
                    t("common.noDescription")
                  )}
                </div>
              )}
            </div>
          )}

          {/* Participants Section */}
          {isPublished && (
            <div className="px-8 py-6 border-t border-gray-100">
              <h2 className="text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                <Users className="h-5 w-5 text-primary" />
                {t("participants.whosComing")} (
                {(game?.maxPlayers || 0) > 0
                  ? String(participantCounts.going) +
                    "/" +
                    String(game?.maxPlayers || "?")
                  : String(participantCounts.going)}
                )
              </h2>

              {isLoadingParticipants ? (
                <div className="flex justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-primary" />
                </div>
              ) : participantsError ? (
                <div className="bg-red-50 border border-red-200 rounded-xl p-4 text-red-700 text-sm">
                  {participantsError}
                </div>
              ) : participants.length > 0 ? (
                <div>
                  {/* People Grid */}
                  <div className="mb-6">
                    <ParticipantGrid
                      participants={participants.filter(
                        (p) => p.status === "going"
                      )}
                      organizerId={game?.organizerId}
                      totalSlots={maxGoingCount}
                      occupiedSlots={participantCounts.going}
                      icon={Crown}
                    />
                  </div>

                  {/* Waitlist */}
                  {(game.maxWaitlistSize || 0) > 0 && (
                    <div className="mt-6 pt-6 border-t border-gray-200">
                      <p className="text-sm font-semibold text-gray-700 mb-3 flex items-center gap-2">
                        <Clock className="h-4 w-4 text-gray-600" />
                        <span>
                          {t("participants.waitlistCount")} (
                          {(game?.maxWaitlistSize || 0) > 0
                            ? String(participantCounts.waitlisted) +
                              "/" +
                              String(game?.maxWaitlistSize || "?")
                            : String(participantCounts.waitlisted)}
                          )
                        </span>
                      </p>
                      <ParticipantGrid
                        participants={participants.filter(
                          (p) => p.status === "waitlisted"
                        )}
                        organizerId={game?.organizerId}
                        totalSlots={maxWaitlistCount}
                        occupiedSlots={participantCounts.waitlisted}
                        icon={Clock}
                        size="sm"
                        opacity={0.7}
                        emptySlotLabel={t("common.available")}
                      />
                    </div>
                  )}
                  {participantCounts.notGoing > 0 && (
                    <div className="mt-6 pt-6 border-t border-gray-200">
                      <p className="text-sm font-semibold text-gray-700 mb-3 flex items-center gap-2">
                        <XCircle className="h-4 w-4 text-gray-600" />
                        <span>
                          {t("participants.notGoingCount")} (
                          {participantCounts.notGoing})
                        </span>
                      </p>
                      <ParticipantGrid
                        participants={participants.filter(
                          (p) => p.status === "not_going"
                        )}
                        organizerId={game?.organizerId}
                        size="sm"
                        opacity={0.6}
                        icon={XCircle}
                      />
                    </div>
                  )}
                </div>
              ) : (
                <div className="bg-yellow-50 border-2 border-dashed border-secondary rounded-xl p-8 text-center">
                  <Users className="h-12 w-12 mx-auto mb-3 text-secondary/40" />
                  <p className="text-gray-600 font-medium mb-2">
                    {t("participants.almostThere")}
                  </p>
                  <p className="text-sm text-gray-500">
                    {t("participants.beFirstToSignUp")}
                  </p>
                </div>
              )}
            </div>
          )}

          {/* Action Buttons */}
          <div className="px-8 py-6 border-t border-gray-100 bg-gray-50/50">
            {isOrganizer && !isPublished ? (
              <div className="space-y-4">
                {/* Requirements Checklist */}
                <div className="bg-white p-4 rounded-xl border-2 border-primary/20">
                  <h3 className="font-semibold text-gray-900 mb-3 flex items-center gap-2">
                    <Rocket className="h-5 w-5 text-primary" />
                    {t("publish.readyToShare")}
                  </h3>

                  <div className="bg-blue-50 border border-blue-200 rounded-lg p-3 mb-4 text-sm text-blue-900">
                    <p className="font-semibold mb-1">
                      📍 {t("publish.visibility")}
                    </p>
                    <p>
                      {t("publish.gameCurrentlyPrivate")}{" "}
                      <span className="font-semibold">
                        {t("publish.onlyVisibleToYou")}
                      </span>
                      {t("publish.oncePublishedAnyone")}
                    </p>
                  </div>

                  <div className="space-y-2 mb-4">
                    {publishRequirements.map((req) => (
                      <div
                        key={req.field}
                        className="flex items-center gap-2 text-sm"
                      >
                        {req.met ? (
                          <CheckCircle2 className="h-4 w-4 text-success flex-shrink-0" />
                        ) : (
                          <XCircle className="h-4 w-4 text-gray-300 flex-shrink-0" />
                        )}
                        <span
                          className={
                            req.met ? "text-gray-700" : "text-gray-400"
                          }
                        >
                          {req.label}
                        </span>
                      </div>
                    ))}
                  </div>

                  <div className="flex gap-2">
                    <Button
                      onClick={handlePublishNow}
                      disabled={isPublishing || !canPublish}
                      className={canPublish ? "bg-success" : "bg-gray-400"}
                    >
                      {isPublishing ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          {t("publish.publishing")}
                        </>
                      ) : (
                        <>
                          <Rocket className="mr-2 h-4 w-4" />
                          {t("publish.publishNow")}
                        </>
                      )}
                    </Button>
                    {canPublish && (
                      <Button
                        variant="outline"
                        onClick={() => setIsEditingSchedule(!isEditingSchedule)}
                      >
                        {isScheduled
                          ? t("publish.reschedule")
                          : t("publish.schedule")}
                      </Button>
                    )}
                  </div>

                  {isEditingSchedule && canPublish && (
                    <div className="mt-4 pt-4 border-t border-gray-200 space-y-3">
                      <DateTimeEditor
                        value={publishAtInput}
                        onChange={setPublishAtInput}
                        locale={datePickerLocale}
                        dateLabel={t("game.date", { defaultValue: "Date" })}
                        timeLabel={t("game.time", { defaultValue: "Time" })}
                      />
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          onClick={() => setIsEditingSchedule(false)}
                        >
                          {t("common.cancel")}
                        </Button>
                        <Button onClick={handleSchedulePublish}>
                          {isScheduled ? t("publish.updateSchedule") : t("publish.schedulePublish")}
                        </Button>
                      </div>
                    </div>
                  )}

                  {publishError && (
                    <div className="mt-3 text-red-600 text-sm">
                      {publishError}
                    </div>
                  )}
                </div>
              </div>
            ) : user && isPublished ? (
              <div>
                {currentUserParticipation && currentUserParticipation.status !== "not_going" ? (
                  <>
                    <Button
                      variant="outline"
                      onClick={() => setShowUngoingConfirmation(true)}
                      disabled={isUpdatingParticipation}
                      className="w-full bg-accent/10 border-accent text-accent hover:bg-accent/20"
                    >
                      {isUpdatingParticipation
                        ? t("participants.updating")
                        : currentUserParticipation.status === "waitlisted"
                        ? t("participants.leaveWaitlist")
                        : t("participants.youreGoing")}
                    </Button>
                    <Dialog
                      open={showUngoingConfirmation}
                      onOpenChange={setShowUngoingConfirmation}
                    >
                      <DialogContent>
                        <DialogHeader>
                          <DialogTitle>
                            {currentUserParticipation?.status === "waitlisted"
                              ? t("participants.leaveWaitlist")
                              : t("participants.changeYourVote")}
                          </DialogTitle>
                          <DialogDescription>
                            {currentUserParticipation?.status === "waitlisted"
                              ? t("participants.leaveWaitlistConfirm")
                              : t("participants.changeToNotGoing")}
                          </DialogDescription>
                        </DialogHeader>
                        <DialogFooter>
                          <Button
                            variant="outline"
                            onClick={() => setShowUngoingConfirmation(false)}
                          >
                            {currentUserParticipation?.status === "waitlisted"
                              ? t("participants.stayOnWaitlist")
                              : t("participants.keepMeGoing")}
                          </Button>
                          <Button
                            variant="destructive"
                            onClick={async () => {
                              setShowUngoingConfirmation(false);
                              await updateParticipation("not_going");
                            }}
                            disabled={isUpdatingParticipation}
                          >
                            {isUpdatingParticipation
                              ? t("participants.updating")
                              : currentUserParticipation?.status === "waitlisted"
                              ? t("participants.yesLeaveWaitlist")
                              : t("participants.yesChangeVote")}
                          </Button>
                        </DialogFooter>
                      </DialogContent>
                    </Dialog>
                  </>
                ) : (
                  <div className="space-y-3">
                    <div className="grid grid-cols-[1fr,auto] gap-3 items-end">
                      <Button
                        onClick={handleJoinWithGuests}
                        disabled={
                          joinButtonDisabled ||
                          (() => {
                            const count = parseInt(guestCountInput, 10);
                            if (isNaN(count) || count < 0) return true;
                            const maxGuests = game?.maxGuestsPerPlayer ?? 0;
                            if (maxGuests !== -1 && count > maxGuests)
                              return true;
                            return false;
                          })()
                        }
                        className="bg-accent w-full"
                      >
                        {isUpdatingParticipation ? (
                          <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            {t("participants.joining")}
                          </>
                        ) : (
                          t("participants.countMeIn")
                        )}
                      </Button>
                      <div className="flex flex-col gap-1">
                        <label className="text-xs font-medium text-gray-600">
                          {t("participants.guests")}
                        </label>
                        <Input
                          type="number"
                          min="0"
                          max={
                            game?.maxGuestsPerPlayer
                          }
                          value={guestCountInput}
                          onChange={(e) => setGuestCountInput(e.target.value)}
                          placeholder="0"
                          disabled={
                            game?.maxGuestsPerPlayer === 0 ||
                            isUpdatingParticipation
                          }
                          className="w-20 text-center"
                        />
                      </div>
                    </div>
                    {(() => {
                      const count = parseInt(guestCountInput, 10);
                      const maxGuests = game?.maxGuestsPerPlayer ?? 0;
                      if (game?.maxGuestsPerPlayer === 0) {
                        return (
                          <p className="text-xs text-gray-500">
                            {t("participants.noGuestsAllowed")}
                          </p>
                        );
                      }
                      if (
                        !isNaN(count) &&
                        count > maxGuests
                      ) {
                        return (
                          <p className="text-xs text-red-600">
                            {t("participants.exceedsMaxGuests", {
                              max: maxGuests,
                            })}
                          </p>
                        );
                      }
                      if (
                        game?.maxGuestsPerPlayer &&
                        game.maxGuestsPerPlayer > 0
                      ) {
                        return (
                          <p className="text-xs text-gray-500">
                            {t("participants.maxGuestsInfo", {
                              max: game.maxGuestsPerPlayer,
                            })}
                          </p>
                        );
                      }
                      return null;
                    })()}
                    {!isOrganizer && isGameFull && (
                      <p className="text-xs text-gray-500">
                        {t("participants.gameIsFull")}
                      </p>
                    )}
                  </div>
                )}
              </div>
            ) : null}
          </div>
        </Card>
        
        {/* Organizer Warning Dialog */}
        <Dialog
          open={showOrganizerWarning}
          onOpenChange={setShowOrganizerWarning}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <Crown className="h-5 w-5 text-primary" />
                {t("participants.organizerPriority")}
              </DialogTitle>
              <DialogDescription>
                {t("participants.joiningWillDisplace")}
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setShowOrganizerWarning(false)}
              >
                {t("common.cancel")}
              </Button>
              <Button
                onClick={confirmOrganizerJoin}
                disabled={isUpdatingParticipation}
                className="bg-primary"
              >
                {isUpdatingParticipation
                  ? t("participants.joining")
                  : t("participants.confirmJoin")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}

function toLocalInputValue(iso: string) {
  if (!iso) return "";
  try {
    const d = new Date(iso);
    const pad = (n: number) => String(n).padStart(2, "0");
    const yyyy = d.getFullYear();
    const mm = pad(d.getMonth() + 1);
    const dd = pad(d.getDate());
    const hh = pad(d.getHours());
    const mi = pad(d.getMinutes());
    return `${yyyy}-${mm}-${dd}T${hh}:${mi}`;
  } catch {
    return "";
  }
}

function fromLocalInputValue(val: string) {
  // Convert local datetime input back to ISO string
  try {
    const d = new Date(val);
    return d.toISOString();
  } catch {
    return "";
  }
}

function formatCentsAsDollars(cents?: number) {
  if (typeof cents !== "number" || Number.isNaN(cents)) return "";
  if (cents === 0) return "0";
  const dollars = cents / 100;
  return dollars.toFixed(2);
}
