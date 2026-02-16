import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Calendar, Users, Clock } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import UserProfileMenu from "@/components/auth/UserProfileMenu";
import { TimeDisplay } from "@/components/ui/TimeDisplay";
import type { User } from "@/opengym/client";

interface Organizer {
  name?: string;
  picture?: string;
}

interface PublicGamePreviewProps {
  gameId: string;
  gameName: string;
  organizer: Organizer;
  gameSpotsLeft?: number;
  startsAt?: string;
  onUserChange?: (user: User) => void;
  currentUser: User | null;
}

export function PublicGamePreview({
  gameName,
  organizer,
  gameSpotsLeft,
  startsAt,
  onUserChange,
  currentUser,
}: PublicGamePreviewProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [daysUntilGame, setDaysUntilGame] = useState<number | null>(null);

  useEffect(() => {
    const calculateDaysUntil = () => {
      if (!startsAt) return;
      const now = new Date();
      const gameDate = new Date(startsAt);
      const diffMs = gameDate.getTime() - now.getTime();
      const days = Math.ceil(diffMs / (1000 * 60 * 60 * 24));
      setDaysUntilGame(days);
    };

    calculateDaysUntil();
    const interval = setInterval(calculateDaysUntil, 60000);
    return () => clearInterval(interval);
  }, [startsAt]);

  const getInitials = (name?: string) => {
    if (!name) return "?";
    const parts = name.split(" ").filter((p) => p.length > 0);
    return parts
      .slice(0, 2)
      .map((p) => p[0].toUpperCase())
      .join("");
  };

  const spotsStatus = useMemo(() => {
    if (gameSpotsLeft === undefined) return null;
    if (gameSpotsLeft === 0) return "full";
    if (gameSpotsLeft <= 2) return "limited";
    return "available";
  }, [gameSpotsLeft]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50">
      <div className="container mx-auto px-4 py-8 max-w-5xl">
        {/* Header */}
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
            {onUserChange && (
              <UserProfileMenu user={currentUser} onUserChange={onUserChange} />
            )}
          </div>
        </div>

        {/* Preview Card */}
        <Card className="overflow-hidden border-l-8 border-l-primary mb-8">
          <div className="p-8">
            {/* Status Badge */}
            <div className="inline-block mb-4">
              <div className="bg-gradient-to-r from-green-100 to-emerald-100 px-4 py-2 rounded-full">
                <p className="text-sm font-semibold text-green-900">
                  {t("game.published", { defaultValue: "Published" })}
                </p>
              </div>
            </div>

            {/* Game Title and Organizer */}
            <div className="mb-6">
              <h1 className="text-4xl font-bold text-gray-900 mb-4">
                {gameName}
              </h1>
              <div className="flex items-center gap-3">
                <Avatar className="h-10 w-10">
                  {organizer.picture && (
                    <AvatarImage src={organizer.picture} alt={organizer.name} />
                  )}
                  <AvatarFallback className="bg-primary text-white">
                    {getInitials(organizer.name)}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <p className="text-sm text-gray-600">
                    {t("game.organizedBy", { defaultValue: "Organized by" })}
                  </p>
                  <p className="font-semibold text-gray-900">
                    {organizer.name || "Unknown"}
                  </p>
                </div>
              </div>
            </div>

            {/* Game Info Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
              {/* Spots Available */}
              <div className="bg-gradient-to-br from-blue-50 to-cyan-50 rounded-lg p-6 border-2 border-blue-200">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="text-sm font-semibold text-gray-600 uppercase tracking-wide mb-1">
                      {t("game.spotsAvailable", { defaultValue: "Spots Available" })}
                    </p>
                    <p className="text-4xl font-bold text-primary">
                      {gameSpotsLeft ?? "—"}
                    </p>
                    {gameSpotsLeft !== undefined && spotsStatus !== "available" && (
                      <p className="text-xs text-gray-500 mt-2">
                        {spotsStatus === "full"
                          ? t("game.gameFull", { defaultValue: "Game is full" })
                          : t("game.limitedSpots", {
                              defaultValue: "Limited spots remaining",
                            })}
                      </p>
                    )}
                  </div>
                  <Users className="h-8 w-8 text-blue-400 flex-shrink-0" />
                </div>
              </div>

              {/* Game Date and Time */}
              {startsAt && (
                <div className="bg-gradient-to-br from-purple-50 to-pink-50 rounded-lg p-6 border-2 border-purple-200">
                  <div className="flex items-start justify-between">
                    <div>
                      <p className="text-sm font-semibold text-gray-600 uppercase tracking-wide mb-1">
                        {t("game.scheduled", { defaultValue: "Scheduled" })}
                      </p>
                      <div className="space-y-1">
                        <TimeDisplay timestamp={startsAt} displayFormat="friendly" />
                        {daysUntilGame !== null && (
                          <p className="text-xs text-gray-500 mt-2">
                            {daysUntilGame <= 0
                              ? t("game.today", { defaultValue: "Today" })
                              : daysUntilGame === 1
                                ? t("game.tomorrow", { defaultValue: "Tomorrow" })
                                : t("game.inDays", {
                                    count: daysUntilGame,
                                    defaultValue: `In ${daysUntilGame} days`,
                                  })}
                          </p>
                        )}
                      </div>
                    </div>
                    <Calendar className="h-8 w-8 text-purple-400 flex-shrink-0" />
                  </div>
                </div>
              )}
            </div>

            {/* Login CTA - Only show for unauthenticated users */}
            {!currentUser && (
              <div className="bg-gradient-to-r from-primary/5 to-secondary/5 rounded-lg p-8 border-2 border-primary/20 text-center">
                <Clock className="h-12 w-12 mx-auto mb-4 text-primary" />
                <h2 className="text-xl font-bold text-gray-900 mb-3">
                  {t("game.readyToJoin", {
                    defaultValue: "Ready to join the game?",
                  })}
                </h2>
                <p className="text-gray-600 mb-6">
                  {t("game.loginToParticipate", {
                    defaultValue:
                      "Log in to see all details, join the game, and connect with other players.",
                  })}
                </p>
                <Button
                  onClick={() => navigate("/login")}
                  className="bg-primary hover:bg-primary/90"
                >
                  {t("common.login", { defaultValue: "Login" })}
                </Button>
              </div>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}
