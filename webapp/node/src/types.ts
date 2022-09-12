// DB row types
export interface AdminUserRow {
  id: number
  password: string
  last_activated_at: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface SessionRow {
  id: number
  user_id: number
  session_id: string
  expired_at: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserRow {
  id: number
  isu_coin: number
  last_getreward_at: number
  last_activated_at: number
  registered_at: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserBanRow {
  id: number
  user_id: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserOneTimeTokenRow {
  id: number
  user_id: number
  token: string
  token_type: number
  expired_at: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserDeviceRow {
  id: number
  user_id: number
  platform_id: string
  platform_type: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserCardRow {
  id: number
  user_id: number
  card_id: number
  amount_per_sec: number
  level: number
  total_exp: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserDeckRow {
  id: number
  user_id: number
  user_card_id_1: number
  user_card_id_2: number
  user_card_id_3: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserItemRow {
  id: number
  user_id: number
  item_type: number
  item_id: number
  amount: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserLoginBonusRow {
  id: number
  user_id: number
  login_bonus_id: number
  last_reward_sequence: number
  loop_count: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserPresentRow {
  id: number
  user_id: number
  sent_at: number
  item_type: number
  item_id: number
  amount: number
  present_message: string
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface UserPresentAllReceivedHistoryRow {
  id: number
  user_id: number
  present_all_id: number
  received_at: number
  created_at: number
  updated_at: number
  deleted_at?: number
}

export interface ItemMasterRow {
  id: number
  item_type: number
  name: string
  description: string
  amount_per_sec: number
  max_level: number
  max_amount_per_sec: number
  base_exp_per_level: number
  gained_exp: number
  shortening_min: number
}

export interface GachaMasterRow {
  id: number
  name: string
  start_at: number
  end_at: number
  display_order: number
  created_at: number
}

export interface GachaItemMasterRow {
  id: number
  gacha_id: number
  item_type: number
  item_id: number
  amount: number
  weight: number
  created_at: number
}

export interface PresentAllMasterRow {
  id: number
  registered_start_at: number
  registereed_end_at: number
  item_type: number
  item_id: number
  amount: number
  present_message: string
  created_at: number
}

export interface LoginBonusMasterRow {
  id: number
  start_at: number
  end_at: number
  column_count: number
  looped: number
  created_at: number
}

export interface LoginBonusRewardMasterRow {
  id: number
  login_bonus_id: number
  reward_sequence: number
  item_type: number
  item_id: number
  amount: number
  created_at: number
}

export interface VersionMasterRow {
  id: number
  status: number
  master_version: string
}

// JSON types
export type Session = {
  id: number
  userId: number
  sessionId: string
  expiredAt: number
  createdAt: number
  updatedAt: number
  deletedAt?: number
}

export type User = {
  id: number
  isuCoin: number
  lastGetRewardAt: number
  lastActivatedAt: number
  registeredAt: number
  createdAt: number
  updatedAt: number
  deletedAt?: number
}

export function toUser(row: UserRow): User {
  return {
    id: row.id,
    isuCoin: row.isu_coin,
    lastGetRewardAt: row.last_getreward_at,
    lastActivatedAt: row.last_activated_at,
    registeredAt: row.registered_at,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    deletedAt: row.deleted_at,
  }
}

export type UserDevice = {
  id: number
  userId: number
  platformId: string
  platformType: number
  createdAt: number
  updatedAt: number
  deletedAt?: number
}

export function toUserDevice(row: UserDeviceRow): UserDevice {
  return {
    id: row.id,
    userId: row.user_id,
    platformId: row.platform_id,
    platformType: row.platform_type,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    deletedAt: row.deleted_at,
  }
}

export type UserCard = {
  id: number
  userId: number
  cardId: number
  amountPerSec: number
  level: number
  totalExp: number
  createdAt: number
  updatedAt: number
  deletedAt?: number
}

export function toUserCard(row: UserCardRow): UserCard {
  return {
    id: row.id,
    userId: row.user_id,
    cardId: row.card_id,
    amountPerSec: row.amount_per_sec,
    level: row.level,
    totalExp: row.total_exp,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    deletedAt: row.deleted_at,
  }
}

export type UserDeck = {
  id: number
  userId: number
  cardId1: number
  cardId2: number
  cardId3: number
  createdAt: number
  updatedAt: number
  deletedAt?: number
}

export function toUserDeck(row: UserDeckRow): UserDeck {
  return {
    id: row.id,
    userId: row.user_id,
    cardId1: row.user_card_id_1,
    cardId2: row.user_card_id_2,
    cardId3: row.user_card_id_3,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    deletedAt: row.deleted_at,
  }
}

export type UserItem = {
  id: number
  userId: number
  itemType: number
  itemId: number
  amount: number
  createdAt: number
  updatedAt: number
  deletedAt?: number
}

export function toUserItem(row: UserItemRow): UserItem {
  return {
    id: row.id,
    userId: row.user_id,
    itemType: row.item_type,
    itemId: row.item_id,
    amount: row.amount,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    deletedAt: row.deleted_at,
  }
}

export type UserLoginBonus = {
  id: number
  userId: number
  loginBonusId: number
  lastRewardSequence: number
  loopCount: number
  createdAt: number
  updatedAt: number
  deletedAt?: number
}

export function toUserLoginBonus(row: UserLoginBonusRow): UserLoginBonus {
  return {
    id: row.id,
    userId: row.user_id,
    loginBonusId: row.login_bonus_id,
    lastRewardSequence: row.last_reward_sequence,
    loopCount: row.loop_count,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    deletedAt: row.deleted_at,
  }
}

export type UserPresent = {
  id: number
  userId: number
  sentAt: number
  itemType: number
  itemId: number
  amount: number
  presentMessage: string
  createdAt: number
  updatedAt: number
  deletedAt?: number
}

export function toUserPresent(row: UserPresentRow): UserPresent {
  return {
    id: row.id,
    userId: row.user_id,
    sentAt: row.sent_at,
    itemType: row.item_type,
    itemId: row.item_id,
    amount: row.amount,
    presentMessage: row.present_message,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    deletedAt: row.deleted_at,
  }
}

export type UserPresentAllReceivedHistory = {
  id: number
  userId: number
  presentAllId: number
  receivedAt: number
  createdAt: number
  updatedAt: number
  deletedAt?: number
}

export function toUserPresentAllReceivedHistory(row: UserPresentAllReceivedHistoryRow): UserPresentAllReceivedHistory {
  return {
    id: row.id,
    userId: row.user_id,
    presentAllId: row.present_all_id,
    receivedAt: row.received_at,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    deletedAt: row.deleted_at,
  }
}

export type ItemMaster = {
  id: number
  itemType: number
  name: string
  description: string
  amountPerSec: number
  maxLevel: number
  maxAmountPerSec: number
  baseExpPerLevel: number
  gainedExp: number
  shotteningMin: number // TODO typoがあるよね
}

export function toItemMaster(row: ItemMasterRow): ItemMaster {
  return {
    id: row.id,
    itemType: row.item_type,
    name: row.name,
    description: row.description,
    amountPerSec: row.amount_per_sec,
    maxLevel: row.max_level,
    maxAmountPerSec: row.max_amount_per_sec,
    baseExpPerLevel: row.base_exp_per_level,
    gainedExp: row.gained_exp,
    shotteningMin: row.shortening_min,
  }
}

export type GachaMaster = {
  id: number
  name: string
  startAt: number
  endAt: number
  displayOrder: number
  createdAt: number
}

export function toGachaMaster(row: GachaMasterRow): GachaMaster {
  return {
    id: row.id,
    name: row.name,
    startAt: row.start_at,
    endAt: row.end_at,
    displayOrder: row.display_order,
    createdAt: row.created_at,
  }
}

export type GachaItemMaster = {
  id: number
  gachaId: number
  itemType: number
  itemId: number
  amount: number
  weight: number
  createdAt: number
}

export function toGachaItemMaster(row: GachaItemMasterRow): GachaItemMaster {
  return {
    id: row.id,
    gachaId: row.gacha_id,
    itemType: row.item_type,
    itemId: row.item_id,
    amount: row.amount,
    weight: row.weight,
    createdAt: row.created_at,
  }
}

export type PresentAllMaster = {
  id: number
  registeredStartAt: number
  registereedEndAt: number
  itemType: number
  itemId: number
  amount: number
  presentMessage: string
  createdAt: number
}

export function toPresentAllMaster(row: PresentAllMasterRow): PresentAllMaster {
  return {
    id: row.id,
    registeredStartAt: row.registered_start_at,
    registereedEndAt: row.registereed_end_at,
    itemType: row.item_type,
    itemId: row.item_id,
    amount: row.amount,
    presentMessage: row.present_message,
    createdAt: row.created_at,
  }
}

export type LoginBonusMaster = {
  id: number
  startAt: number
  endAt: number
  columnCount: number
  looped: boolean
  createdAt: number
}

export function toLoginBonusMaster(row: LoginBonusMasterRow): LoginBonusMaster {
  return {
    id: row.id,
    startAt: row.start_at,
    endAt: row.end_at,
    columnCount: row.column_count,
    looped: !!row.looped,
    createdAt: row.created_at,
  }
}

export type LoginBonusRewardMaster = {
  id: number
  loginBonusId: number
  rewardSequence: number
  itemType: number
  itemId: number
  amount: number
  createdAt: number
}

export function toLoginBonusRewardMaster(row: LoginBonusRewardMasterRow): LoginBonusRewardMaster {
  return {
    id: row.id,
    loginBonusId: row.login_bonus_id,
    rewardSequence: row.reward_sequence,
    itemType: row.item_type,
    itemId: row.item_id,
    amount: row.amount,
    createdAt: row.created_at,
  }
}

export type VersionMaster = {
  id: number
  status: number
  masterVersion: string
}

export function toVersionMaster(row: VersionMasterRow): VersionMaster {
  return {
    id: row.id,
    status: row.status,
    masterVersion: row.master_version,
  }
}
