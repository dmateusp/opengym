import { useMemo, useState } from "react";
import { enUS, pt } from "date-fns/locale";
import { DateTimeEditor } from "./DateTimeEditor";

export default {
  title: "UI/DateTimeEditor",
};

const toLocalDateTime = (d: Date) => {
  const pad = (n: number) => String(n).padStart(2, "0");
  const yyyy = d.getFullYear();
  const mm = pad(d.getMonth() + 1);
  const dd = pad(d.getDate());
  const hh = pad(d.getHours());
  const mi = pad(d.getMinutes());
  return `${yyyy}-${mm}-${dd}T${hh}:${mi}`;
};

export const Playground = () => {
  const [value, setValue] = useState<string>(toLocalDateTime(new Date()));
  const [lang, setLang] = useState<"en" | "pt-PT">("en");

  const locale = useMemo(() => (lang === "pt-PT" ? pt : enUS), [lang]);

  const labels = lang === "pt-PT"
    ? { date: "Data", time: "Hora" }
    : { date: "Date", time: "Time" };

  return (
    <div className="p-8 space-y-6">
      <div className="flex items-center gap-3">
        <span className="text-sm text-gray-700">Language:</span>
        <button
          className={`px-3 py-1 rounded border ${lang === "en" ? "bg-blue-500 text-white" : "bg-white"}`}
          onClick={() => setLang("en")}
        >
          English
        </button>
        <button
          className={`px-3 py-1 rounded border ${lang === "pt-PT" ? "bg-blue-500 text-white" : "bg-white"}`}
          onClick={() => setLang("pt-PT")}
        >
          Português
        </button>
      </div>

      <p className="text-sm text-gray-700">Current value: {value || "(none)"}</p>

      <DateTimeEditor
        value={value}
        onChange={setValue}
        locale={locale}
        dateLabel={labels.date}
        timeLabel={labels.time}
      />
    </div>
  );
};
