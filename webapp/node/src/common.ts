import { Request, Response } from 'express'
import mysql, { OkPacket } from 'mysql2/promise'
import { v4 as uuidv4 } from 'uuid'

// successResponse responds success.
export function successResponse(res: Response, v: any) {
  //console.log(`[OK] status=200`)
  res.status(200).json(v)
}

// errorResponse returns error.
export function errorResponse(res: Response, statusCode: number, err: Error) {
  const stack = err.stack?.split('\n')[1].trim()
  console.error(`status=${statusCode.toString()}, err=${err} ${stack}`)
  res.status(statusCode).json({
    status_code: statusCode,
    message: err.message,
  })
}

// getRequestTime リクエストを受けた時間をコンテキストからunixtimeで取得する
export function getRequestTime(_: Request, res: Response): number {
  const requestTime: number = res.locals.requestTime
  if (!requestTime) {
    return 0
  }

  return requestTime
}

// getUserID gets userID by path param.
export function getUserId(req: Request): number {
  return parseInt(req.params.userId, 10)
}

// generateSessionID
export function generateUUID() {
  return uuidv4()
}

export const useGenerateId = (db: mysql.Pool) => {
  // generateID uniqueなIDを生成する
  const generateId = async (): Promise<number> => {
    let id = 0
    let updateErr: any
    for (const _ of Array(100)) {
      try {
        const [result] = await db.query<OkPacket>('UPDATE id_generator SET id=LAST_INSERT_ID(id+1)')

        id = result.insertId
        return id
      } catch (error: any) {
        if (error.errno && error.errno === 1213) {
          updateErr = error
        }
      }
    }

    throw new Error(`failed to generate id: ${updateErr.toString()}`)
  }

  return {
    generateId,
  }
}
