import { useEffect, useState } from "react";
import Header from "../../parts/Header/header";
import { PrimaryButton } from "../../components/Button";
import { InputOption, InputSelect } from "../../components/Input";
import { TableList } from "../../components/TableList";
import { adminListMaster, listMasterResponse } from "../../services/admin";

interface masterListPageProps {
  sessionId: string;
  handleSetSessionId: (sessionId: string) => void;
}

const MasterList = ({ sessionId, handleSetSessionId }: masterListPageProps) => {
  const [inputValue, setInputValue] = useState({
    masterName: "version",
  });
  const [masterList, setMasterList] = useState<listMasterResponse>();
  const [isLoading, setIsLoading] = useState<boolean>();

  useEffect(() => {
    const sessId = localStorage.getItem("admin-session");
    if (sessId === null) {
      window.alert("session not found");
      window.location.replace("/");
      return;
    }

    handleSetSessionId(sessId);
  }, []);

  const handleSelectChange = (e: { target: HTMLSelectElement }) => {
    const { name, value } = e.target;
    setInputValue({ ...inputValue, [name]: value });
  };

  const handleGetListMaster = async () => {
    const masterVersion = localStorage.getItem("master-version");
    if (masterVersion === null) {
      window.alert("Failed to get master version");
      return;
    }

    try {
      setIsLoading(true);
      const res = await adminListMaster(masterVersion, sessionId);
      setMasterList(res);
      setIsLoading(false);
    } catch (e) {
      window.alert(`Failed to get list master: ${e.message}`);
    }
  };

  return (
    <>
      <Header activeTab="masterList" sessionId={sessionId} />

      <div style={{ padding: "10px" }}>
        <InputSelect name="masterName" value={inputValue.masterName} handleChange={handleSelectChange}>
          <InputOption value="version">version</InputOption>
          <InputOption value="item">item</InputOption>
          <InputOption value="gacha">gacha</InputOption>
          <InputOption value="gachaItem">gacha item</InputOption>
          <InputOption value="presentAll">present all</InputOption>
          <InputOption value="loginBonusReward">login bonus reward</InputOption>
          <InputOption value="loginBonus">login bonus</InputOption>
        </InputSelect>

        <div style={{ marginLeft: "10px", display: "inline-block" }}>
          <PrimaryButton text="Get" handleClick={handleGetListMaster} />
        </div>
      </div>

      <div style={{ padding: "10px" }}>
        {inputValue.masterName === "version" && typeof masterList?.versionMaster !== "undefined" ? (
          <TableList list={masterList.versionMaster} />
        ) : (
          <></>
        )}
        {inputValue.masterName === "item" && typeof masterList?.items !== "undefined" ? (
          <TableList list={masterList.items} />
        ) : (
          <></>
        )}
        {inputValue.masterName === "gacha" && typeof masterList?.gachas !== "undefined" ? (
          <TableList list={masterList.gachas} />
        ) : (
          <></>
        )}
        {inputValue.masterName === "gachaItem" && typeof masterList?.gachaItems !== "undefined" ? (
          <TableList list={masterList.gachaItems} />
        ) : (
          <></>
        )}
        {inputValue.masterName === "presentAll" && typeof masterList?.presentAlls !== "undefined" ? (
          <TableList list={masterList.presentAlls} />
        ) : (
          <></>
        )}
        {inputValue.masterName === "loginBonusReward" && typeof masterList?.loginBonusRewards !== "undefined" ? (
          <TableList list={masterList.loginBonusRewards} />
        ) : (
          <></>
        )}
        {inputValue.masterName === "loginBonus" && typeof masterList?.loginBonuses !== "undefined" ? (
          <TableList list={masterList.loginBonuses} />
        ) : (
          <></>
        )}
      </div>
    </>
  );
};

export default MasterList;
