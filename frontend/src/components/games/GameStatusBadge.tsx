import { CheckCircle2, Clock, CircleDashed } from "lucide-react";
import { TimeDisplay } from "@/components/ui/TimeDisplay";

interface GameStatusBadgeProps {
  state: "published" | "scheduled" | "draft";
  publishedAt?: Date;
}

export function GameStatusBadge({ state, publishedAt }: GameStatusBadgeProps) {
  switch (state) {
    case "published":
      return (
        <span className="inline-flex items-center gap-2 text-sm font-semibold px-3 py-1.5 rounded-full bg-success/10 text-success">
          <CheckCircle2 className="h-4 w-4" />
          Published
        </span>
      );
    case "scheduled":
      return (
        <span className="inline-flex items-center gap-1 text-sm font-semibold px-3 py-1.5 rounded-full bg-amber-50 text-amber-700">
          <Clock className="h-4 w-4" />
          Publishing
          {publishedAt ? (
            <span>
              on{" "}
              <TimeDisplay
                timestamp={publishedAt.toISOString()}
                displayFormat="friendly"
              />
            </span>
          ) : (
            " soon"
          )}
        </span>
      );
    case "draft":
      return (
        <span className="inline-flex items-center gap-2 text-sm font-semibold px-3 py-1.5 rounded-full bg-gray-100 text-gray-600">
          <CircleDashed className="h-4 w-4" />
          Draft
        </span>
      );
  }
}
