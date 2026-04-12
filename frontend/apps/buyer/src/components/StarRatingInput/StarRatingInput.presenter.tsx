"use client";

export interface StarRatingInputPresenterProps {
  value: number;
  hoverValue: number | null;
  onChange: (value: number) => void;
  onHover: (value: number | null) => void;
  maxStars?: number;
  size?: "md" | "lg";
  disabled?: boolean;
}

const sizeMap = { md: "w-7 h-7", lg: "w-9 h-9" } as const;

export function StarRatingInputPresenter({
  value,
  hoverValue,
  onChange,
  onHover,
  maxStars = 5,
  size = "md",
  disabled = false,
}: StarRatingInputPresenterProps) {
  const displayValue = hoverValue ?? value;
  const cls = sizeMap[size];

  return (
    <fieldset
      className="inline-flex items-center gap-0.5 border-0 p-0 m-0"
      onMouseLeave={() => onHover(null)}
      role="radiogroup"
    >
      {Array.from({ length: maxStars }, (_, i) => {
        const starValue = i + 1;
        const filled = starValue <= displayValue;
        return (
          <button
            key={i}
            type="button"
            disabled={disabled}
            onClick={() => onChange(starValue)}
            onMouseEnter={() => onHover(starValue)}
            className={`${cls} transition-colors ${disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer"}`}
            role="radio"
            aria-checked={starValue === value}
            aria-label={`${starValue}`}
          >
            <svg
              className={`${cls} ${filled ? "text-yellow-400" : "text-gray-300"}`}
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fillRule="evenodd"
                d="M10.868 2.884c-.321-.772-1.415-.772-1.736 0l-1.83 4.401-4.753.381c-.833.067-1.171 1.107-.536 1.651l3.62 3.102-1.106 4.637c-.194.813.691 1.456 1.405 1.02L10 15.591l4.069 2.485c.713.436 1.598-.207 1.404-1.02l-1.106-4.637 3.62-3.102c.635-.544.297-1.584-.536-1.65l-4.752-.382-1.831-4.401Z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        );
      })}
    </fieldset>
  );
}
