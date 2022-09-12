import React from "react";

interface buttonProps {
  text: string;
  disabled?: boolean;
  handleClick: React.MouseEventHandler<HTMLButtonElement>;
  style?: React.CSSProperties;
}

export const PrimaryButton = ({ text, disabled, handleClick, style }: buttonProps) => {
  return (
    <>
      {disabled ? (
        <button className="btn btn-primary" style={style} onClick={handleClick} disabled>
          {text}
        </button>
      ) : (
        <button className="btn btn-primary" style={style} onClick={handleClick}>
          {text}
        </button>
      )}
    </>
  );
};

export const SecondaryButton = ({ text, handleClick, style }: buttonProps) => (
  <button className="btn btn-secondary" style={style} onClick={handleClick}>
    {text}
  </button>
);

export const DangerButton = ({ text, handleClick, style }: buttonProps) => (
  <button className="btn btn-danger" style={style} onClick={handleClick}>
    {text}
  </button>
);
