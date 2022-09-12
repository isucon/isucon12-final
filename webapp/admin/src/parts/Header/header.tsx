import React, { useState } from "react";
import { Link } from "react-router-dom";
import { adminLogout } from "../../services/admin";
import titleLogoImg from "../../assets/img/title_logo.png";

interface homePageProps {
  activeTab: string;
  sessionId: string;
  handleSetSessionId?: (sessionId: string) => void;
}

const Header = ({ activeTab, sessionId, handleSetSessionId }: homePageProps) => {
  const handleLogout = async () => {
    const masterVersion = localStorage.getItem("master-version");
    if (masterVersion === null) {
      window.alert("Failed to get master version");
      return;
    }

    if (!window.confirm(`ログアウトします`)) {
      return;
    }

    try {
      const res = await adminLogout(masterVersion, sessionId);

      localStorage.removeItem("admin-session");
      localStorage.removeItem("master-version");
      handleSetSessionId("");

      window.location.replace("/");
    } catch (e) {
      window.alert(`Failed to logout: ${e.message}`);
    }
  };

  return (
    <div className="header" style={{ display: "flex", alignItems: "center" }}>
      <div className="logo">
        <Link to="/">
          <img width="64px" src={titleLogoImg} />
        </Link>
      </div>

      <ul className="nav">
        <li className="nav-item">
          <Link className={activeTab === "home" ? "nav-link active" : "nav-link"} to="/">
            Home
          </Link>
        </li>
        {sessionId === "" ? (
          <></>
        ) : (
          <>
            <li className="nav-item">
              <Link className={activeTab === "user" ? "nav-link active" : "nav-link"} to="/user">
                User
              </Link>
            </li>
            <li className="nav-item">
              <Link className={activeTab === "masterList" ? "nav-link active" : "nav-link"} to="/master/list">
                MasterList
              </Link>
            </li>
            <li className="nav-item">
              <Link className={activeTab === "masterImport" ? "nav-link active" : "nav-link"} to="/master/import">
                MasterImport
              </Link>
            </li>
            <li className="nav-item">
              <Link className="nav-link" to="#" onClick={handleLogout}>
                Logout
              </Link>
            </li>
          </>
        )}
      </ul>
    </div>
  );
};

export default Header;
