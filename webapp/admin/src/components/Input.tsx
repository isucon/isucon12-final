import React, { FC, ReactNode } from "react";

interface inputTextProps {
  placeholder: string;
  name: string;
  value: string;
  handleChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
}

export const InputText = (prop: inputTextProps) => (
  <input
    className="input-text"
    name={prop.name}
    type="text"
    value={prop.value}
    placeholder={prop.placeholder}
    onChange={prop.handleChange}
  />
);

export const InputPassword = (props: inputTextProps) => (
  <input
    className="input-text"
    name={props.name}
    type="password"
    value={props.value}
    placeholder={props.placeholder}
    onChange={props.handleChange}
  />
);

interface inputSelectProps {
  name: string;
  value: string;
  handleChange: any;
  style?: React.CSSProperties;
  children: ReactNode;
}

export const InputSelect: FC<inputSelectProps> = props => (
  <select
    className="input-select"
    name={props.name}
    value={props.value}
    defaultValue=""
    onChange={props.handleChange}
    style={props.style}>
    {props.children}
  </select>
);

interface inputOptionProps {
  value: string;
  children: ReactNode;
}

export const InputOption: FC<inputOptionProps> = ({ value, children }) => <option value={value}>{children}</option>;
