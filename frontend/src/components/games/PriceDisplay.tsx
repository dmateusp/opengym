import { useTranslation } from "react-i18next";

interface PriceDisplayProps {
  totalPriceCents?: number;
  currentPlayers?: number;
  maxPlayers?: number;
  className?: string;
}

function formatCentsAsDollars(cents?: number) {
  if (typeof cents !== "number" || Number.isNaN(cents)) return "—";
  if (cents === 0) return "0";
  const dollars = Math.floor(cents / 100);
  const centsRemainder = cents % 100;
  return `${dollars}.${String(centsRemainder).padStart(2, "0")}`;
}

export function PriceDisplay({
  totalPriceCents,
  currentPlayers,
  maxPlayers,
  className,
}: PriceDisplayProps) {
  const total =
    typeof totalPriceCents === "number" ? totalPriceCents : undefined;
  const current =
    typeof currentPlayers === "number" ? currentPlayers : undefined;
  const players = typeof maxPlayers === "number" ? maxPlayers : undefined;

  const { t } = useTranslation();
  // Special case for free games
  if (total === 0) {
    return (
      <div
        className={`inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50/60 px-3 py-1 ${className ?? ""}`}
      >
        <span className="font-semibold text-emerald-700/90">{t("game.free")}</span>
      </div>
    );
  }

  let perPlayerNow: number | undefined;
  if (typeof total === "number" && typeof current === "number" && current > 0) {
    perPlayerNow = Math.round(total / current);
  }

  let perPlayerFull: number | undefined;
  if (typeof total === "number" && typeof players === "number" && players > 0) {
    perPlayerFull = Math.round(total / players);
  }

  return (
    <div
      className={`rounded-lg border border-gray-200 bg-white p-3 ${className ?? ""}`}
    >
      <div className="mb-2 rounded-md border border-gray-200 bg-gray-50 px-3 py-2">
        <div className="text-[10px] font-semibold uppercase tracking-wide text-gray-500">
          {t("game.total", { defaultValue: "total" })}
        </div>
        <div className="text-xl font-bold leading-tight text-gray-900">
          {formatCentsAsDollars(total)}
        </div>
      </div>

      <div className="flex items-center justify-between gap-3">
        <span className="text-base font-semibold text-gray-900">
          {perPlayerNow !== undefined ? formatCentsAsDollars(perPlayerNow) : "n/a"}
        </span>
        <span className="text-xs text-gray-600">
          {t("game.perPersonNow", { defaultValue: "per person now" })}
        </span>
      </div>

      <div className="mt-1 flex items-center justify-between gap-3">
        <span className="text-base font-semibold text-gray-800">
          {perPlayerFull !== undefined
            ? formatCentsAsDollars(perPlayerFull)
            : t("common.notAvailable", { defaultValue: "n/a" })}
        </span>
        <span className="text-xs text-gray-600">
          {t("game.perPersonIfFull", {
            defaultValue: "per person if full",
          })}
        </span>
      </div>

      <div className="mt-2 h-px w-full bg-gray-100" />
      <div className="mt-2 text-[11px] text-gray-500">
        {perPlayerFull !== undefined
          ? t("game.costDependsOnAttendance", {
              defaultValue: "Final split depends on attendance",
            })
          : t("game.perPersonIfFullUnavailable", {
              defaultValue: "n/a per person if full",
            })}
      </div>
    </div>
  );
}
