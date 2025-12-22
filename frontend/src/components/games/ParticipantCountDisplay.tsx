interface ParticipantCountDisplayProps {
  count: number;
  maxCount?: number;
  label: string;
  color?: 'primary' | 'accent' | 'gray';
  showDisabled?: boolean;
}

export function ParticipantCountDisplay({
  count,
  maxCount,
  label,
  color = 'primary',
  showDisabled = false,
}: ParticipantCountDisplayProps) {
  const colorClasses = {
    primary: 'text-primary',
    accent: 'text-accent',
    gray: 'text-gray-400',
  };

  const isDisabled = maxCount === 0;

  return (
    <div className="text-center">
      <div className={`text-2xl font-bold ${colorClasses[color]} mb-1`}>
        {isDisabled ? (
          <span className="text-gray-400">—</span>
        ) : (
          <>
            {count}
            {typeof maxCount === 'number' && maxCount > 0 ? `/${maxCount}` : ''}
          </>
        )}
      </div>
      <div className="text-xs text-gray-600 font-medium">
        {label}
        {showDisabled && isDisabled ? ' (Off)' : ''}
      </div>
    </div>
  );
}
