<?php

declare(strict_types=1);

namespace App\IsuConquest;

use Exception;
use Fig\Http\Message\StatusCodeInterface;
use JsonException;
use JsonSerializable;
use PDOException;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Ramsey\Uuid\Uuid;
use RuntimeException;
use Slim\Routing\RouteContext;

trait Common
{
    private string $errInvalidRequestBody = 'invalid request body';
    private string $errInvalidMasterVersion = 'invalid master version';
    private string $errInvalidItemType = 'invalid item type';
    private string $errInvalidToken = 'invalid token';
    private string $errGetRequestTime = 'failed to get request time';
    private string $errExpiredSession = 'session expired';
    private string $errUserNotFound = 'not found user';
    private string $errUserDeviceNotFound = 'not found user device';
    private string $errItemNotFound = 'not found item';
    private string $errLoginBonusRewardNotFound = 'not found login bonus reward';
    private string $errNoFormFile = 'no such file';
    private string $errUnauthorized = 'unauthorized user';
    private string $errForbidden = 'forbidden';
    private string $errGeneratePassword = 'failed to password hash';

    /**
     * getRequestTime リクエストを受けた時間をコンテキストからunixtimeで取得する
     *
     * @throws Exception
     */
    private function getRequestTime(Request $request): int
    {
        $v = $request->getAttribute('requestTime');

        return is_int($v) ? $v : throw new Exception($this->errGetRequestTime);
    }

    // //////////////////////////////////////
    // util

    /**
     * successResponse responds success.
     *
     * @throws JsonException
     */
    private function successResponse(Response $response, JsonSerializable|array $v): Response
    {
        $responseBody = json_encode($v, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_THROW_ON_ERROR);
        $response->getBody()->write($responseBody);

        return $response->withStatus(StatusCodeInterface::STATUS_OK)
            ->withHeader('Content-Type', 'application/json; charset=UTF-8');
    }

    /**
     * generateID uniqueなIDを生成する
     * @throws PDOException
     * @throws RuntimeException
     */
    private function generateID(): int
    {
        $hasNewTransaction = false;
        if (!$this->db->inTransaction()) {
            $this->db->beginTransaction();
            $hasNewTransaction = true;
        }

        try {
            /** @var ?PDOException $updateErr */
            $updateErr = null;
            for ($i = 0; $i < 100; $i++) {
                try {
                    $this->db->exec('UPDATE id_generator SET id=LAST_INSERT_ID(id+1)');
                } catch (PDOException $e) {
                    if ($e->getCode() === 1213) {
                        $updateErr = $e;
                        continue;
                    }
                    throw $e;
                }

                $id =  $this->db->lastInsertId();
                if ($id === false) {
                    throw new RuntimeException('failed to generate id: ', $this->db->errorInfo()[2]);
                }

                if ($hasNewTransaction) {
                    $this->db->commit();
                }
                return (int)$id;
            }

            throw new RuntimeException('failed to generate id: ' . $updateErr->getMessage(), previous: $updateErr);
        } catch (PDOException | RuntimeException $e) {
            if ($hasNewTransaction) {
                $this->db->rollBack();
            }
            throw $e;
        }
    }

    /**
     * generateUUID
     */
    private function generateUUID(): string
    {
        return UUid::uuid4()->toString();
    }

    /**
     * getUserID gets userID by path param.
     *
     * @throws RuntimeException
     */
    private function getUserID(Request $request): int
    {
        $userIDStr = RouteContext::fromRequest($request)
            ->getRoute()
            ->getArgument('userID');
        if (is_null($userIDStr)) {
            throw new RuntimeException();
        }

        $userID = filter_var($userIDStr, FILTER_VALIDATE_INT);
        if (!is_int($userID)) {
            throw new RuntimeException();
        }

        return $userID;
    }
}
