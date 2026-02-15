import { Pencil } from "lucide-react";

export interface EditableFieldDisplayProps {
  isEditing: boolean;
  isEditable: boolean;
  children: React.ReactNode;
  onClick?: () => void;
}

export function EditableFieldDisplay({
  isEditing,
  isEditable,
  children,
  onClick,
}: EditableFieldDisplayProps) {
  return (
    <div
      onClick={onClick}
      className={`flex items-center gap-2 group transition ${
        isEditable && !isEditing ? "cursor-text" : ""
      }`}
    >
      <div className="flex-1">{children}</div>
      {isEditable && !isEditing && (
        <Pencil className="h-4 w-4 text-gray-400 opacity-30 group-hover:opacity-100 transition-opacity flex-shrink-0" />
      )}
    </div>
  );
}
