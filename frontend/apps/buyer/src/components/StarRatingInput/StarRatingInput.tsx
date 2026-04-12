"use client";

import { useState } from "react";
import { StarRatingInputPresenter } from "./StarRatingInput.presenter";

export interface StarRatingInputProps {
  value: number;
  onChange: (value: number) => void;
  maxStars?: number;
  size?: "md" | "lg";
  disabled?: boolean;
}

export default function StarRatingInput({
  value,
  onChange,
  maxStars,
  size,
  disabled,
}: StarRatingInputProps) {
  const [hoverValue, setHoverValue] = useState<number | null>(null);

  return (
    <StarRatingInputPresenter
      value={value}
      hoverValue={hoverValue}
      onChange={onChange}
      onHover={setHoverValue}
      maxStars={maxStars}
      size={size}
      disabled={disabled}
    />
  );
}
