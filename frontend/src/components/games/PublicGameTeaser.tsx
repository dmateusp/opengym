import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Clock, Rocket } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import UserProfileMenu from "@/components/auth/UserProfileMenu";
import type { User } from "@/opengym/client";

interface Organizer {
  name?: string;
  picture?: string;
}

interface PublicGameTeaserProps {
  gameId: string;
  gameName: string;
  organizer: Organizer;
  publishedAt: string;
  onUserChange?: (user: User) => void;
  currentUser: User | null;
}

export function PublicGameTeaser({
  gameName,
  organizer,
  publishedAt,
  onUserChange,
  currentUser,
}: PublicGameTeaserProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [timeRemaining, setTimeRemaining] = useState<{
    days: number;
    hours: number;
    minutes: number;
    seconds: number;
  } | null>(null);

  useEffect(() => {
    const updateCountdown = () => {
      const now = Date.now();
      const publishTime = new Date(publishedAt).getTime();
      const diff = publishTime - now;

      if (diff <= 0) {
        setTimeRemaining(null);
        return;
      }

      const days = Math.floor(diff / (1000 * 60 * 60 * 24));
      const hours = Math.floor((diff / (1000 * 60 * 60)) % 24);
      const minutes = Math.floor((diff / 1000 / 60) % 60);
      const seconds = Math.floor((diff / 1000) % 60);

      setTimeRemaining({ days, hours, minutes, seconds });
    };

    updateCountdown();
    const interval = setInterval(updateCountdown, 1000);
    return () => clearInterval(interval);
  }, [publishedAt]);

  const getInitials = (name?: string) => {
    if (!name) return "?";
    const parts = name.split(" ").filter((p) => p.length > 0);
    return parts
      .slice(0, 2)
      .map((p) => p[0].toUpperCase())
      .join("");
  };

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

        {/* Teaser Card */}
        <Card className="overflow-hidden border-l-8 border-l-primary mb-8">
          <div className="p-8 text-center">
            {/* Coming Soon Badge */}
            <div className="inline-block mb-6">
              <div className="bg-gradient-to-r from-purple-100 to-blue-100 px-4 py-2 rounded-full">
                <p className="text-sm font-semibold text-purple-900">
                  {t("game.comingSoon", { defaultValue: "Coming Soon" })}
                </p>
              </div>
            </div>

            {/* Game Name */}
            <h1 className="text-4xl font-bold text-gray-900 mb-4">{gameName}</h1>

            {/* Organizer */}
            <div className="flex items-center justify-center gap-3 mb-8">
              <Avatar className="h-10 w-10">
                {organizer.picture && (
                  <AvatarImage src={organizer.picture} alt={organizer.name} />
                )}
                <AvatarFallback className="bg-primary text-white">
                  {getInitials(organizer.name)}
                </AvatarFallback>
              </Avatar>
              <div className="text-left">
                <p className="text-sm text-gray-600">
                  {t("game.organizedBy", { defaultValue: "Organized by" })}
                </p>
                <p className="font-semibold text-gray-900">
                  {organizer.name || "Unknown"}
                </p>
              </div>
            </div>

            {/* Countdown */}
            <div className="mb-8">
              <p className="text-lg text-gray-600 mb-6">
                {t("game.getsPublishedIn", {
                  defaultValue: "This game gets published in",
                })}
              </p>

              {timeRemaining ? (
                <div className="grid grid-cols-4 gap-3 mb-4">
                  {[
                    { value: timeRemaining.days, label: "Days" },
                    { value: timeRemaining.hours, label: "Hours" },
                    { value: timeRemaining.minutes, label: "Minutes" },
                    { value: timeRemaining.seconds, label: "Seconds" },
                  ].map((item, idx) => (
                    <div
                      key={idx}
                      className="bg-white rounded-lg border-2 border-gray-200 p-4"
                    >
                      <div className="text-3xl font-bold text-primary">
                        {String(item.value).padStart(2, "0")}
                      </div>
                      <div className="text-xs text-gray-600 uppercase tracking-wide mt-2">
                        {item.label}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="bg-gradient-to-r from-green-50 to-emerald-50 rounded-lg p-6 border-2 border-green-200">
                  <Rocket className="h-8 w-8 mx-auto mb-2 text-green-600" />
                  <p className="text-green-800 font-semibold">
                    {t("game.nowPublished", { defaultValue: "Now published!" })}
                  </p>
                </div>
              )}
            </div>

            {/* Call to Action */}
            {!currentUser && (
              <div className="bg-blue-50 rounded-lg p-6 border-2 border-blue-200">
                <Clock className="h-6 w-6 mx-auto mb-3 text-blue-600" />
                <p className="text-blue-900 mb-4">
                  {t("game.loginToJoinWhenPublished", {
                    defaultValue:
                      "Log in to be notified when this game is published and join right away!",
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
