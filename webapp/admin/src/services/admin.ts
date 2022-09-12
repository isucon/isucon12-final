const { VITE_API_URL } = import.meta.env;

interface session {
  id: number;
  userId: number;
  sessionId: string;
  expiredAt: string;
  createdAt: string;
  updatedAt: string;
  deletedAt: string;
}

export interface errorResponse {
  status_code: number;
  message: string;
}

interface adminLoginRequest {
  userId: number;
  password: string;
}

interface adminLoginResponse {
  session: session;
}

// adminLogin 管理者ログイン
export async function adminLogin(masterVersion: string, body: adminLoginRequest): Promise<adminLoginResponse> {
  const res = await fetch(`${VITE_API_URL}/admin/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    mode: "cors",
    body: JSON.stringify(body),
  });

  if (res.status !== 200) {
    throw await res.json();
  }

  return res.json();
}

// adminLogout 管理者ログアウト
export async function adminLogout(masterVersion: string, sessionId: string) {
  const res = await fetch(`${VITE_API_URL}/admin/logout`, {
    method: "DELETE",
    headers: {
      "x-session": sessionId,
    },
    mode: "cors",
  });

  if (res.status !== 204) {
    throw await res.json();
  }

  return;
}

export interface listMasterResponse {
  gachaItems: gachaItem[];
  gachas: gacha[];
  items: item[];
  loginBonuses: loginBonus[];
  loginBonusRewards: loginBonusReward[];
  presentAlls: presentAll[];
  versionMaster: versionMaster[];
}

// adminListMaster マスターデータ全取得
export async function adminListMaster(masterVersion: string, sessionId: string): Promise<listMasterResponse> {
  const res = await fetch(`${VITE_API_URL}/admin/master`, {
    method: "GET",
    headers: {
      "x-session": sessionId,
    },
    mode: "cors",
  });

  if (res.status !== 200) {
    throw await res.json();
  }

  return res.json();
}

interface adminUpdateMaster {
  versionMaster: versionMaster;
}

// adminUpdateMaster マスター更新
export async function adminUpdateMaster(
  masterVersion: string,
  sessionId: string,
  version: File | null,
  item: File | null,
  gacha: File | null,
  gachaItem: File | null,
  presentAll: File | null,
  loginBonusReward: File | null,
  loginBonus: File | null
): Promise<adminUpdateMaster> {
  const formData = new FormData();
  if (version !== null) {
    formData.append("versionMaster", version);
  }
  if (item !== null) {
    formData.append("itemMaster", item);
  }
  if (gacha !== null) {
    formData.append("gachaMaster", gacha);
  }
  if (gachaItem !== null) {
    formData.append("gachaItemMaster", gachaItem);
  }
  if (presentAll !== null) {
    formData.append("presentAllMaster", presentAll);
  }
  if (loginBonus !== null) {
    formData.append("loginBonusMaster", loginBonus);
  }
  if (loginBonusReward !== null) {
    formData.append("loginBonusRewardMaster", loginBonusReward);
  }

  const res = await fetch(`${VITE_API_URL}/admin/master`, {
    method: "PUT",
    headers: {
      "x-session": sessionId,
    },
    mode: "cors",
    body: formData,
  });

  if (res.status !== 200) {
    throw await res.json();
  }

  return res.json();
}

export interface adminGetUserResponse {
  user: user;
  userCards: userCard[];
  userDecks: userDeck[];
  userDevices: userDevice[];
  userItems: userItem[];
  userLoginBonuses: userLoginBonus[];
  userPresentAllReceivedHistory: userPresentAllReceivedHistory[];
  userPresents: userPresent[];
}

// adminGetUser ユーザ情報取得
export async function adminGetUser(
  masterVersion: string,
  sessionId: string,
  userId: string
): Promise<adminGetUserResponse> {
  const res = await fetch(`${VITE_API_URL}/admin/user/${userId}`, {
    method: "GET",
    headers: {
      "x-session": sessionId,
    },
    mode: "cors",
  });

  if (res.status !== 200) {
    throw await res.json();
  }

  return res.json();
}

export interface adminBanUserResponse {
  user: user;
}

// adminBanUser ユーザをBANする
export async function adminBanUser(
  masterVersion: string,
  sessionId: string,
  userId: string
): Promise<adminBanUserResponse> {
  const res = await fetch(`${VITE_API_URL}/admin/user/${userId}/ban`, {
    method: "POST",
    headers: {
      "x-session": sessionId,
      "Content-Type": "application/json",
    },
    mode: "cors",
  });

  if (res.status !== 200) {
    throw await res.json();
  }

  return res.json();
}

interface gachaItem {
  id: number;
  gachaId: number;
  itemId: number;
  itemType: number;
  weight: number;
  amount: number;
  createdAt: string;
}

interface gacha {
  id: number;
  name: string;
  displayOrder: number;
  startAt: string;
  endAt: string;
  createdAt: string;
}

interface item {
  id: number;
  name: string;
  amountPerSec: number;
  baseExpPerLevel: number;
  description: string;
  gainedExp: number;
  itemType: number;
  maxAmountPerSec: number;
  maxLevel: number;
  shorteningMin: number;
  // createdAt: string
}

interface loginBonusReward {
  id: number;
  amount: number;
  itemId: number;
  itemType: number;
  loginBonusId: number;
  rewardSequence: number;
  createdAt: string;
}

interface loginBonus {
  id: number;
  columnCount: number;
  looped: boolean;
  startAt: string;
  endAt: string;
  createdAt: string;
}

interface presentAll {
  id: number;
  amount: number;
  itemId: number;
  itemType: number;
  presentMessage: string;
  registeredStartAt: string;
  registeredEndAt: string;
  createdAt: string;
}

interface versionMaster {
  id: number;
  masterVersion: string;
  status: number;
  // createdAt: string
}

interface user {
  id: number;
  isCoin: number;
  registeredAt: string;
  lastActivatedAt: string;
  lastGetRewardAt: string;
  createdAt: string;
  updatedAt: string;
  deletedAt: string;
}

interface userCard {
  amountPerSec: number;
  cardId: number;
  createdAt: string;
  deletedAt: string;
  id: number;
  level: number;
  totalExp: number;
  updatedAt: string;
  userId: number;
}

interface userDeck {
  cardId1: number;
  cardId2: number;
  cardId3: number;
  createdAt: string;
  deletedAt: string;
  id: number;
  updatedAt: string;
  userId: number;
}

interface userDevice {
  id: number;
  userId: number;
  platformId: string;
  platformType: number;
  createdAt: string;
  updatedAt: string;
  deletedAt: string;
}

interface userItem {
  amount: number;
  createdAt: string;
  deletedAt: string;
  id: number;
  itemId: number;
  itemType: number;
  updatedAt: string;
  userId: number;
}

interface userLoginBonus {
  createdAt: string;
  deletedAt: string;
  id: number;
  lastRewardSequence: number;
  loginBonusId: number;
  loopCount: number;
  updatedAt: string;
  userId: number;
}

interface userPresentAllReceivedHistory {
  createdAt: string;
  deletedAt: string;
  id: number;
  presentAllId: number;
  receivedAt: string;
  updatedAt: string;
  userId: number;
}

interface userPresent {
  amount: number;
  createdAt: string;
  deletedAt: string;
  id: number;
  itemId: number;
  itemType: number;
  presentMessage: string;
  sentAt: string;
  updatedAt: string;
  userId: number;
}
