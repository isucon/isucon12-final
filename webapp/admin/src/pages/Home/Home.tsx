import React, { useState } from "react";
import { PrimaryButton } from "../../components/Button";
import { InputPassword, InputText } from "../../components/Input";
import Header from "../../parts/Header/header";
import { adminLogin } from "../../services/admin";

interface homePageProps {
  sessionId: string;
  handleSetSessionId: (sessionId: string) => void;
}

const Home = ({ sessionId, handleSetSessionId }: homePageProps) => {
  const [inputValue, setInputValue] = useState({
    userId: "",
    password: "",
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setInputValue({ ...inputValue, [name]: value });
  };

  const handleLoginSubmit = async () => {
    const { userId, password } = inputValue;

    if (userId === "" || password === "") {
      window.alert("userID or password is empty");
      return;
    }

    const masterVersion = localStorage.getItem("master-version");
    if (masterVersion === null) {
      window.alert("Failed to get master version");
      return;
    }

    try {
      const res = await adminLogin(masterVersion, { userId: Number(userId), password: password });
      localStorage.setItem("admin-session", res.session.sessionId);
      handleSetSessionId(res.session.sessionId);

      window.alert("Success login");
    } catch (e) {
      window.alert(`Failed to login: ${e.message}`);
    }
  };

  return (
    <>
      <Header activeTab="home" sessionId={sessionId} handleSetSessionId={handleSetSessionId} />

      <h1 style={{ marginLeft: "10px" }}>welcome to ISU CONQUEST admin</h1>

      <div style={{ padding: "10px" }}>
        <InputText placeholder="userId ex)12345" name="userId" value={inputValue.userId} handleChange={handleChange} />
      </div>
      <div style={{ padding: "10px" }}>
        <InputPassword placeholder="password" name="password" value={inputValue.password} handleChange={handleChange} />
      </div>
      <div style={{ padding: "10px" }}>
        <PrimaryButton text="Login" handleClick={handleLoginSubmit} />
      </div>
    </>
  );
};

export default Home;
