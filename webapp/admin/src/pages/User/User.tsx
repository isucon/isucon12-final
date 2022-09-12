import React, { useEffect, useState } from "react";
import { DangerButton, PrimaryButton } from "../../components/Button";
import { InputOption, InputSelect, InputText } from "../../components/Input";
import { TableList } from "../../components/TableList";
import Header from "../../parts/Header/header";
import { adminBanUser, adminGetUser, adminGetUserResponse } from "../../services/admin";

interface userPageProps {
  sessionId: string;
  handleSetSessionId: (sessionId: string) => void;
}

const User = ({ sessionId, handleSetSessionId }: userPageProps) => {
  const [inputValue, setInputValue] = useState({
    userId: "",
    data: "user",
  });
  const [userInfo, setUserInfo] = useState<adminGetUserResponse>();
  const [isLoading, setIsLoading] = useState<boolean>(false);

  useEffect(() => {
    const sessId = localStorage.getItem("admin-session");
    if (sessId === null) {
      window.alert("session not found");
      window.location.replace("/");
      return;
    }

    handleSetSessionId(sessId);
  }, []);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setInputValue({ ...inputValue, [name]: value });
  };

  const handleSelectChange = (e: { target: HTMLSelectElement }) => {
    const { name, value } = e.target;
    setInputValue({ ...inputValue, [name]: value });
  };

  const handleSubmitGetUser = async () => {
    const userId = inputValue.userId;

    if (userId === "") {
      window.alert("userID is empty");
      return;
    }

    const masterVersion = localStorage.getItem("master-version");
    if (masterVersion === null) {
      window.alert("Failed to get master version");
      return;
    }

    try {
      setIsLoading(true);
      const res = await adminGetUser(masterVersion, sessionId, userId);
      setUserInfo(res);
      setIsLoading(false);
    } catch (e) {
      window.alert(`Failed to get user: ${e.message}`);
    }
  };

  const handleSubmitBanUser = async () => {
    const userId = inputValue.userId;

    if (userId === "") {
      window.alert("userID is empty");
      return;
    }

    const masterVersion = localStorage.getItem("master-version");
    if (masterVersion === null) {
      window.alert("Failed to get master version");
      return;
    }

    if (!window.confirm(`${userId}をBANします。よろしいですか？`)) {
      return;
    }

    try {
      setIsLoading(true);
      const res = await adminBanUser(masterVersion, sessionId, userId);
      window.alert(`${userId}をBANしました`);
      setIsLoading(false);
    } catch (e) {
      window.alert(`Failed to ban user: ${e.message}`);
    }
  };

  return (
    <>
      <Header activeTab="user" sessionId={sessionId} handleSetSessionId={handleSetSessionId} />

      <div style={{ padding: "10px" }}>
        <div>
          <InputText
            name="userId"
            placeholder="userId ex)12345"
            value={inputValue.userId}
            handleChange={handleChange}
          />

          <div style={{ marginLeft: "10px", display: "inline-block" }}>
            <PrimaryButton text="Get" handleClick={handleSubmitGetUser} disabled={isLoading} />
          </div>

          <div style={{ marginLeft: "10px", display: "inline-block" }}>
            <DangerButton text="BAN" handleClick={handleSubmitBanUser} />
          </div>
        </div>

        <div>
          <InputSelect name="data" value={inputValue.data} handleChange={handleSelectChange} style={{ width: "300px" }}>
            <InputOption value="user">user</InputOption>
            <InputOption value="userDevices">user devices</InputOption>
            <InputOption value="userCards">user cards</InputOption>
            <InputOption value="userDecks">user decks</InputOption>
            <InputOption value="userItems">user items</InputOption>
            <InputOption value="userLoginBonuses">user login bonuses</InputOption>
            <InputOption value="userPresents">user presents</InputOption>
            <InputOption value="userPresentAllReceivedHistory">user present all received history</InputOption>
          </InputSelect>
        </div>
      </div>

      <div style={{ padding: "10px" }}>
        {isLoading ? (
          <span>Loading...</span>
        ) : (
          <>
            {inputValue.data === "user" && typeof userInfo?.user !== "undefined" ? (
              <TableList list={[userInfo.user]} />
            ) : (
              <></>
            )}
            {inputValue.data === "userDevices" && typeof userInfo?.userDevices !== "undefined" ? (
              <TableList list={userInfo.userDevices} />
            ) : (
              <></>
            )}
            {inputValue.data === "userCards" && typeof userInfo?.userCards !== "undefined" ? (
              <TableList list={userInfo.userCards} />
            ) : (
              <></>
            )}
            {inputValue.data === "userDecks" && typeof userInfo?.userDecks !== "undefined" ? (
              <TableList list={userInfo.userDecks} />
            ) : (
              <></>
            )}
            {inputValue.data === "userItems" && typeof userInfo?.userItems !== "undefined" ? (
              <TableList list={userInfo.userItems} />
            ) : (
              <></>
            )}
            {inputValue.data === "userLoginBonuses" && typeof userInfo?.userLoginBonuses !== "undefined" ? (
              <TableList list={userInfo.userLoginBonuses} />
            ) : (
              <></>
            )}
            {inputValue.data === "userPresents" && typeof userInfo?.userPresents !== "undefined" ? (
              <TableList list={userInfo.userPresents} />
            ) : (
              <></>
            )}
            {inputValue.data === "userPresentAllReceivedHistory" &&
            typeof userInfo?.userPresentAllReceivedHistory !== "undefined" ? (
              <TableList list={userInfo.userPresentAllReceivedHistory} />
            ) : (
              <></>
            )}
          </>
        )}
      </div>
    </>
  );
};

export default User;
