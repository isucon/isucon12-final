import { useEffect, useState } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import Home from "./pages/Home/Home";
import User from "./pages/User/User";
import MasterList from "./pages/Master/MasterList";
import MasterImport from "./pages/Master/MasterImport";

import "./assets/css/main.css";

const App = () => {
  const [sessionId, setSessionId] = useState("");

  const handleSetSessionId = (sessId: string) => {
    setSessionId(sessId);
  };

  useEffect(() => {
    localStorage.setItem("master-version", "1");
    const sessId = localStorage.getItem("admin-session");
    setSessionId(sessId === null ? "" : sessId);
  }, []);

  return (
    <div>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Home sessionId={sessionId} handleSetSessionId={handleSetSessionId} />} />
          <Route path="/user" element={<User sessionId={sessionId} handleSetSessionId={handleSetSessionId} />} />
          <Route
            path="/master/list"
            element={<MasterList sessionId={sessionId} handleSetSessionId={handleSetSessionId} />}
          />
          <Route
            path="/master/import"
            element={<MasterImport sessionId={sessionId} handleSetSessionId={handleSetSessionId} />}
          />
        </Routes>
      </BrowserRouter>
    </div>
  );
};

export default App;
