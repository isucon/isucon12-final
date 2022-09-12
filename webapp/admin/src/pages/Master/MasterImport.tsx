import React, { useEffect, useRef, useState } from "react";
import Header from "../../parts/Header/header";
import { DangerButton, PrimaryButton } from "../../components/Button";
import { TableList } from "../../components/TableList";
import { adminUpdateMaster } from "../../services/admin";

interface masterImportPageProps {
  sessionId: string;
  handleSetSessionId: (sessionId: string) => void;
}

const MasterImport = ({ sessionId, handleSetSessionId }: masterImportPageProps) => {
  const versionMasterRef = useRef(null);
  const itemMasterRef = useRef(null);
  const gachaMasterRef = useRef(null);
  const gachaItemMasterRef = useRef(null);
  const presentAllMasterRef = useRef(null);
  const loginBonusRewardMasterRef = useRef(null);
  const loginBonusMasterRef = useRef(null);
  const [fileName, setFileName] = useState({
    version: "",
    item: "",
    gacha: "",
    gachaItem: "",
    presentAll: "",
    loginBonusReward: "",
    loginBonus: "",
  });
  const [file, setFile] = useState({
    version: null,
    item: null,
    gacha: null,
    gachaItem: null,
    presentAll: null,
    loginBonusReward: null,
    loginBonus: null,
  });

  useEffect(() => {
    const sessId = localStorage.getItem("admin-session");
    if (sessId === null) {
      window.alert("session not found");
      window.location.replace("/");
      return;
    }

    handleSetSessionId(sessId);
  }, []);

  const handleClick = async () => {
    if (Object.values(file).every(val => val === null)) {
      window.alert("マスタを登録してください");
      return;
    }

    const masterVersion = localStorage.getItem("master-version");
    if (masterVersion === null) {
      window.alert("Failed to get master version");
      return;
    }

    try {
      const res = await adminUpdateMaster(
        masterVersion,
        sessionId,
        file.version,
        file.item,
        file.gacha,
        file.gachaItem,
        file.presentAll,
        file.loginBonusReward,
        file.loginBonus
      );
      localStorage.setItem("master-version", res.versionMaster.masterVersion);
      window.alert(`Success: active master is ${res.versionMaster.masterVersion}`);
    } catch (e) {
      window.alert(`Failed to get user: ${e.message}`);
    }
  };

  const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFileName({ ...fileName, [e.target.name]: e.target.files[0].name });
    setFile({ ...file, [e.target.name]: e.target.files[0] });
  };

  const fileUpload = (ref: React.MutableRefObject<any>) => {
    ref.current.click();
  };

  const fileClear = (ref: React.MutableRefObject<any>) => {
    ref.current.value = null;
    setFileName({ ...fileName, [ref.current.name]: null });
    setFile({ ...file, [ref.current.name]: null });
  };

  return (
    <>
      <Header activeTab="masterImport" sessionId={sessionId} handleSetSessionId={handleSetSessionId} />

      <div style={{ padding: "10px" }}>
        <PrimaryButton text="Import" handleClick={handleClick} />
      </div>

      <div style={{ padding: "10px" }}>
        <TableList
          list={[
            {
              dataName: "version",
              file: fileName.version,
              uploadBtn: (
                <>
                  <PrimaryButton text="upload" handleClick={() => fileUpload(versionMasterRef)} />
                  <DangerButton
                    text="clear"
                    handleClick={() => fileClear(versionMasterRef)}
                    style={{ marginLeft: "15px" }}
                  />
                </>
              ),
            },
            {
              dataName: "item",
              file: fileName.item,
              uploadBtn: (
                <>
                  <PrimaryButton text="upload" handleClick={() => fileUpload(itemMasterRef)} />
                  <DangerButton
                    text="clear"
                    handleClick={() => fileClear(itemMasterRef)}
                    style={{ marginLeft: "15px" }}
                  />
                </>
              ),
            },
            {
              dataName: "gacha",
              file: fileName.gacha,
              uploadBtn: (
                <>
                  <PrimaryButton text="upload" handleClick={() => fileUpload(gachaMasterRef)} />
                  <DangerButton
                    text="clear"
                    handleClick={() => fileClear(gachaMasterRef)}
                    style={{ marginLeft: "15px" }}
                  />
                </>
              ),
            },
            {
              dataName: "gachaItem",
              file: fileName.gachaItem,
              uploadBtn: (
                <>
                  <PrimaryButton text="upload" handleClick={() => fileUpload(gachaItemMasterRef)} />
                  <DangerButton
                    text="clear"
                    handleClick={() => fileClear(gachaItemMasterRef)}
                    style={{ marginLeft: "15px" }}
                  />
                </>
              ),
            },
            {
              dataName: "presentAll",
              file: fileName.presentAll,
              uploadBtn: (
                <>
                  <PrimaryButton text="upload" handleClick={() => fileUpload(presentAllMasterRef)} />
                  <DangerButton
                    text="clear"
                    handleClick={() => fileClear(presentAllMasterRef)}
                    style={{ marginLeft: "15px" }}
                  />
                </>
              ),
            },
            {
              dataName: "loginBonus",
              file: fileName.loginBonus,
              uploadBtn: (
                <>
                  <PrimaryButton text="upload" handleClick={() => fileUpload(loginBonusMasterRef)} />
                  <DangerButton
                    text="clear"
                    handleClick={() => fileClear(loginBonusMasterRef)}
                    style={{ marginLeft: "15px" }}
                  />
                </>
              ),
            },
            {
              dataName: "loginBonusReward",
              file: fileName.loginBonusReward,
              uploadBtn: (
                <>
                  <PrimaryButton text="upload" handleClick={() => fileUpload(loginBonusRewardMasterRef)} />
                  <DangerButton
                    text="clear"
                    handleClick={() => fileClear(loginBonusRewardMasterRef)}
                    style={{ marginLeft: "15px" }}
                  />
                </>
              ),
            },
          ]}
        />
        {/* hidden file inputs */}
        <input hidden type="file" ref={itemMasterRef} name="item" accept="text/csv" onChange={handleFileInputChange} />
        <input
          hidden
          type="file"
          ref={versionMasterRef}
          name="version"
          accept="text/csv"
          onChange={handleFileInputChange}
        />
        <input
          hidden
          type="file"
          ref={gachaMasterRef}
          name="gacha"
          accept="text/csv"
          onChange={handleFileInputChange}
        />
        <input
          hidden
          type="file"
          ref={gachaItemMasterRef}
          name="gachaItem"
          accept="text/csv"
          onChange={handleFileInputChange}
        />
        <input
          hidden
          type="file"
          ref={presentAllMasterRef}
          name="presentAll"
          accept="text/csv"
          onChange={handleFileInputChange}
        />
        <input
          hidden
          type="file"
          ref={loginBonusMasterRef}
          name="loginBonus"
          accept="text/csv"
          onChange={handleFileInputChange}
        />
        <input
          hidden
          type="file"
          ref={loginBonusRewardMasterRef}
          name="loginBonusReward"
          accept="text/csv"
          onChange={handleFileInputChange}
        />
        {/* end hidden file inputs */}
      </div>
    </>
  );
};

export default MasterImport;
