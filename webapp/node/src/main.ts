import express, { Request, Response, NextFunction, RequestHandler } from 'express'
import cors from 'cors'
import morgan from 'morgan'
import mysql, { RowDataPacket } from 'mysql2/promise'
import childProcess from 'child_process'
import util from 'util'

const exec = util.promisify(childProcess.exec)

import { useGenerateId, successResponse, errorResponse, getRequestTime, generateUUID, getUserId } from './common'

import {
  Session,
  User,
  UserDevice,
  UserCard,
  UserDeck,
  UserItem,
  UserLoginBonus,
  UserPresent,
  GachaMaster,
  GachaItemMaster,
  SessionRow,
  UserRow,
  UserBanRow,
  UserDeviceRow,
  UserCardRow,
  UserDeckRow,
  UserItemRow,
  UserLoginBonusRow,
  UserPresentRow,
  UserPresentAllReceivedHistoryRow,
  UserOneTimeTokenRow,
  GachaItemMasterRow,
  GachaMasterRow,
  ItemMasterRow,
  LoginBonusMasterRow,
  LoginBonusRewardMasterRow,
  PresentAllMasterRow,
  VersionMasterRow,
  toUser,
  toUserCard,
  toUserDeck,
  toUserItem,
  toUserLoginBonus,
  toUserPresent,
  toGachaItemMaster,
  toGachaMaster,
} from './types'

import { useAdmin } from './admin'

const ErrInvalidRequestBody = 'invalid request body'
const ErrInvalidMasterVersion = 'invalid master version'
const ErrInvalidItemType = 'invalid item type'
const ErrInvalidToken = 'invalid token'
const ErrGetRequestTime = 'failed to get request time'
const ErrExpiredSession = 'session expired'
const ErrUserNotFound = 'not found user'
const ErrUserDeviceNotFound = 'not found user device'
const ErrItemNotFound = 'not found item'
const ErrLoginBonusRewardNotFound = 'not found login bonus reward'
const ErrUnauthorized = 'unauthorized user'
const ErrForbidden = 'forbidden'
const ErrGeneratePassword = 'failed to password hash'

const DeckCardNumber = 3
const PresentCountPerPage = 100
const SQLDirectory = '../sql/'

const JSTOffset = 9 * 60 * 60 * 1000

const db = mysql.createPool({
  host: process.env['ISUCON_DB_HOST'] ?? '127.0.0.1',
  port: Number(process.env['ISUCON_DB_PORT'] ?? 3306),
  user: process.env['ISUCON_DB_USER'] ?? 'isucon',
  password: process.env['ISUCON_DB_PASSWORD'] ?? 'isucon',
  database: process.env['ISUCON_DB_NAME'] ?? 'isucon',
  timezone: '+09:00',
  connectionLimit: 50,
})

const { generateId } = useGenerateId(db)

const corsOption = {
  origin: '*',
  methods: ['GET', 'POST'],
  allowHeaders: ['Content-Type', 'x-master-version', 'x-session'],
}

const app = express()
app.use(express.json())
app.use(cors(corsOption))
app.use(morgan('combined'))
app.set('etag', false)

// route specific middlewares
app.use('/user', apiMiddleware)
app.use('/login', apiMiddleware)

type CreateUserRequest = {
  viewerId: string
  platformType: number
}

type UpdatedResource = {
  now: number
  user?: User
  userDevice?: UserDevice
  userCards?: UserCard[]
  userDecks?: UserDeck[]
  userItems?: UserItem[]
  userLoginBonuses?: UserLoginBonus[]
  userPresents?: UserPresent[]
}

type ConsumeItem = {
  id: number
  amount: number
}

type GachaData = {
  gacha: GachaMaster
  gachaItemList: GachaItemMaster[]
}

interface TargetUserCardDataRow {
  id: number
  user_id: number
  card_id: number
  amount_per_sec: number
  level: number
  total_exp: number

  // lv1のときの生産性
  base_amount_per_sec: number
  // 最高レベル
  max_level: number
  // lv maxのときの生産性
  max_amount_per_sec: number
  // lv1 -> lv2に上がるときのexp
  base_exp_per_level: number
}

interface ConsumeUserItemDataRow {
  id: number
  user_id: number
  item_id: number
  item_type: number
  amount: number
  created_at: number
  updated_at: number
  gained_exp: number

  consumeAmount: number
}

// リクエスト型定義
type LoginRequest = {
  viewerId: string
  userId: number
}

type RewardRequest = {
  viewerId: string
}

type UpdateDeckRequest = {
  viewerId: string
  cardIds: number[]
}

type AddExpToCardRequest = {
  viewerId: string
  oneTimeToken: string
  items: ConsumeItem[]
}

type ReceivePresentRequest = {
  viewerId: string
  presentIds: number[]
}

type DrawGachaRequest = {
  viewerId: string
  oneTimeToken: string
}

// レスポンス型定義
type InitializeResponse = {
  language: string
}

type CreateUserResponse = {
  userId: number
  viewerId: string
  sessionId: string
  createdAt: number
  updatedResources: UpdatedResource
}

type LoginResponse = {
  viewerId: string
  sessionId: string
  updatedResources: UpdatedResource
}

type HomeResponse = {
  now: number
  user: User
  deck?: UserDeck
  totalAmountPerSec: number
  pastTime: number
}

type RewardResponse = {
  updatedResources: UpdatedResource
}

type UpdateDeckResponse = {
  updatedResources: UpdatedResource
}

type ListItemResponse = {
  oneTimeToken: string
  user: User
  items: UserItem[]
  cards: UserCard[]
}

type AddExpToCardResponse = {
  updatedResources: UpdatedResource
}

type ListPresentResponse = {
  presents: UserPresent[]
  isNext: boolean
}

type ReceivePresentResponse = {
  updatedResources: UpdatedResource
}

type ListGachaResponse = {
  oneTimeToken: string
  gachas: GachaData[]
}

type DrawGachaResponse = {
  presents: UserPresent[]
}

// apiMiddleware
async function apiMiddleware(req: Request, res: Response, next: NextFunction) {
  const isuDate = req.get('x-isu-date')
  if (isuDate) {
    const requestAt = Date.parse(isuDate)
    if (requestAt) {
      res.locals.requestTime = Math.floor(requestAt / 1000)
    }
  }
  if (!res.locals.requestTime) {
    const now = Math.floor(new Date().getTime() / 1000)
    res.locals.requestTime = now
  }

  // マスタ確認
  const query = 'SELECT * FROM version_masters WHERE status=1'
  const [[versionMaster]] = await db.query<(VersionMasterRow & RowDataPacket)[]>(query)
  if (!versionMaster) {
    return errorResponse(res, 404, new Error('active master version is not found'))
  }

  if (versionMaster.master_version != req.get('x-master-version')) {
    return errorResponse(res, 422, new Error(ErrInvalidMasterVersion))
  }

  // check ban
  const userId = getUserId(req)
  if (userId) {
    const isBan = await checkBan(userId)
    if (isBan) {
      return errorResponse(res, 403, new Error(ErrForbidden))
    }
  }

  next()
}

// checkSessionMiddleware
async function checkSessionMiddleware(req: Request, res: Response, next: NextFunction) {
  const sessId = req.get('x-session')
  if (!sessId) {
    return errorResponse(res, 401, new Error(ErrUnauthorized))
  }

  const userId = getUserId(req)
  if (!userId) {
    return errorResponse(res, 400, new Error('invalid userId'))
  }

  const requestAt = getRequestTime(req, res)
  if (!requestAt) {
    return errorResponse(res, 500, new Error(ErrGetRequestTime))
  }

  let query = 'SELECT * FROM user_sessions WHERE session_id=? AND deleted_at IS NULL'
  const [[userSession]] = await db.query<(SessionRow & RowDataPacket)[]>(query, [sessId])
  if (!userSession) {
    return errorResponse(res, 401, new Error(ErrUnauthorized))
  }

  if (userSession.user_id != userId) {
    return errorResponse(res, 403, new Error(ErrForbidden))
  }

  if (userSession.expired_at < requestAt) {
    query = 'UPDATE user_sessions SET deleted_at=? WHERE session_id=?'
    await db.query(query, [requestAt, sessId])
    return errorResponse(res, 401, new Error(ErrExpiredSession))
  }

  next()
}

// checkOneTimeToken
async function checkOneTimeToken(token: string, tokenType: number, requestAt: number): Promise<boolean> {
  let query = 'SELECT * FROM user_one_time_tokens WHERE token=? AND token_type=? AND deleted_at IS NULL'
  const [[tk]] = await db.query<(UserOneTimeTokenRow & RowDataPacket)[]>(query, [token, tokenType])
  if (!tk) {
    return false
  }

  if (tk.expired_at < requestAt) {
    query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?'
    await db.query(query, [requestAt, token])
    return false
  }

  // 使ったトークンを失効する
  query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?'
  await db.query(query, [requestAt, token])

  return true
}

// checkViewerID
async function checkViewerID(userId: number, viewerId: string): Promise<boolean> {
  const query = 'SELECT * FROM user_devices WHERE user_id=? AND platform_id=?'
  const [[device]] = await db.query<(UserDeviceRow & RowDataPacket)[]>(query, [userId, viewerId])
  if (!device) {
    return false
  }
  return true
}

// checkBan
async function checkBan(userId: number): Promise<boolean> {
  const query = 'SELECT * FROM user_bans WHERE user_id=?'
  const [[userBan]] = await db.query<(UserBanRow & RowDataPacket)[]>(query, userId)
  if (!userBan) {
    return false
  }
  return true
}

// loginProcess ログイン処理
async function loginProcess(
  conn: mysql.Connection,
  userId: number,
  requestAt: number
): Promise<{
  user?: User
  loginBonuses?: UserLoginBonus[]
  presents?: UserPresent[]
  error?: Error
}> {
  let query = 'SELECT * FROM users WHERE id=?'
  const [[userRow]] = await conn.query<(UserRow & RowDataPacket)[]>(query, [userId])
  if (!userRow) {
    return {
      error: new Error(ErrUserNotFound),
    }
  }

  // ログインボーナス処理
  const loginBonuses = await obtainLoginBonus(conn, userId, requestAt)

  // 全員プレゼント取得
  const allPresents = await obtainPresent(conn, userId, requestAt)

  const [[result]] = await conn.query<({ isu_coin: number } & RowDataPacket)[]>(
    'SELECT isu_coin FROM users WHERE id=?',
    [userRow.id]
  )
  if (!result) {
    return {
      error: new Error(ErrUserNotFound),
    }
  }

  userRow.isu_coin = result.isu_coin
  userRow.updated_at = requestAt
  userRow.last_activated_at = requestAt

  query = 'UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?'
  await conn.query(query, [requestAt, requestAt, userId])

  return {
    user: toUser(userRow),
    loginBonuses,
    presents: allPresents,
  }
}

// isCompleteTodayLogin ログイン処理が終わっているか
function isCompleteTodayLogin(lastActivatedAt: Date, requestAt: Date): boolean {
  return (
    lastActivatedAt.getFullYear() === requestAt.getFullYear() &&
    lastActivatedAt.getMonth() === requestAt.getMonth() &&
    lastActivatedAt.getDay() === requestAt.getDay()
  )
}

// obtainLoginBonus
async function obtainLoginBonus(conn: mysql.Connection, userId: number, requestAt: number): Promise<UserLoginBonus[]> {
  // login bonus masterから有効なログインボーナスを取得
  let query = 'SELECT * FROM login_bonus_masters WHERE start_at <= ? AND end_at >= ?'
  const [loginBonuses] = await conn.query<(LoginBonusMasterRow & RowDataPacket)[]>(query, [requestAt, requestAt])

  const sendLoginBonuses: UserLoginBonus[] = []

  for (const bonus of loginBonuses) {
    let initBonus = false
    // ボーナスの進捗取得
    query = 'SELECT * FROM user_login_bonuses WHERE user_id=? AND login_bonus_id=?'
    const [[userBonusRow]] = await conn.query<(UserLoginBonusRow & RowDataPacket)[]>(query, [userId, bonus.id])
    let userBonus: UserLoginBonus
    if (!userBonusRow) {
      initBonus = true

      const ubId = await generateId()
      userBonus = {
        // ボーナス初期化
        id: ubId,
        userId: userId,
        loginBonusId: bonus.id,
        lastRewardSequence: 0,
        loopCount: 1,
        createdAt: requestAt,
        updatedAt: requestAt,
      }
    } else {
      userBonus = toUserLoginBonus(userBonusRow)
    }

    // ボーナス進捗更新
    if (userBonus.lastRewardSequence < bonus.column_count) {
      userBonus.lastRewardSequence++
    } else {
      if (bonus.looped) {
        userBonus.loopCount += 1
        userBonus.lastRewardSequence = 1
      } else {
        // 上限まで付与完了
        continue
      }
    }
    userBonus.updatedAt = requestAt

    // 今回付与するリソース取得
    query = 'SELECT * FROM login_bonus_reward_masters WHERE login_bonus_id=? AND reward_sequence=?'
    const [[rewardItem]] = await conn.query<(LoginBonusRewardMasterRow & RowDataPacket)[]>(query, [
      bonus.id,
      userBonus.lastRewardSequence,
    ])
    if (!rewardItem) {
      throw ErrLoginBonusRewardNotFound
    }

    await obtainItem(conn, userId, rewardItem.item_id, rewardItem.item_type, rewardItem.amount, requestAt)

    // 進捗の保存
    if (initBonus) {
      query =
        'INSERT INTO user_login_bonuses(id, user_id, login_bonus_id, last_reward_sequence, loop_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
      await conn.query(query, [
        userBonus.id,
        userBonus.userId,
        userBonus.loginBonusId,
        userBonus.lastRewardSequence,
        userBonus.loopCount,
        userBonus.createdAt,
        userBonus.updatedAt,
      ])
    } else {
      query = 'UPDATE user_login_bonuses SET last_reward_sequence=?, loop_count=?, updated_at=? WHERE id=?'
      await conn.query(query, [userBonus.lastRewardSequence, userBonus.loopCount, userBonus.updatedAt, userBonus.id])
    }

    sendLoginBonuses.push(userBonus)
  }

  return sendLoginBonuses
}

// obtainPresent プレゼント付与処理
async function obtainPresent(conn: mysql.Connection, userId: number, requestAt: number): Promise<UserPresent[]> {
  let query = 'SELECT * FROM present_all_masters WHERE registered_start_at <= ? AND registered_end_at >= ?'
  const [normalPresents] = await conn.query<(PresentAllMasterRow & RowDataPacket)[]>(query, [requestAt, requestAt])

  // 全員プレゼント取得情報更新
  const obtainPresents: UserPresent[] = []
  for (const np of normalPresents) {
    query = 'SELECT * FROM user_present_all_received_history WHERE user_id=? AND present_all_id=?'
    const [[received]] = await conn.query<(UserPresentAllReceivedHistoryRow & RowDataPacket)[]>(query, [userId, np.id])
    if (received) {
      // プレゼント配布済
      continue
    }

    // user present boxに入れる
    const pId = await generateId()
    const up: UserPresent = {
      id: pId,
      userId,
      sentAt: requestAt,
      itemType: np.item_type,
      itemId: np.item_id,
      amount: np.amount,
      presentMessage: np.present_message,
      createdAt: requestAt,
      updatedAt: requestAt,
    }
    query =
      'INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)'
    await conn.query(query, [
      up.id,
      up.userId,
      up.sentAt,
      up.itemType,
      up.itemId,
      up.amount,
      up.presentMessage,
      up.createdAt,
      up.updatedAt,
    ])

    // historyに入れる
    const phId = await generateId()
    const history: UserPresentAllReceivedHistoryRow = {
      id: phId,
      user_id: userId,
      present_all_id: np.id,
      received_at: requestAt,
      created_at: requestAt,
      updated_at: requestAt,
    }
    query =
      'INSERT INTO user_present_all_received_history(id, user_id, present_all_id, received_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)'
    await conn.query(query, [
      history.id,
      history.user_id,
      history.present_all_id,
      history.received_at,
      history.created_at,
      history.updated_at,
    ])

    obtainPresents.push(up)
  }

  return obtainPresents
}

// obtainItem アイテム付与処理
async function obtainItem(
  conn: mysql.Connection,
  userId: number,
  itemId: number,
  itemType: number,
  obtainAmount: number,
  requestAt: number
): Promise<{
  coins: number[]
  cards: UserCard[]
  items: UserItem[]
}> {
  const obtainCoins: number[] = []
  const obtainCards: UserCard[] = []
  const obtainItems: UserItem[] = []

  switch (itemType) {
    case 1: {
      // coin
      let query = 'SELECT * FROM users WHERE id=?'
      const [[user]] = await conn.query<(UserRow & RowDataPacket)[]>(query, [userId])
      if (!user) {
        throw ErrUserNotFound
      }

      query = 'UPDATE users SET isu_coin=? WHERE id=?'
      const totalCoin = user.isu_coin + obtainAmount
      await conn.query(query, [totalCoin, user.id])
      obtainCoins.push(obtainAmount)
      break
    }

    case 2: {
      // card(ハンマー)
      let query = 'SELECT * FROM item_masters WHERE id=? AND item_type=?'
      const [[item]] = await conn.query<(ItemMasterRow & RowDataPacket)[]>(query, [itemId, itemType])
      if (!item) {
        throw ErrItemNotFound
      }

      const cId = await generateId()
      const card: UserCard = {
        id: cId,
        userId,
        cardId: item.id,
        amountPerSec: item.amount_per_sec,
        level: 1,
        totalExp: 0,
        createdAt: requestAt,
        updatedAt: requestAt,
      }

      query =
        'INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)'
      await conn.query(query, [
        card.id,
        card.userId,
        card.cardId,
        card.amountPerSec,
        card.level,
        card.totalExp,
        card.createdAt,
        card.updatedAt,
      ])
      obtainCards.push(card)
      break
    }

    case 3:
    case 4: {
      // 強化素材
      let query = 'SELECT * FROM item_masters WHERE id=? AND item_type=?'
      const [[item]] = await conn.query<(ItemMasterRow & RowDataPacket)[]>(query, [itemId, itemType])
      if (!item) {
        throw ErrItemNotFound
      }

      // 所持数取得
      query = 'SELECT * FROM user_items WHERE user_id=? AND item_id=?'
      const [[uitem]] = await conn.query<(UserItemRow & RowDataPacket)[]>(query, [userId, item.id])
      let userItem: UserItem
      if (!uitem) {
        // 新規作成
        const uitemId = await generateId()
        userItem = {
          id: uitemId,
          userId,
          itemType: item.item_type,
          itemId: item.id,
          amount: obtainAmount,
          createdAt: requestAt,
          updatedAt: requestAt,
        }

        query =
          'INSERT INTO user_items(id, user_id, item_id, item_type, amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
        await conn.query(query, [
          userItem.id,
          userId,
          userItem.itemId,
          userItem.itemType,
          userItem.amount,
          requestAt,
          requestAt,
        ])
      } else {
        // 更新
        userItem = toUserItem(uitem)
        userItem.amount += obtainAmount
        userItem.updatedAt = requestAt
        query = 'UPDATE user_items SET amount=?, updated_at=? WHERE id=?'
        await conn.query(query, [userItem.amount, userItem.updatedAt, userItem.id])
      }

      obtainItems.push(userItem)
      break
    }

    default:
      throw ErrInvalidItemType
  }

  return {
    coins: obtainCoins,
    cards: obtainCards,
    items: obtainItems,
  }
}

// getEnv gets environment variable.
function getEnv(key: string, defaultValue: string): string {
  const val = process.env[key]
  if (val !== undefined) {
    return val
  }

  return defaultValue
}

// see: https://expressjs.com/en/advanced/best-practice-performance.html#handle-exceptions-properly
const wrap =
  (fn: (req: Request, res: Response, next: NextFunction) => Promise<Response | void>): RequestHandler =>
  (req, res, next) =>
    fn(req, res, next).catch(next)

// initialize 初期化処理
// POST /initialize
app.post(
  '/initialize',
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      await exec(['/bin/sh', '-c', SQLDirectory + 'init.sh'].join(' '))

      const data: InitializeResponse = {
        language: 'node',
      }
      return successResponse(res, data)
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// createUser ユーザの作成
// POST /user
app.post(
  '/user',
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const { viewerId, platformType }: CreateUserRequest = req.body

      if (!viewerId || platformType < 1 || platformType > 3) {
        return errorResponse(res, 400, new Error(ErrInvalidRequestBody))
      }

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      const conn = await db.getConnection()
      await conn.beginTransaction()
      try {
        // ユーザ作成
        const uId = await generateId()
        const userRow: UserRow = {
          id: uId,
          isu_coin: 0,
          last_getreward_at: requestAt,
          last_activated_at: requestAt,
          registered_at: requestAt,
          created_at: requestAt,
          updated_at: requestAt,
        }
        let query =
          'INSERT INTO users(id, last_activated_at, registered_at, last_getreward_at, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?)'
        await conn.query(query, [
          userRow.id,
          userRow.last_activated_at,
          userRow.registered_at,
          userRow.last_getreward_at,
          userRow.created_at,
          userRow.updated_at,
        ])

        const udId = await generateId()
        const userDevice: UserDevice = {
          id: udId,
          userId: userRow.id,
          platformId: viewerId,
          platformType: platformType,
          createdAt: requestAt,
          updatedAt: requestAt,
        }
        query =
          'INSERT INTO user_devices(id, user_id, platform_id, platform_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)'
        await conn.query(query, [userDevice.id, userRow.id, viewerId, platformType, requestAt, requestAt])

        // 初期デッキ付与
        query = 'SELECT * FROM item_masters WHERE id=?'
        const [[initCard]] = await conn.query<(ItemMasterRow & RowDataPacket)[]>(query, [2])
        if (!initCard) {
          await conn.rollback()
          return errorResponse(res, 404, new Error(ErrItemNotFound))
        }

        const initCards: UserCard[] = []
        for (const _ of Array(3)) {
          const cId = await generateId()
          const card: UserCard = {
            id: cId,
            userId: userRow.id,
            cardId: initCard.id,
            amountPerSec: initCard.amount_per_sec,
            level: 1,
            totalExp: 0,
            createdAt: requestAt,
            updatedAt: requestAt,
          }
          query =
            'INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)'
          await conn.query(query, [
            card.id,
            card.userId,
            card.cardId,
            card.amountPerSec,
            card.level,
            card.totalExp,
            card.createdAt,
            card.updatedAt,
          ])
          initCards.push(card)
        }

        const deckId = await generateId()
        const initDeck: UserDeck = {
          id: deckId,
          userId: userRow.id,
          cardId1: initCards[0].id,
          cardId2: initCards[1].id,
          cardId3: initCards[2].id,
          createdAt: requestAt,
          updatedAt: requestAt,
        }
        query =
          'INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
        await conn.query(query, [
          initDeck.id,
          initDeck.userId,
          initDeck.cardId1,
          initDeck.cardId2,
          initDeck.cardId3,
          initDeck.createdAt,
          initDeck.updatedAt,
        ])

        // ログイン処理
        const { user, loginBonuses, presents, error } = await loginProcess(conn, userRow.id, requestAt)
        if (error) {
          if (
            error.message === ErrUserNotFound ||
            error.message === ErrItemNotFound ||
            error.message === ErrLoginBonusRewardNotFound
          ) {
            await conn.rollback()
            return errorResponse(res, 404, error)
          }
          if (error.message === ErrInvalidItemType) {
            await conn.rollback()
            return errorResponse(res, 400, error)
          }
          await conn.rollback()
          return errorResponse(res, 500, error)
        }

        // generate session
        const sId = await generateId()
        const sessId = generateUUID()

        const sess: Session = {
          id: sId,
          userId: userRow.id,
          sessionId: sessId,
          createdAt: requestAt,
          updatedAt: requestAt,
          expiredAt: requestAt + 86400,
        }
        query =
          'INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)'
        await conn.query(query, [sess.id, sess.userId, sess.sessionId, sess.createdAt, sess.updatedAt, sess.expiredAt])

        await conn.commit()

        const payload: CreateUserResponse = {
          userId: userRow.id,
          viewerId: viewerId,
          sessionId: sess.sessionId,
          createdAt: requestAt,
          updatedResources: {
            now: requestAt,
            user,
            userDevice,
            userCards: initCards,
            userDecks: [initDeck],
            userLoginBonuses: loginBonuses,
            userPresents: presents,
          },
        }
        return successResponse(res, payload)
      } catch (error) {
        await conn.rollback()
        return errorResponse(res, 500, error as Error)
      } finally {
        conn.release()
      }
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// login ログイン
// POST /login
app.post(
  '/login',
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const { viewerId, userId }: LoginRequest = req.body

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      const query = 'SELECT * FROM users WHERE id=?'
      const [[userRow]] = await db.query<(UserRow & RowDataPacket)[]>(query, [userId])
      if (!userRow) {
        return errorResponse(res, 404, new Error(ErrUserNotFound))
      }

      // check ban
      const isBan = await checkBan(userRow.id)
      if (isBan) {
        return errorResponse(res, 403, new Error(ErrForbidden))
      }

      // viewer id check
      if (!(await checkViewerID(userRow.id, viewerId))) {
        return errorResponse(res, 404, new Error(ErrUserDeviceNotFound))
      }

      const conn = await db.getConnection()
      await conn.beginTransaction()
      try {
        // sessionを更新
        let query = 'UPDATE user_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
        await conn.query(query, [requestAt, userId])

        const sId = await generateId()
        const sessId = generateUUID()

        const sess: Session = {
          id: sId,
          userId,
          sessionId: sessId,
          createdAt: requestAt,
          updatedAt: requestAt,
          expiredAt: requestAt + 86400,
        }
        query =
          'INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)'
        await conn.query(query, [sess.id, sess.userId, sess.sessionId, sess.createdAt, sess.updatedAt, sess.expiredAt])

        // すでにログインしているユーザはログイン処理をしない
        const lastActivatedAtDate = new Date(userRow.last_activated_at * 1000 + JSTOffset)
        const requestAtDate = new Date(requestAt * 1000 + JSTOffset)
        if (isCompleteTodayLogin(lastActivatedAtDate, requestAtDate)) {
          userRow.updated_at = requestAt
          userRow.last_activated_at = requestAt

          query = 'UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?'
          await conn.query(query, [requestAt, requestAt, userId])

          await conn.commit()

          const payload: LoginResponse = {
            viewerId,
            sessionId: sess.sessionId,
            updatedResources: {
              now: requestAt,
              user: toUser(userRow),
            },
          }
          return successResponse(res, payload)
        }

        // login process
        const { user, loginBonuses, presents, error } = await loginProcess(conn, userId, requestAt)
        if (error) {
          await conn.rollback()
          return errorResponse(res, 500, error)
        }

        await conn.commit()

        const payload: LoginResponse = {
          viewerId,
          sessionId: sess.sessionId,
          updatedResources: {
            now: requestAt,
            user,
            userLoginBonuses: loginBonuses,
            userPresents: presents,
          },
        }
        return successResponse(res, payload)
      } catch (error) {
        await conn.rollback()
        return errorResponse(res, 500, error as Error)
      } finally {
        conn.release()
      }
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// listGacha ガチャ一覧
// GET /user/:userId/gacha/index
app.get(
  '/user/:userId/gacha/index',
  checkSessionMiddleware,
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const userId = getUserId(req)

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      let query = 'SELECT * FROM gacha_masters WHERE start_at <= ? AND end_at >= ? ORDER BY display_order ASC'
      const [gachaMasterList] = await db.query<(GachaMasterRow & RowDataPacket)[]>(query, [requestAt, requestAt])
      if (!gachaMasterList.length) {
        const payload: ListGachaResponse = {
          oneTimeToken: '',
          gachas: [],
        }
        return successResponse(res, payload)
      }

      // ガチャ排出アイテム取得
      const gachaDataList: GachaData[] = []
      query = 'SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC'
      for (const gacha of gachaMasterList) {
        const [gachaItem] = await db.query<(GachaItemMasterRow & RowDataPacket)[]>(query, [gacha.id])
        if (!gachaItem.length) {
          return errorResponse(res, 404, new Error('not found gacha item'))
        }

        gachaDataList.push({
          gacha: toGachaMaster(gacha),
          gachaItemList: gachaItem.map(toGachaItemMaster),
        })
      }

      // generate one time token
      query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
      await db.query(query, [requestAt, userId])

      const tId = await generateId()
      const tk = generateUUID()
      const token: UserOneTimeTokenRow = {
        id: tId,
        user_id: userId,
        token: tk,
        token_type: 1,
        created_at: requestAt,
        updated_at: requestAt,
        expired_at: requestAt + 600,
      }
      query =
        'INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
      await db.query(query, [
        token.id,
        token.user_id,
        token.token,
        token.token_type,
        token.created_at,
        token.updated_at,
        token.expired_at,
      ])

      const payload: ListGachaResponse = {
        oneTimeToken: token.token,
        gachas: gachaDataList,
      }
      return successResponse(res, payload)
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// drawGacha ガチャを引く
// POST /user/:userId/gacha/draw/:gachaId/:n
app.post(
  '/user/:userId/gacha/draw/:gachaId/:n',
  checkSessionMiddleware,
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const userId = getUserId(req)

      const gachaId = req.params.gachaId
      if (!gachaId) {
        return errorResponse(res, 400, new Error('invalid gachaId'))
      }

      const gachaCount = parseInt(req.params.n, 10)
      if (gachaCount !== 1 && gachaCount !== 10) {
        return errorResponse(res, 400, new Error('invalid draw gacha times'))
      }

      const { viewerId, oneTimeToken }: DrawGachaRequest = req.body

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      if (!(await checkOneTimeToken(oneTimeToken, 1, requestAt))) {
        return errorResponse(res, 400, new Error(ErrInvalidToken))
      }

      if (!(await checkViewerID(userId, viewerId))) {
        return errorResponse(res, 404, new Error(ErrUserDeviceNotFound))
      }

      const consumedCoin = gachaCount * 1000

      // userのisucoinが足りるか
      let query = 'SELECT * FROM users WHERE id=?'
      const [[user]] = await db.query<(UserRow & RowDataPacket)[]>(query, [userId])
      if (!user) {
        return errorResponse(res, 404, new Error(ErrUserNotFound))
      }
      if (user.isu_coin < consumedCoin) {
        return errorResponse(res, 409, new Error('not enough isucon'))
      }

      // gachaIdからガチャマスタの取得
      query = 'SELECT * FROM gacha_masters WHERE id=? AND start_at <= ? AND end_at >= ?'
      const [[gachaInfo]] = await db.query<(GachaMasterRow & RowDataPacket)[]>(query, [gachaId, requestAt, requestAt])
      if (!gachaInfo) {
        return errorResponse(res, 404, new Error('not found gacha'))
      }

      // gachaItemMasterからアイテムリスト取得
      const [gachaItemList] = await db.query<(GachaItemMasterRow & RowDataPacket)[]>(
        'SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC',
        [gachaId]
      )
      if (!gachaItemList.length) {
        return errorResponse(res, 404, new Error('not found gacha item'))
      }

      // weightの合計値を算出
      const [[weight]] = await db.query<({ 'SUM(weight)': number } & RowDataPacket)[]>(
        'SELECT SUM(weight) FROM gacha_item_masters WHERE gacha_id=?',
        [gachaId]
      )
      if (!res) {
        return errorResponse(res, 404, new Error('sql: no rows in result set'))
      }
      const sum = weight['SUM(weight)']

      // random値の導出 & 抽選
      const result: GachaItemMaster[] = []
      for (const _ of Array(gachaCount)) {
        const random = Math.floor(Math.random() * sum)
        let boundary = 0
        for (const item of gachaItemList) {
          boundary += item.weight
          if (random < boundary) {
            result.push(toGachaItemMaster(item))
            break
          }
        }
      }

      const conn = await db.getConnection()
      await conn.beginTransaction()
      try {
        // 直付与 => プレゼントに入れる
        const presents: UserPresent[] = []
        for (const item of result) {
          const pId = await generateId()
          const present: UserPresent = {
            id: pId,
            userId,
            sentAt: requestAt,
            itemType: item.itemType,
            itemId: item.itemId,
            amount: item.amount,
            presentMessage: `${gachaInfo.name}の付与アイテムです`,
            createdAt: requestAt,
            updatedAt: requestAt,
          }

          query =
            'INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)'
          await conn.query(query, [
            present.id,
            present.userId,
            present.sentAt,
            present.itemType,
            present.itemId,
            present.amount,
            present.presentMessage,
            present.createdAt,
            present.updatedAt,
          ])

          presents.push(present)
        }

        // isucoinをへらす
        query = 'UPDATE users SET isu_coin=? WHERE id=?'
        const totalCoin = user.isu_coin - consumedCoin
        await conn.query(query, [totalCoin, user.id])

        await conn.commit()

        const payload: DrawGachaResponse = {
          presents,
        }
        return successResponse(res, payload)
      } catch (error) {
        await conn.rollback()
        return errorResponse(res, 500, error as Error)
      } finally {
        conn.release()
      }
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// listPresent プレゼント一覧
// GET /user/:userId/present/index/:n
app.get(
  '/user/:userId/present/index/:n',
  checkSessionMiddleware,
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const n = parseInt(req.params.n, 10)
      if (n === 0) {
        return errorResponse(res, 400, new Error('index number is more than 1'))
      }

      const userId = getUserId(req)

      const offset = PresentCountPerPage * (n - 1)
      const query = `
      SELECT * FROM user_presents 
      WHERE user_id = ? AND deleted_at IS NULL
      ORDER BY created_at DESC, id
      LIMIT ? OFFSET ?`
      const [presentList] = await db.query<(UserPresentRow & RowDataPacket)[]>(query, [
        userId,
        PresentCountPerPage,
        offset,
      ])

      const [[result]] = await db.query<({ 'COUNT(*)': number } & RowDataPacket)[]>(
        'SELECT COUNT(*) FROM user_presents WHERE user_id = ? AND deleted_at IS NULL',
        [userId]
      )
      const presentCount = result['COUNT(*)'] as number

      let isNext = false
      if (presentCount > offset + PresentCountPerPage) {
        isNext = true
      }

      const payload: ListPresentResponse = {
        presents: presentList.map(toUserPresent),
        isNext,
      }
      return successResponse(res, payload)
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// receivePresent プレゼント受け取り
// POST /user/:userId/present/receive
app.post(
  '/user/:userId/present/receive',
  checkSessionMiddleware,
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const { viewerId, presentIds }: ReceivePresentRequest = req.body

      const userId = getUserId(req)

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      if (!presentIds.length) {
        return errorResponse(res, 422, new Error('presentIds is empty'))
      }

      if (!(await checkViewerID(userId, viewerId))) {
        return errorResponse(res, 404, new Error(ErrUserDeviceNotFound))
      }

      // user_presentsに入っているが未取得のプレゼント取得
      const questions = Array(presentIds.length).fill('?')
      let query = 'SELECT * FROM user_presents WHERE id IN (' + questions.join(',') + ') AND deleted_at IS NULL'
      const [obtainPresent] = await db.query<(UserPresentRow & RowDataPacket)[]>(query, presentIds)

      if (!obtainPresent.length) {
        const payload: ReceivePresentResponse = {
          updatedResources: {
            now: requestAt,
            userPresents: [],
          },
        }
        return successResponse(res, payload)
      }

      const conn = await db.getConnection()
      await conn.beginTransaction()
      try {
        // 配布処理
        for (const present of obtainPresent) {
          if (present.deleted_at) {
            await conn.rollback()
            return errorResponse(res, 500, new Error('received present'))
          }

          present.updated_at = requestAt
          present.deleted_at = requestAt

          query = 'UPDATE user_presents SET deleted_at=?, updated_at=? WHERE id=?'
          await conn.query(query, [requestAt, requestAt, present.id])

          try {
            await obtainItem(conn, present.user_id, present.item_id, present.item_type, present.amount, requestAt)
          } catch (error: any) {
            if (error.message === ErrUserNotFound || error.message === ErrItemNotFound) {
              await conn.rollback()
              return errorResponse(res, 404, error)
            }
            if (error.message === ErrInvalidItemType) {
              await conn.rollback()
              return errorResponse(res, 400, error)
            }
            await conn.rollback()
            return errorResponse(res, 500, error)
          }
        }

        await conn.commit()

        const payload: ReceivePresentResponse = {
          updatedResources: {
            now: requestAt,
            userPresents: obtainPresent.map(toUserPresent),
          },
        }
        return successResponse(res, payload)
      } catch (error) {
        await conn.rollback()
        return errorResponse(res, 500, error as Error)
      } finally {
        conn.release()
      }
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// listItem アイテムリスト
// GET /user/:userId/item
app.get(
  '/user/:userId/item',
  checkSessionMiddleware,
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const userId = getUserId(req)

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      let query = 'SELECT * FROM users WHERE id=?'
      const [[user]] = await db.query<(UserRow & RowDataPacket)[]>(query, [userId])
      if (!user) {
        throw ErrUserNotFound
      }

      query = 'SELECT * FROM user_items WHERE user_id = ?'
      const [itemList] = await db.query<(UserItemRow & RowDataPacket)[]>(query, [userId])

      query = 'SELECT * FROM user_cards WHERE user_id=?'
      const [cardList] = await db.query<(UserCardRow & RowDataPacket)[]>(query, [userId])

      // generate one time token
      query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
      await db.query(query, [requestAt, userId])

      const tId = await generateId()
      const tk = generateUUID()
      const token: UserOneTimeTokenRow = {
        id: tId,
        user_id: userId,
        token: tk,
        token_type: 2,
        created_at: requestAt,
        updated_at: requestAt,
        expired_at: requestAt + 600,
      }

      query =
        'INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
      await db.query(query, [
        token.id,
        token.user_id,
        token.token,
        token.token_type,
        token.created_at,
        token.updated_at,
        token.expired_at,
      ])

      const payload: ListItemResponse = {
        oneTimeToken: token.token,
        items: itemList.map(toUserItem),
        user: toUser(user),
        cards: cardList.map(toUserCard),
      }
      return successResponse(res, payload)
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// addExpToCard 装備強化
// POST /user/:userId/card/addexp/:cardId
app.post(
  '/user/:userId/card/addexp/:cardId',
  checkSessionMiddleware,
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const cardId = parseInt(req.params.cardId, 10)
      const userId = getUserId(req)

      const { viewerId, oneTimeToken, items }: AddExpToCardRequest = req.body

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      if (!(await checkOneTimeToken(oneTimeToken, 2, requestAt))) {
        return errorResponse(res, 400, new Error(ErrInvalidToken))
      }

      if (!(await checkViewerID(userId, viewerId))) {
        return errorResponse(res, 404, new Error(ErrUserDeviceNotFound))
      }

      // get target card
      let query = `
      SELECT uc.id , uc.user_id , uc.card_id , uc.amount_per_sec , uc.level, uc.total_exp, im.amount_per_sec as 'base_amount_per_sec', im.max_level , im.max_amount_per_sec , im.base_exp_per_level
      FROM user_cards as uc
      INNER JOIN item_masters as im ON uc.card_id = im.id
      WHERE uc.id = ? AND uc.user_id=?
      `
      const [[card]] = await db.query<(TargetUserCardDataRow & RowDataPacket)[]>(query, [cardId, userId])
      if (!card) {
        return errorResponse(res, 404, new Error('sql: no rows in result set'))
      }

      if (card.level === card.max_level) {
        return errorResponse(res, 400, new Error('target card is max level'))
      }

      // 消費アイテムの所持チェック
      const consumeItems: ConsumeUserItemDataRow[] = []
      query = `
      SELECT ui.id, ui.user_id, ui.item_id, ui.item_type, ui.amount, ui.created_at, ui.updated_at, im.gained_exp
      FROM user_items as ui
      INNER JOIN item_masters as im ON ui.item_id = im.id
      WHERE ui.item_type = 3 AND ui.id=? AND ui.user_id=?
      `
      for (const item of items) {
        const [[consumeItem]] = await db.query<(ConsumeUserItemDataRow & RowDataPacket)[]>(query, [item.id, userId])
        if (!consumeItem) {
          return errorResponse(res, 404, new Error('sql: no rows in result set'))
        }

        if (item.amount > consumeItem.amount) {
          return errorResponse(res, 400, new Error('item not enough'))
        }
        consumeItem.consumeAmount = item.amount
        consumeItems.push(consumeItem)
      }

      // 経験値付与
      // 経験値をカード付与
      for (const consumeItem of consumeItems) {
        card.total_exp += consumeItem.gained_exp * consumeItem.consumeAmount
      }

      // lvup判定(lv upしたら生産性を加算)
      for (;;) {
        const nextLvThreadhold = Math.floor(card.base_exp_per_level * Math.pow(1.2, card.level - 1))
        if (nextLvThreadhold > card.total_exp) {
          break
        }

        // lv up処理
        card.level += 1
        card.amount_per_sec += (card.max_amount_per_sec - card.base_amount_per_sec) / (card.max_level - 1)
      }

      const conn = await db.getConnection()
      await conn.beginTransaction()
      try {
        // cardのlvと経験値の更新、itemの消費
        query = 'UPDATE user_cards SET amount_per_sec=?, level=?, total_exp=?, updated_at=? WHERE id=?'
        await conn.query(query, [card.amount_per_sec, card.level, card.total_exp, requestAt, card.id])

        query = 'UPDATE user_items SET amount=?, updated_at=? WHERE id=?'
        for (const consumeItem of consumeItems) {
          await conn.query(query, [consumeItem.amount - consumeItem.consumeAmount, requestAt, consumeItem.id])
        }

        // get response data
        query = 'SELECT * FROM user_cards WHERE id=?'
        const [[resultCard]] = await conn.query<(UserCardRow & RowDataPacket)[]>(query, [card.id])
        if (!resultCard) {
          await conn.rollback()
          return errorResponse(res, 404, new Error('not found card'))
        }
        const resultItems: UserItem[] = []
        for (const consumeItem of consumeItems) {
          resultItems.push({
            id: consumeItem.id,
            userId: consumeItem.user_id,
            itemId: consumeItem.item_id,
            itemType: consumeItem.item_type,
            amount: consumeItem.amount - consumeItem.consumeAmount,
            createdAt: consumeItem.created_at,
            updatedAt: requestAt,
          })
        }
        await conn.commit()

        const payload: AddExpToCardResponse = {
          updatedResources: {
            now: requestAt,
            userCards: [toUserCard(resultCard)],
            userItems: resultItems,
          },
        }
        return successResponse(res, payload)
      } catch (error) {
        await conn.rollback()
        return errorResponse(res, 500, error as Error)
      } finally {
        conn.release()
      }
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// updateDeck 装備変更
// POST /user/:userId/card
app.post(
  '/user/:userId/card',
  checkSessionMiddleware,
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const userId = getUserId(req)

      const { viewerId, cardIds }: UpdateDeckRequest = req.body

      if (cardIds.length !== DeckCardNumber) {
        return errorResponse(res, 400, new Error('invalid number of cards'))
      }

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      if (!(await checkViewerID(userId, viewerId))) {
        return errorResponse(res, 404, new Error(ErrUserDeviceNotFound))
      }

      // カード所持情報のバリデーション
      let query = 'SELECT * FROM user_cards WHERE id IN (?,?,?)'
      const [cards] = await db.query<(UserCardRow & RowDataPacket)[]>(query, cardIds)
      if (cards.length !== DeckCardNumber) {
        return errorResponse(res, 400, new Error('invalid card ids'))
      }

      const conn = await db.getConnection()
      await conn.beginTransaction()
      try {
        // update data
        query = 'UPDATE user_decks SET updated_at=?, deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
        await conn.query(query, [requestAt, requestAt, userId])

        const udId = await generateId()
        const newDeck: UserDeck = {
          id: udId,
          userId,
          cardId1: cardIds[0],
          cardId2: cardIds[1],
          cardId3: cardIds[2],
          createdAt: requestAt,
          updatedAt: requestAt,
        }
        query =
          'INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)'
        await conn.query(query, [
          newDeck.id,
          newDeck.userId,
          newDeck.cardId1,
          newDeck.cardId2,
          newDeck.cardId3,
          newDeck.createdAt,
          newDeck.updatedAt,
        ])

        await conn.commit()

        const payload: UpdateDeckResponse = {
          updatedResources: {
            now: requestAt,
            userDecks: [newDeck],
          },
        }
        return successResponse(res, payload)
      } catch (error) {
        await conn.rollback()
        return errorResponse(res, 500, error as Error)
      } finally {
        conn.release()
      }
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// reward ゲーム報酬受取
// POST /user/:userId/reward
app.post(
  '/user/:userId/reward',
  checkSessionMiddleware,
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const userId = getUserId(req)

      const { viewerId }: RewardRequest = req.body

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      if (!(await checkViewerID(userId, viewerId))) {
        return errorResponse(res, 404, new Error(ErrUserDeviceNotFound))
      }

      // 最後に取得した報酬時刻取得
      let query = 'SELECT * FROM users WHERE id=?'
      const [[user]] = await db.query<(UserRow & RowDataPacket)[]>(query, [userId])
      if (!user) {
        return errorResponse(res, 404, new Error(ErrUserNotFound))
      }

      // 使っているデッキの取得
      query = 'SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL'
      const [[deck]] = await db.query<(UserDeckRow & RowDataPacket)[]>(query, [userId])
      if (!deck) {
        return errorResponse(res, 404, new Error('sql: no rows in result set'))
      }

      query = 'SELECT * FROM user_cards WHERE id IN (?, ?, ?)'
      const [cards] = await db.query<(UserCardRow & RowDataPacket)[]>(query, [
        deck.user_card_id_1,
        deck.user_card_id_2,
        deck.user_card_id_3,
      ])
      if (cards.length !== DeckCardNumber) {
        return errorResponse(res, 400, new Error('invalid cards length'))
      }

      // 経過時間*生産性のcoin (1椅子 = 1coin)
      const pastTime = requestAt - user.last_getreward_at
      const getCoin = pastTime * (cards[0].amount_per_sec + cards[1].amount_per_sec + cards[2].amount_per_sec)

      // 報酬の保存(ゲーム内通貨の保存)(users)
      user.isu_coin += getCoin
      user.last_getreward_at = requestAt

      query = 'UPDATE users SET isu_coin=?, last_getreward_at=? WHERE id=?'
      await db.query(query, [user.isu_coin, user.last_getreward_at, user.id])

      const payload: RewardResponse = {
        updatedResources: {
          now: requestAt,
          user: toUser(user),
        },
      }
      return successResponse(res, payload)
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// home ホーム取得
// GET /user/:userId/home
app.get(
  '/user/:userId/home',
  checkSessionMiddleware,
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    try {
      const userId = getUserId(req)

      const requestAt = getRequestTime(req, res)
      if (!requestAt) {
        return errorResponse(res, 500, new Error(ErrGetRequestTime))
      }

      // 装備情報
      let query = 'SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL'
      const [[deck]] = await db.query<(UserDeckRow & RowDataPacket)[]>(query, [userId])

      // 生産性
      const cards: UserCard[] = []
      if (deck) {
        const cardIds = [deck.user_card_id_1, deck.user_card_id_2, deck.user_card_id_3]
        query = 'SELECT * FROM user_cards WHERE id IN (?,?,?)'
        const [cardRows] = await db.query<(UserCardRow & RowDataPacket)[]>(query, cardIds)
        cards.push(...cardRows.map(toUserCard))
      }
      let totalAmountPerSec = 0
      for (const card of cards) {
        totalAmountPerSec += card.amountPerSec
      }

      // 経過時間
      query = 'SELECT * FROM users WHERE id=?'
      const [[user]] = await db.query<(UserRow & RowDataPacket)[]>(query, [userId])
      if (!user) {
        return errorResponse(res, 404, new Error(ErrUserNotFound))
      }
      const pastTime = requestAt - user.last_getreward_at

      const payload: HomeResponse = {
        now: requestAt,
        user: toUser(user),
        deck: deck ? toUserDeck(deck) : undefined,
        totalAmountPerSec,
        pastTime,
      }
      return successResponse(res, payload)
    } catch (error) {
      return errorResponse(res, 500, error as Error)
    }
  })
)

// health ヘルスチェック
app.get(
  '/health',
  wrap(async (req: Request, res: Response, _next: NextFunction) => {
    res.status(200).send('OK')
  })
)

const { adminRouter } = useAdmin(db)

app.use('/', adminRouter)

const port = getEnv('SERVER_APP_PORT', '8080')
console.log('Start server: address=:' + port + ' ...')
app.listen(port)
