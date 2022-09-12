import { Request, Response, NextFunction, RequestHandler, Router } from 'express'
import mysql, { RowDataPacket } from 'mysql2/promise'
import bcrypt from 'bcrypt'
import multer from 'multer'
import { parse } from 'csv-parse/sync'

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
  UserPresentAllReceivedHistory,
  ItemMaster,
  GachaMaster,
  GachaItemMaster,
  PresentAllMaster,
  LoginBonusMaster,
  LoginBonusRewardMaster,
  VersionMaster,
  AdminUserRow,
  SessionRow,
  UserRow,
  UserDeviceRow,
  UserCardRow,
  UserDeckRow,
  UserItemRow,
  UserLoginBonusRow,
  UserPresentRow,
  UserPresentAllReceivedHistoryRow,
  ItemMasterRow,
  GachaMasterRow,
  GachaItemMasterRow,
  PresentAllMasterRow,
  LoginBonusMasterRow,
  LoginBonusRewardMasterRow,
  VersionMasterRow,
  toUser,
  toUserDevice,
  toUserCard,
  toUserDeck,
  toUserItem,
  toUserLoginBonus,
  toUserPresent,
  toUserPresentAllReceivedHistory,
  toItemMaster,
  toGachaMaster,
  toGachaItemMaster,
  toPresentAllMaster,
  toLoginBonusMaster,
  toLoginBonusRewardMaster,
  toVersionMaster,
} from './types'

const ErrGetRequestTime = 'failed to get request time'
const ErrUserNotFound = 'not found user'
const ErrUnauthorized = 'unauthorized user'
const ErrExpiredSession = 'session expired'
const ErrNoFormFile = 'no such file'

const upload = multer()

// see: https://expressjs.com/en/advanced/best-practice-performance.html#handle-exceptions-properly
const wrap =
  (fn: (req: Request, res: Response, next: NextFunction) => Promise<Response | void>): RequestHandler =>
  (req, res, next) =>
    fn(req, res, next).catch(next)

type AdminLoginRequest = {
  userId: number
  password: string
}

type AdminLoginResponse = {
  session: Session
}

type AdminListMasterResponse = {
  versionMaster: VersionMaster[]
  items: ItemMaster[]
  gachas: GachaMaster[]
  gachaItems: GachaItemMaster[]
  presentAlls: PresentAllMaster[]
  loginBonusRewards: LoginBonusRewardMaster[]
  loginBonuses: LoginBonusMaster[]
}

type AdminUserResponse = {
  user: User

  userDevices: UserDevice[]
  userCards: UserCard[]
  userDecks: UserDeck[]
  userItems: UserItem[]
  userLoginBonuses: UserLoginBonus[]
  userPresents: UserPresent[]
  userPresentAllReceivedHistory: UserPresentAllReceivedHistory[]
}

type AdminBanUserResponse = {
  user: User
}

type AdminUpdateMasterResponse = {
  versionMaster: VersionMaster
}

function verifyPassword(hash: string, pw: string): boolean {
  return bcrypt.compareSync(pw, hash)
}

// noContentResponse
function noContentResponse(res: Response, statusCode: number) {
  res.status(statusCode).send()
}

// readFromFileToCSV ファイルからcsvレコードを取得する
function readFormFileToCSV(req: Request, name: string): { records?: string[][]; error?: Error } {
  const files = req.files as { [fieldname: string]: Express.Multer.File[] }
  if (!files || !files[name] || !files[name].length) {
    return {
      error: new Error(ErrNoFormFile),
    }
  }

  const buf = files[name][0].buffer

  const records: any[] = parse(buf.toString(), {
    skip_empty_lines: true,
  })

  return {
    records,
  }
}

export const useAdmin = (db: mysql.Pool) => {
  const adminRouter = Router()
  const { generateId } = useGenerateId(db)

  // adminMiddleware
  const adminMiddleware = async (req: Request, res: Response, next: NextFunction) => {
    const now = Math.floor(new Date().getTime() / 1000)
    res.locals.requestTime = now

    next()
  }

  adminRouter.use(adminMiddleware)

  // adminSessionCheckMiddleware
  const adminSessionCheckMiddleware = async (req: Request, res: Response, next: NextFunction) => {
    const sessId = req.get('x-session')

    let query = 'SELECT * FROM admin_sessions WHERE session_id=? AND deleted_at IS NULL'
    const [[adminSession]] = await db.query<(SessionRow & RowDataPacket)[]>(query, [sessId])

    if (!adminSession) {
      return errorResponse(res, 401, new Error(ErrUnauthorized))
    }

    const requestAt = getRequestTime(req, res)
    if (!requestAt) {
      return errorResponse(res, 500, new Error(ErrGetRequestTime))
    }

    if (adminSession.expired_at < requestAt) {
      query = 'UPDATE admin_sessions SET deleted_at=? WHERE session_id=?'
      await db.query(query, [requestAt, sessId])
      return errorResponse(res, 401, new Error(ErrExpiredSession))
    }

    next()
  }

  // adminLogin 管理者権限ログイン
  // POST /admin/login
  adminRouter.post(
    '/admin/login',
    wrap(async (req: Request, res: Response, _: NextFunction) => {
      try {
        const { userId, password }: AdminLoginRequest = req.body

        const requestAt = getRequestTime(req, res)
        if (!requestAt) {
          return errorResponse(res, 500, new Error(ErrGetRequestTime))
        }

        const conn = await db.getConnection()
        await conn.beginTransaction()
        try {
          // userの存在確認
          let query = 'SELECT * FROM admin_users WHERE id=?'
          const [[user]] = await conn.query<(AdminUserRow & RowDataPacket)[]>(query, [userId])
          if (!user) {
            await conn.rollback()
            return errorResponse(res, 404, new Error(ErrUserNotFound))
          }

          // verify password
          if (!verifyPassword(user.password, password)) {
            await conn.rollback()
            return errorResponse(res, 401, new Error(ErrUnauthorized))
          }

          query = 'UPDATE admin_users SET last_activated_at=?, updated_at=? WHERE id=?'
          await conn.query(query, [requestAt, requestAt, userId])

          // すでにあるsessionをdeleteにする
          query = 'UPDATE admin_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL'
          await conn.query(query, [requestAt, userId])

          // create session
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
            'INSERT INTO admin_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)'
          await conn.query(query, [
            sess.id,
            sess.userId,
            sess.sessionId,
            sess.createdAt,
            sess.updatedAt,
            sess.expiredAt,
          ])

          await conn.commit()

          const payload: AdminLoginResponse = {
            session: sess,
          }
          return successResponse(res, payload)
        } catch (error) {
          await conn.rollback()
          return errorResponse(res, 500, error as Error)
        } finally {
          await conn.rollback()
          conn.release()
        }
      } catch (error) {
        return errorResponse(res, 500, error as Error)
      }
    })
  )

  // adminLogout 管理者権限ログアウト
  // DELETE /admin/logout
  adminRouter.delete(
    '/admin/logout',
    adminSessionCheckMiddleware,
    wrap(async (req: Request, res: Response, _: NextFunction) => {
      try {
        const sessId = req.get('x-session')

        const requestAt = getRequestTime(req, res)
        if (!requestAt) {
          return errorResponse(res, 500, new Error(ErrGetRequestTime))
        }

        // すでにあるsessionをdeleteにする
        const query = 'UPDATE admin_sessions SET deleted_at=? WHERE session_id=? AND deleted_at IS NULL'
        await db.query(query, [requestAt, sessId])

        return noContentResponse(res, 204)
      } catch (error) {
        return errorResponse(res, 500, error as Error)
      }
    })
  )

  // adminListMaster マスタデータ閲覧
  // GET /admin/master
  adminRouter.get(
    '/admin/master',
    adminSessionCheckMiddleware,
    wrap(async (req: Request, res: Response, _: NextFunction) => {
      try {
        const [masterVersions] = await db.query<(VersionMasterRow & RowDataPacket)[]>('SELECT * FROM version_masters')

        const [items] = await db.query<(ItemMasterRow & RowDataPacket)[]>('SELECT * FROM item_masters')

        const [gachas] = await db.query<(GachaMasterRow & RowDataPacket)[]>('SELECT * FROM gacha_masters')

        const [gachaItems] = await db.query<(GachaItemMasterRow & RowDataPacket)[]>('SELECT * FROM gacha_item_masters')

        const [presentAlls] = await db.query<(PresentAllMasterRow & RowDataPacket)[]>(
          'SELECT * FROM present_all_masters'
        )

        const [loginBonuses] = await db.query<(LoginBonusMasterRow & RowDataPacket)[]>(
          'SELECT * FROM login_bonus_masters'
        )

        const [loginBonusRewards] = await db.query<(LoginBonusRewardMasterRow & RowDataPacket)[]>(
          'SELECT * FROM login_bonus_reward_masters'
        )

        const payload: AdminListMasterResponse = {
          versionMaster: masterVersions.map(toVersionMaster),
          items: items.map(toItemMaster),
          gachas: gachas.map(toGachaMaster),
          gachaItems: gachaItems.map(toGachaItemMaster),
          presentAlls: presentAlls.map(toPresentAllMaster),
          loginBonuses: loginBonuses.map(toLoginBonusMaster),
          loginBonusRewards: loginBonusRewards.map(toLoginBonusRewardMaster),
        }
        return successResponse(res, payload)
      } catch (error) {
        return errorResponse(res, 500, error as Error)
      }
    })
  )

  // adminUpdateMaster マスタデータ更新
  // PUT /admin/master
  adminRouter.put(
    '/admin/master',
    upload.fields([
      { name: 'versionMaster' },
      { name: 'itemMaster' },
      { name: 'gachaMaster' },
      { name: 'gachaItemMaster' },
      { name: 'presentAllMaster' },
      { name: 'loginBonusMaster' },
      { name: 'loginBonusRewardMaster' },
    ]),
    adminSessionCheckMiddleware,
    wrap(async (req: Request, res: Response, _: NextFunction) => {
      try {
        const conn = await db.getConnection()
        await conn.beginTransaction()
        try {
          // version master
          {
            const { records: versionMasterRecs, error } = readFormFileToCSV(req, 'versionMaster')
            if (error && error.message != ErrNoFormFile) {
              await conn.rollback()
              return errorResponse(res, 400, new Error('invalid versionMaster'))
            }
            if (versionMasterRecs) {
              const data: any[] = []
              versionMasterRecs.forEach((val, index) => {
                if (index === 0) return
                data.push({
                  id: val[0],
                  status: val[1],
                  master_version: val[2],
                })
              })

              const questions = Array(data.length).fill('(?,?,?)')
              const query =
                'INSERT INTO version_masters(id, status, master_version) VALUES ' +
                questions.join(',') +
                ' ON DUPLICATE KEY UPDATE status=VALUES(status), master_version=VALUES(master_version)'
              await conn.query(query, data.map((x) => Object.values(x)).flat())
            } else {
              console.log('Skip Update Master: versionMaster')
            }
          }

          // item
          {
            const { records: itemMasterRecs, error } = readFormFileToCSV(req, 'itemMaster')
            if (error && error.message != ErrNoFormFile) {
              await conn.rollback()
              return errorResponse(res, 400, new Error('invalid itemMaster'))
            }
            if (itemMasterRecs) {
              const data: any[] = []
              itemMasterRecs.forEach((val, index) => {
                if (index === 0) return
                data.push({
                  id: val[0],
                  item_type: val[1],
                  name: val[2],
                  description: val[3],
                  amount_per_sec: val[4],
                  max_level: val[5],
                  max_amount_per_sec: val[6],
                  base_exp_per_level: val[7],
                  gained_exp: val[8],
                  shortening_min: val[9],
                })
              })

              const questions = Array(data.length).fill('(?,?,?,?,?,?,?,?,?,?)')
              const query =
                'INSERT INTO item_masters(id, item_type, name, description, amount_per_sec, max_level, max_amount_per_sec, base_exp_per_level, gained_exp, shortening_min) VALUES ' +
                questions.join(',') +
                ' ON DUPLICATE KEY UPDATE item_type=VALUES(item_type), name=VALUES(name), description=VALUES(description), amount_per_sec=VALUES(amount_per_sec), max_level=VALUES(max_level), max_amount_per_sec=VALUES(max_amount_per_sec), base_exp_per_level=VALUES(base_exp_per_level), gained_exp=VALUES(gained_exp), shortening_min=VALUES(shortening_min)'
              await conn.query(query, data.map((x) => Object.values(x)).flat())
            } else {
              console.log('Skip Update Master: itemMaster')
            }
          }

          // gacha
          {
            const { records: gachaRecs, error } = readFormFileToCSV(req, 'gachaMaster')
            if (error && error.message != ErrNoFormFile) {
              await conn.rollback()
              return errorResponse(res, 400, new Error('invalid gachaMaster'))
            }
            if (gachaRecs) {
              const data: any[] = []
              gachaRecs.forEach((val, index) => {
                if (index === 0) return
                data.push({
                  id: val[0],
                  name: val[1],
                  start_at: val[2],
                  end_at: val[3],
                  display_order: val[4],
                  created_at: val[5],
                })
              })

              const questions = Array(data.length).fill('(?,?,?,?,?,?)')
              const query =
                'INSERT INTO gacha_masters(id, name, start_at, end_at, display_order, created_at) VALUES ' +
                questions.join(',') +
                ' ON DUPLICATE KEY UPDATE name=VALUES(name), start_at=VALUES(start_at), end_at=VALUES(end_at), display_order=VALUES(display_order), created_at=VALUES(created_at)'
              await conn.query(query, data.map((x) => Object.values(x)).flat())
            } else {
              console.log('Skip Update Master: gachaMaster')
            }
          }

          // gacha item
          {
            const { records: gachaItemRecs, error } = readFormFileToCSV(req, 'gachaItemMaster')
            if (error && error.message != ErrNoFormFile) {
              await conn.rollback()
              return errorResponse(res, 400, new Error('invalid gachaItemMaster'))
            }
            if (gachaItemRecs) {
              const data: any[] = []
              gachaItemRecs.forEach((val, index) => {
                if (index === 0) return
                data.push({
                  id: val[0],
                  gacha_id: val[1],
                  item_type: val[2],
                  item_id: val[3],
                  amount: val[4],
                  weight: val[5],
                  created_at: val[6],
                })
              })

              const questions = Array(data.length).fill('(?,?,?,?,?,?,?)')
              const query =
                'INSERT INTO gacha_item_masters(id, gacha_id, item_type, item_id, amount, weight, created_at) VALUES ' +
                questions.join(',') +
                ' ON DUPLICATE KEY UPDATE gacha_id=VALUES(gacha_id), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), weight=VALUES(weight), created_at=VALUES(created_at)'
              await conn.query(query, data.map((x) => Object.values(x)).flat())
            } else {
              console.log('Skip Update Master: gachaItemMaster')
            }
          }

          // present all
          {
            const { records: presentAllRecs, error } = readFormFileToCSV(req, 'presentAllMaster')
            if (error && error.message != ErrNoFormFile) {
              await conn.rollback()
              return errorResponse(res, 400, new Error('invalid presentAllMaster'))
            }
            if (presentAllRecs) {
              const data: any[] = []
              presentAllRecs.forEach((val, index) => {
                if (index === 0) return
                data.push({
                  id: val[0],
                  registered_start_at: val[1],
                  registered_end_at: val[2],
                  item_type: val[3],
                  item_id: val[4],
                  amount: val[5],
                  present_message: val[6],
                  created_at: val[7],
                })
              })

              const questions = Array(data.length).fill('(?,?,?,?,?,?,?,?)')
              const query =
                'INSERT INTO present_all_masters(id, registered_start_at, registered_end_at, item_type, item_id, amount, present_message, created_at) VALUES ' +
                questions.join(',') +
                ' ON DUPLICATE KEY UPDATE registered_start_at=VALUES(registered_start_at), registered_end_at=VALUES(registered_end_at), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), present_message=VALUES(present_message), created_at=VALUES(created_at)'
              await conn.query(query, data.map((x) => Object.values(x)).flat())
            } else {
              console.log('Skip Update Master: presentAllMaster')
            }
          }

          // login bonuses
          {
            const { records: loginBonusRecs, error } = readFormFileToCSV(req, 'loginBonusMaster')
            if (error && error.message != ErrNoFormFile) {
              await conn.rollback()
              return errorResponse(res, 400, new Error('invalid loginBonusMaster'))
            }
            if (loginBonusRecs) {
              const data: any[] = []
              loginBonusRecs.forEach((val, index) => {
                if (index === 0) return
                const looped = val[4] === 'TRUE' ? 1 : 0
                data.push({
                  id: val[0],
                  start_at: val[1],
                  end_at: val[2],
                  column_count: val[3],
                  looped,
                  created_at: val[5],
                })
              })

              const questions = Array(data.length).fill('(?,?,?,?,?,?)')
              const query =
                'INSERT INTO login_bonus_masters(id, start_at, end_at, column_count, looped, created_at) VALUES ' +
                questions.join(',') +
                ' ON DUPLICATE KEY UPDATE start_at=VALUES(start_at), end_at=VALUES(end_at), column_count=VALUES(column_count), looped=VALUES(looped), created_at=VALUES(created_at)'
              await conn.query(query, data.map((x) => Object.values(x)).flat())
            } else {
              console.log('Skip Update Master: loginBonusMaster')
            }
          }

          // login bonus rewards
          {
            const { records: loginBonusRewardRecs, error } = readFormFileToCSV(req, 'loginBonusRewardMaster')
            if (error && error.message != ErrNoFormFile) {
              await conn.rollback()
              return errorResponse(res, 400, new Error('invalid loginBonusRewardMaster'))
            }
            if (loginBonusRewardRecs) {
              const data: any[] = []
              loginBonusRewardRecs.forEach((val, index) => {
                if (index === 0) return
                data.push({
                  id: val[0],
                  login_bonus_id: val[1],
                  reward_sequence: val[2],
                  item_type: val[3],
                  item_id: val[4],
                  amount: val[5],
                  created_at: val[6],
                })
              })

              const questions = Array(data.length).fill('(?,?,?,?,?,?,?)')
              const query =
                'INSERT INTO login_bonus_reward_masters(id, login_bonus_id, reward_sequence, item_type, item_id, amount, created_at) VALUES ' +
                questions.join(',') +
                ' ON DUPLICATE KEY UPDATE login_bonus_id=VALUES(login_bonus_id), reward_sequence=VALUES(reward_sequence), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), created_at=VALUES(created_at)'
              await conn.query(query, data.map((x) => Object.values(x)).flat())
            } else {
              console.log('Skip Update Master: loginBonusRewardMaster')
            }
          }

          const [[activeMaster]] = await conn.query<(VersionMasterRow & RowDataPacket)[]>(
            'SELECT * FROM version_masters WHERE status=1'
          )
          if (!activeMaster) {
            await conn.rollback()
            return errorResponse(res, 500, new Error('failed to fetch version_masters'))
          }

          await conn.commit()

          const payload: AdminUpdateMasterResponse = {
            versionMaster: toVersionMaster(activeMaster),
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

  // adminUser ユーザの詳細画面
  // GET /admin/user/:userId
  adminRouter.get(
    '/admin/user/:userId',
    adminSessionCheckMiddleware,
    wrap(async (req: Request, res: Response, _: NextFunction) => {
      try {
        const userId = getUserId(req)

        let query = 'SELECT * FROM users WHERE id=?'
        const [[user]] = await db.query<(UserRow & RowDataPacket)[]>(query, [userId])
        if (!user) {
          return errorResponse(res, 404, new Error(ErrUserNotFound))
        }

        query = 'SELECT * FROM user_devices WHERE user_id=?'
        const [devices] = await db.query<(UserDeviceRow & RowDataPacket)[]>(query, [userId])

        query = 'SELECT * FROM user_cards WHERE user_id=?'
        const [cards] = await db.query<(UserCardRow & RowDataPacket)[]>(query, [userId])

        query = 'SELECT * FROM user_decks WHERE user_id=?'
        const [decks] = await db.query<(UserDeckRow & RowDataPacket)[]>(query, [userId])

        query = 'SELECT * FROM user_items WHERE user_id=?'
        const [items] = await db.query<(UserItemRow & RowDataPacket)[]>(query, [userId])

        query = 'SELECT * FROM user_login_bonuses WHERE user_id=?'
        const [loginBonuses] = await db.query<(UserLoginBonusRow & RowDataPacket)[]>(query, [userId])

        query = 'SELECT * FROM user_presents WHERE user_id=?'
        const [presents] = await db.query<(UserPresentRow & RowDataPacket)[]>(query, [userId])

        query = 'SELECT * FROM user_present_all_received_history WHERE user_id=?'
        const [presentHistory] = await db.query<(UserPresentAllReceivedHistoryRow & RowDataPacket)[]>(query, [userId])

        const payload: AdminUserResponse = {
          user: toUser(user),
          userDevices: devices.map(toUserDevice),
          userCards: cards.map(toUserCard),
          userDecks: decks.map(toUserDeck),
          userItems: items.map(toUserItem),
          userLoginBonuses: loginBonuses.map(toUserLoginBonus),
          userPresents: presents.map(toUserPresent),
          userPresentAllReceivedHistory: presentHistory.map(toUserPresentAllReceivedHistory),
        }
        return successResponse(res, payload)
      } catch (error) {
        return errorResponse(res, 500, error as Error)
      }
    })
  )

  // adminBanUser ユーザBAN処理
  // POST /admin/user/:userId/ban
  adminRouter.post(
    '/admin/user/:userId/ban',
    adminSessionCheckMiddleware,
    wrap(async (req: Request, res: Response, _: NextFunction) => {
      try {
        const userId = getUserId(req)

        const requestAt = getRequestTime(req, res)
        if (!requestAt) {
          return errorResponse(res, 500, new Error(ErrGetRequestTime))
        }

        let query = 'SELECT * FROM users WHERE id=?'
        const [[user]] = await db.query<(UserRow & RowDataPacket)[]>(query, [userId])
        if (!user) {
          return errorResponse(res, 400, new Error(ErrUserNotFound))
        }

        const banId = await generateId()
        query =
          'INSERT user_bans(id, user_id, created_at, updated_at) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE updated_at = ?'
        await db.query(query, [banId, userId, requestAt, requestAt, requestAt])

        const payload: AdminBanUserResponse = {
          user: toUser(user),
        }
        return successResponse(res, payload)
      } catch (error) {
        return errorResponse(res, 500, error as Error)
      }
    })
  )

  return {
    adminRouter,
  }
}
