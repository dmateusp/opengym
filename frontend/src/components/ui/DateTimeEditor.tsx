import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Locale } from "date-fns";
import { ChevronDownIcon } from "lucide-react";
import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

export type DateTimeEditorProps = {
  value?: string;
  onChange: (value: string) => void;
  locale: Locale;
  dateLabel?: string;
  timeLabel?: string;
};

export function DateTimeEditor({
  value,
  onChange,
  locale,
  dateLabel = "Date",
  timeLabel = "Time",
}: DateTimeEditorProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const selectedDate = parseLocalDateTime(value);
  const timePart = extractTimePart(value) ?? "12:00";

  return (
    <div className="flex gap-4">
      <div className="flex flex-col gap-3">
        <Label htmlFor="date-picker" className="px-1">
          {dateLabel}
        </Label>
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              id="date-picker"
              className="w-40 justify-between font-normal"
            >
              {selectedDate ? selectedDate.toLocaleDateString(locale.code) : t("common.selectDate")}
              <ChevronDownIcon className="h-4 w-4" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-auto p-0" align="start">
            <Calendar
              mode="single"
              selected={selectedDate}
              defaultMonth={selectedDate ?? new Date()}
              captionLayout="dropdown"
              onSelect={(date) => {
                if (!date) return;
                onChange(buildLocalDateTimeString(date, timePart));
                setOpen(false);
              }}
              locale={locale}
            />
          </PopoverContent>
        </Popover>
      </div>
      <div className="flex flex-col gap-3">
        <Label htmlFor="time-picker" className="px-1">
          {timeLabel}
        </Label>
        <Input
          type="time"
          id="time-picker"
          value={timePart}
          className="w-32 bg-background appearance-none [&::-webkit-calendar-picker-indicator]:hidden [&::-webkit-calendar-picker-indicator]:appearance-none"
          onChange={(e) => {
            const date = selectedDate ?? new Date();
            onChange(buildLocalDateTimeString(date, e.target.value || "00:00"));
          }}
        />
      </div>
    </div>
  );
}

function parseLocalDateTime(val?: string) {
  if (!val) return undefined;
  const [datePart, timePart] = val.split("T");
  if (!datePart) return undefined;
  const [year, month, day] = datePart.split("-").map((n) => Number.parseInt(n, 10));
  const [hours, minutes] = (timePart || "00:00").split(":").map((n) => Number.parseInt(n, 10));
  if ([year, month, day].some((n) => Number.isNaN(n))) return undefined;
  const d = new Date(year, (month ?? 1) - 1, day ?? 1, hours || 0, minutes || 0);
  return Number.isNaN(d.getTime()) ? undefined : d;
}

function extractTimePart(val?: string) {
  if (!val) return undefined;
  const parts = val.split("T");
  if (parts.length < 2) return undefined;
  const time = parts[1].slice(0, 5);
  if (!/^\d{2}:\d{2}$/.test(time)) return undefined;
  return time;
}

function buildLocalDateTimeString(date: Date, time: string) {
  const pad = (n: number) => String(n).padStart(2, "0");
  const [hours, minutes] = (time || "00:00").split(":");
  const hh = pad(Number.parseInt(hours || "0", 10));
  const mm = pad(Number.parseInt(minutes || "0", 10));
  const yyyy = date.getFullYear();
  const mon = pad(date.getMonth() + 1);
  const dd = pad(date.getDate());
  return `${yyyy}-${mon}-${dd}T${hh}:${mm}`;
}
