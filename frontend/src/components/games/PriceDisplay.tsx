import { useTranslation } from "react-i18next";

interface PriceDisplayProps {
  totalPriceCents?: number;
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
  maxPlayers,
  className,
}: PriceDisplayProps) {
  const total =
    typeof totalPriceCents === "number" ? totalPriceCents : undefined;
  const players = typeof maxPlayers === "number" ? maxPlayers : undefined;

  const { t } = useTranslation();
  // Special case for free games
  if (total === 0) {
    return (
      <div className={className}>
        <span className="font-medium text-gray-900">{t("game.free")}</span>
      </div>
    );
  }

  let perPlayer: number | undefined;
  if (typeof total === "number" && typeof players === "number" && players > 0) {
    perPlayer = Math.round(total / players);
  }

  return (
    <div className={className}>
      <span className="font-medium text-gray-900">
        {perPlayer !== undefined ? formatCentsAsDollars(perPlayer) : "n/a"}
      </span>
      <span className="text-gray-500">
        {" "}
        ({formatCentsAsDollars(total)} total)
      </span>
    </div>
  );
}
