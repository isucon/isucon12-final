<?php

declare(strict_types=1);

namespace App\IsuConquest;

use Exception;
use Fig\Http\Message\StatusCodeInterface;
use PDO;
use PDOException;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Message\UploadedFileInterface;
use Psr\Http\Server\RequestHandlerInterface as RequestHandler;
use Psr\Log\LoggerInterface as Logger;
use RuntimeException;
use Slim\Exception\HttpBadRequestException;
use Slim\Exception\HttpInternalServerErrorException;
use Slim\Exception\HttpNotFoundException;
use Slim\Exception\HttpUnauthorizedException;

final class AdminHandler
{
    use Common;

    public function __construct(
        private readonly PDO $db,
        private readonly Logger $logger,
    ) {
    }

    // //////////////////////////////////////
    // admin

    /**
     * adminMiddleware
     */
    public function adminMiddleware(Request $request, RequestHandler $handler): Response
    {
        $request = $request->withAttribute('requestTime', time());

        // next
        return $handler->handle($request);
    }

    /**
     * adminSessionCheckMiddleware
     */
    public function adminSessionCheckMiddleware(Request $request, RequestHandler $handler): Response
    {
        if (!$request->hasHeader('x-session')) {
            throw new HttpUnauthorizedException($request, $this->errUnauthorized);
        }
        $sessID = $request->getHeader('x-session')[0];

        $query = 'SELECT * FROM admin_sessions WHERE session_id=? AND deleted_at IS NULL';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->execute([$sessID]);
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpUnauthorizedException($request, $this->errUnauthorized);
        }
        $adminSession = Session::fromDBRow($row);

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        if ($adminSession->expiredAt < $requestAt) {
            $query = 'UPDATE admin_sessions SET deleted_at=? WHERE session_id=?';
            try {
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
                $stmt->bindValue(2, $sessID);
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
            throw new HttpUnauthorizedException($request, $this->errExpiredSession);
        }

        // next
        return $handler->handle($request);
    }

    /**
     * adminLogin 管理者権限ログイン
     * POST /admin/login
     */
    public function adminLogin(Request $request, Response $response): Response
    {
        try {
            $req = new AdminLoginRequest($request->getBody()->getContents());
        } catch (Exception $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $this->db->beginTransaction();

        $query = 'SELECT * FROM admin_users WHERE id=?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $req->userID, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request, $this->errUserNotFound);
        }
        $user = AdminUser::fromDBRow($row);

        // verify password
        if (!password_verify($req->password, $user->password)) {
            throw new HttpUnauthorizedException($request, $this->errUnauthorized);
        }

        $query = 'UPDATE admin_users SET last_activated_at=?, updated_at=? WHERE id=?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(3, $req->userID, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        // すでにあるsessionをdeleteにする
        $query = 'UPDATE admin_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(2, $req->userID, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        // create session
        try {
            $sID = $this->generateID();
            $sessID = $this->generateUUID();
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        $sess = new Session(
            id: $sID,
            userID: $req->userID,
            sessionID: $sessID,
            createdAt: $requestAt,
            updatedAt: $requestAt,
            expiredAt: $requestAt + 86400,
        );

        $query = 'INSERT INTO admin_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $sess->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $sess->userID, PDO::PARAM_INT);
            $stmt->bindValue(3, $sess->sessionID);
            $stmt->bindValue(4, $sess->createdAt, PDO::PARAM_INT);
            $stmt->bindValue(5, $sess->updatedAt, PDO::PARAM_INT);
            $stmt->bindValue(6, $sess->expiredAt, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $this->db->commit();

        return $this->successResponse($response, new AdminLoginResponse(
            adminSession: $sess,
        ));
    }

    /**
     * adminLogout 管理者権限ログアウト
     * DELETE /admin/logout
     */
    public function adminLogout(Request $request, Response $response): Response
    {
        $sessID = $request->getHeader('x-session')[0];

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        // すでにあるsessionをdeleteにする
        $query = 'UPDATE admin_sessions SET deleted_at=? WHERE session_id=? AND deleted_at IS NULL';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(2, $sessID);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        return $response->withStatus(StatusCodeInterface::STATUS_NO_CONTENT);
    }

    /**
     * adminListMaster マスタデータ閲覧
     * GET /admin/master
     */
    public function adminListMaster(Request $request, Response $response): Response
    {
        /** @var list<VersionMaster> $masterVersions */
        $masterVersions = [];
        try {
            $stmt = $this->db->query('SELECT * FROM version_masters');
            while ($row = $stmt->fetch()) {
                $masterVersions[] = VersionMaster::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        /** @var list<ItemMaster> $items */
        $items = [];
        try {
            $stmt = $this->db->query('SELECT * FROM item_masters');
            while ($row = $stmt->fetch()) {
                $items[] = ItemMaster::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        /** @var list<GachaMaster> $gachas */
        $gachas = [];
        try {
            $stmt = $this->db->query('SELECT * FROM gacha_masters');
            while ($row = $stmt->fetch()) {
                $gachas[] = GachaMaster::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        /** @var list<GachaItemMaster> $gachaItems */
        $gachaItems = [];
        try {
            $stmt = $this->db->query('SELECT * FROM gacha_item_masters');
            while ($row = $stmt->fetch()) {
                $gachaItems[] = GachaItemMaster::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        /** @var list<PresentAllMaster> $presentAlls */
        $presentAlls = [];
        try {
            $stmt = $this->db->query('SELECT * FROM present_all_masters');
            while ($row = $stmt->fetch()) {
                $presentAlls[] = PresentAllMaster::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        /** @var list<LoginBonusMaster> $loginBonuses */
        $loginBonuses = [];
        try {
            $stmt = $this->db->query('SELECT * FROM login_bonus_masters');
            while ($row = $stmt->fetch()) {
                $loginBonuses[] = LoginBonusMaster::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        /** @var list<LoginBonusRewardMaster> $loginBonusRewards */
        $loginBonusRewards = [];
        try {
            $stmt = $this->db->query('SELECT * FROM login_bonus_reward_masters');
            while ($row = $stmt->fetch()) {
                $loginBonusRewards[] = LoginBonusRewardMaster::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        return $this->successResponse($response, new AdminListMasterResponse(
            versionMaster: $masterVersions,
            items: $items,
            gachas: $gachas,
            gachaItems: $gachaItems,
            presentAlls: $presentAlls,
            loginBonuses: $loginBonuses,
            loginBonusRewards: $loginBonusRewards,
        ));
    }

    /**
     * adminUpdateMaster マスタデータ更新
     * PUT /admin/master
     */
    public function adminUpdateMaster(Request $request, Response $response): Response
    {
        $this->db->beginTransaction();
        // version master
        try {
            $versionMasterRecs = $this->readFormFileToCSV($request, 'versionMaster');
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err !== $this->errNoFormFile) {
                throw new HttpBadRequestException($request, $e->getMessage(), $e);
            }
        }
        if (isset($versionMasterRecs)) {
            /** @var list<string> $valuesClause */
            $valuesClause = [];
            /** @var list<array<string, mixed>> $data */
            $data = [];
            foreach ($versionMasterRecs as $i => $v) {
                if ($i === 0) {
                    continue;
                }
                $valuesClause[] = '(?, ?, ?)';
                $data[] = [
                    'id'             => (int)$v[0],
                    'status'         => (int)$v[1],
                    'master_version' => $v[2],
                ];
            }

            $query = 'INSERT INTO version_masters(id, status, master_version) VALUES ' . implode(', ', $valuesClause) . ' ON DUPLICATE KEY UPDATE status=VALUES(status), master_version=VALUES(master_version)';
            try {
                $stmt = $this->db->prepare($query);
                foreach ($data as $i => $v) {
                    $stmt->bindValue(3 * $i + 1, $v['id'], PDO::PARAM_INT);
                    $stmt->bindValue(3 * $i + 2, $v['status'], PDO::PARAM_INT);
                    $stmt->bindValue(3 * $i + 3, $v['master_version']);
                }
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
        } else {
            $this->logger->debug('Skip Update Master: versionMaster');
        }

        // item
        try {
            $itemMasterRecs = $this->readFormFileToCSV($request, 'itemMaster');
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err !== $this->errNoFormFile) {
                throw new HttpBadRequestException($request, $e->getMessage(), $e);
            }
        }
        if (isset($itemMasterRecs)) {
            /** @var list<string> $valuesClause */
            $valuesClause = [];
            /** @var list<array<string, mixed>> $data */
            $data = [];
            foreach ($itemMasterRecs as $i => $v) {
                if ($i === 0) {
                    continue;
                }
                $valuesClause[] = '(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)';
                $data[] = [
                    'id'                 => (int)$v[0],
                    'item_type'          => (int)$v[1],
                    'name'               => $v[2],
                    'description'        => $v[3],
                    'amount_per_sec'     => (int)$v[4],
                    'max_level'          => (int)$v[5],
                    'max_amount_per_sec' => (int)$v[6],
                    'base_exp_per_level' => (int)$v[7],
                    'gained_exp'         => (int)$v[8],
                    'shortening_min'     => (int)$v[9],
                ];
            }

            $query = implode(' ', [
                'INSERT INTO item_masters(id, item_type, name, description, amount_per_sec, max_level, max_amount_per_sec, base_exp_per_level, gained_exp, shortening_min)',
                'VALUES ' . implode(', ', $valuesClause),
                'ON DUPLICATE KEY UPDATE item_type=VALUES(item_type), name=VALUES(name), description=VALUES(description), amount_per_sec=VALUES(amount_per_sec), max_level=VALUES(max_level), max_amount_per_sec=VALUES(max_amount_per_sec), base_exp_per_level=VALUES(base_exp_per_level), gained_exp=VALUES(gained_exp), shortening_min=VALUES(shortening_min)',
            ]);
            try {
                $stmt = $this->db->prepare($query);
                foreach ($data as $i => $v) {
                    $stmt->bindValue(10 * $i + 1, $v['id'], PDO::PARAM_INT);
                    $stmt->bindValue(10 * $i + 2, $v['item_type'], PDO::PARAM_INT);
                    $stmt->bindValue(10 * $i + 3, $v['name']);
                    $stmt->bindValue(10 * $i + 4, $v['description'], PDO::PARAM_INT);
                    $stmt->bindValue(10 * $i + 5, $v['amount_per_sec'], PDO::PARAM_INT);
                    $stmt->bindValue(10 * $i + 6, $v['max_level'], PDO::PARAM_INT);
                    $stmt->bindValue(10 * $i + 7, $v['max_amount_per_sec'], PDO::PARAM_INT);
                    $stmt->bindValue(10 * $i + 8, $v['base_exp_per_level'], PDO::PARAM_INT);
                    $stmt->bindValue(10 * $i + 9, $v['gained_exp'], PDO::PARAM_INT);
                    $stmt->bindValue(10 * $i + 10, $v['shortening_min'], PDO::PARAM_INT);
                }
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
        } else {
            $this->logger->debug('Skip Update Master: itemMaster');
        }

        // gacha
        try {
            $gachaRecs = $this->readFormFileToCSV($request, 'gachaMaster');
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err !== $this->errNoFormFile) {
                throw new HttpBadRequestException($request, $e->getMessage(), $e);
            }
        }
        if (isset($gachaRecs)) {
            /** @var list<string> $valuesClause */
            $valuesClause = [];
            /** @var list<array<string, mixed>> $data */
            $data = [];
            foreach ($gachaRecs as $i => $v) {
                if ($i === 0) {
                    continue;
                }
                $valuesClause[] = '(?, ?, ?, ?, ?, ?)';
                $data[] = [
                    'id'            => (int)$v[0],
                    'name'          => $v[1],
                    'start_at'      => (int)$v[2],
                    'end_at'        => (int)$v[3],
                    'display_order' => (int)$v[4],
                    'created_at'    => (int)$v[5],
                ];
            }

            $query = implode(' ', [
                'INSERT INTO gacha_masters(id, name, start_at, end_at, display_order, created_at)',
                'VALUES ' . implode(', ', $valuesClause),
                'ON DUPLICATE KEY UPDATE name=VALUES(name), start_at=VALUES(start_at), end_at=VALUES(end_at), display_order=VALUES(display_order), created_at=VALUES(created_at)',
            ]);
            try {
                $stmt = $this->db->prepare($query);
                foreach ($data as $i => $v) {
                    $stmt->bindValue(6 * $i + 1, $v['id'], PDO::PARAM_INT);
                    $stmt->bindValue(6 * $i + 2, $v['name']);
                    $stmt->bindValue(6 * $i + 3, $v['start_at'], PDO::PARAM_INT);
                    $stmt->bindValue(6 * $i + 4, $v['end_at'], PDO::PARAM_INT);
                    $stmt->bindValue(6 * $i + 5, $v['display_order'], PDO::PARAM_INT);
                    $stmt->bindValue(6 * $i + 6, $v['created_at'], PDO::PARAM_INT);
                }
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
        } else {
            $this->logger->debug('Skip Update Master: gachaMaster');
        }

        // gacha item
        try {
            $gachaItemRecs = $this->readFormFileToCSV($request, 'gachaItemMaster');
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err !== $this->errNoFormFile) {
                throw new HttpBadRequestException($request, $e->getMessage(), $e);
            }
        }
        if (isset($gachaItemRecs)) {
            /** @var list<string> $valuesClause */
            $valuesClause = [];
            /** @var list<array<string, mixed>> $data */
            $data = [];
            foreach ($gachaItemRecs as $i => $v) {
                if ($i === 0) {
                    continue;
                }
                $valuesClause[] = '(?, ?, ?, ?, ?, ?, ?)';
                $data[] = [
                    'id'         => (int)$v[0],
                    'gacha_id'   => (int)$v[1],
                    'item_type'  => (int)$v[2],
                    'item_id'    => (int)$v[3],
                    'amount'     => (int)$v[4],
                    'weight'     => (int)$v[5],
                    'created_at' => (int)$v[6],
                ];
            }

            $query = implode(' ', [
                'INSERT INTO gacha_masters(id, name, start_at, end_at, display_order, created_at)',
                'VALUES ' . implode(', ', $valuesClause),
                'ON DUPLICATE KEY UPDATE name=VALUES(name), start_at=VALUES(start_at), end_at=VALUES(end_at), display_order=VALUES(display_order), created_at=VALUES(created_at)',
            ]);
            try {
                $stmt = $this->db->prepare($query);
                foreach ($data as $i => $v) {
                    $stmt->bindValue(7 * $i + 1, $v['id'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 2, $v['gacha_id'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 3, $v['item_type'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 4, $v['item_id'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 5, $v['amount'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 6, $v['weight'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 7, $v['created_at'], PDO::PARAM_INT);
                }
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
        } else {
            $this->logger->debug('Skip Update Master: gachaItemMaster');
        }

        // present all
        try {
            $presentAllRecs = $this->readFormFileToCSV($request, 'presentAllMaster');
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err !== $this->errNoFormFile) {
                throw new HttpBadRequestException($request, $e->getMessage(), $e);
            }
        }
        if (isset($presentAllRecs)) {
            /** @var list<string> $valuesClause */
            $valuesClause = [];
            /** @var list<array<string, mixed>> $data */
            $data = [];
            foreach ($presentAllRecs as $i => $v) {
                if ($i === 0) {
                    continue;
                }
                $valuesClause[] = '(?, ?, ?, ?, ?, ?, ?, ?)';
                $data[] = [
                    'id'                  => (int)$v[0],
                    'registered_start_at' => (int)$v[1],
                    'registered_end_at'   => (int)$v[2],
                    'item_type'           => (int)$v[3],
                    'item_id'             => (int)$v[4],
                    'amount'              => (int)$v[5],
                    'present_message'     => $v[6],
                    'created_at'          => (int)$v[7],
                ];
            }

            $query = implode(' ', [
                'INSERT INTO present_all_masters(id, registered_start_at, registered_end_at, item_type, item_id, amount, present_message, created_at)',
                'VALUES ' . implode(', ', $valuesClause),
                'ON DUPLICATE KEY UPDATE registered_start_at=VALUES(registered_start_at), registered_end_at=VALUES(registered_end_at), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), present_message=VALUES(present_message), created_at=VALUES(created_at)',
            ]);
            try {
                $stmt = $this->db->prepare($query);
                foreach ($data as $i => $v) {
                    $stmt->bindValue(8 * $i + 1, $v['id'], PDO::PARAM_INT);
                    $stmt->bindValue(8 * $i + 2, $v['registered_start_at'], PDO::PARAM_INT);
                    $stmt->bindValue(8 * $i + 3, $v['registered_end_at'], PDO::PARAM_INT);
                    $stmt->bindValue(8 * $i + 4, $v['item_type'], PDO::PARAM_INT);
                    $stmt->bindValue(8 * $i + 5, $v['item_id'], PDO::PARAM_INT);
                    $stmt->bindValue(8 * $i + 6, $v['amount'], PDO::PARAM_INT);
                    $stmt->bindValue(8 * $i + 7, $v['present_message']);
                    $stmt->bindValue(8 * $i + 8, $v['created_at'], PDO::PARAM_INT);
                }
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
        } else {
            $this->logger->debug('Skip Update Master: presentAllMaster');
        }

        // login bonuses
        try {
            $loginBonusRecs = $this->readFormFileToCSV($request, 'loginBonusMaster');
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err !== $this->errNoFormFile) {
                throw new HttpBadRequestException($request, $e->getMessage(), $e);
            }
        }
        if (isset($loginBonusRecs)) {
            /** @var list<string> $valuesClause */
            $valuesClause = [];
            /** @var list<array<string, mixed>> $data */
            $data = [];
            foreach ($loginBonusRecs as $i => $v) {
                if ($i === 0) {
                    continue;
                }
                $looped = 0;
                if ($v[4] === 'TRUE') {
                    $looped = 1;
                }
                $valuesClause[] = '(?, ?, ?, ?, ?, ?)';
                $data[] = [
                    'id'           => (int)$v[0],
                    'start_at'     => (int)$v[1],
                    'end_at'       => (int)$v[2],
                    'column_count' => (int)$v[3],
                    'looped'       => $looped,
                    'created_at'   => (int)$v[5],
                ];
            }

            $query = implode(' ', [
                'INSERT INTO login_bonus_masters(id, start_at, end_at, column_count, looped, created_at)',
                'VALUES ' . implode(', ', $valuesClause),
                'ON DUPLICATE KEY UPDATE start_at=VALUES(start_at), end_at=VALUES(end_at), column_count=VALUES(column_count), looped=VALUES(looped), created_at=VALUES(created_at)',
            ]);
            try {
                $stmt = $this->db->prepare($query);
                foreach ($data as $i => $v) {
                    $stmt->bindValue(6 * $i + 1, $v['id'], PDO::PARAM_INT);
                    $stmt->bindValue(6 * $i + 2, $v['start_at'], PDO::PARAM_INT);
                    $stmt->bindValue(6 * $i + 3, $v['end_at'], PDO::PARAM_INT);
                    $stmt->bindValue(6 * $i + 4, $v['column_count'], PDO::PARAM_INT);
                    $stmt->bindValue(6 * $i + 5, $v['looped'], PDO::PARAM_INT);
                    $stmt->bindValue(6 * $i + 6, $v['created_at'], PDO::PARAM_INT);
                }
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
        } else {
            $this->logger->debug('Skip Update Master: loginBonusMaster');
        }

        // login bonus rewards
        try {
            $loginBonusRewardRecs = $this->readFormFileToCSV($request, 'loginBonusRewardMaster');
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err !== $this->errNoFormFile) {
                throw new HttpBadRequestException($request, $e->getMessage(), $e);
            }
        }
        if (isset($loginBonusRewardRecs)) {
            /** @var list<string> $valuesClause */
            $valuesClause = [];
            /** @var list<array<string, mixed>> $data */
            $data = [];
            foreach ($loginBonusRewardRecs as $i => $v) {
                if ($i === 0) {
                    continue;
                }
                $valuesClause[] = '(?, ?, ?, ?, ?, ?, ?)';
                $data[] = [
                    'id'              => (int)$v[0],
                    'login_bonus_id'  => (int)$v[1],
                    'reward_sequence' => (int)$v[2],
                    'item_type'       => (int)$v[3],
                    'item_id'         => (int)$v[4],
                    'amount'          => (int)$v[5],
                    'created_at'      => (int)$v[6],
                ];
            }

            $query = implode(' ', [
                'INSERT INTO login_bonus_reward_masters(id, login_bonus_id, reward_sequence, item_type, item_id, amount, created_at)',
                'VALUES ' . implode(', ', $valuesClause),
                'ON DUPLICATE KEY UPDATE login_bonus_id=VALUES(login_bonus_id), reward_sequence=VALUES(reward_sequence), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), created_at=VALUES(created_at)',
            ]);
            try {
                $stmt = $this->db->prepare($query);
                foreach ($data as $i => $v) {
                    $stmt->bindValue(7 * $i + 1, $v['id'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 2, $v['login_bonus_id'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 3, $v['reward_sequence'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 4, $v['item_type'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 5, $v['item_id'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 6, $v['amount'], PDO::PARAM_INT);
                    $stmt->bindValue(7 * $i + 7, $v['created_at'], PDO::PARAM_INT);
                }
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
        } else {
            $this->logger->debug('Skip Update Master: loginBonusRewardMaster');
        }

        try {
            $row = $this->db->query('SELECT * FROM version_masters WHERE status=1')
                ->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpInternalServerErrorException($request);
        }
        $activeMaster = VersionMaster::fromDBRow($row);

        $this->db->commit();

        return $this->successResponse($response, new AdminUpdateMasterResponse(
            versionMaster: $activeMaster,
        ));
    }

    /**
     * readFromFileToCSV ファイルからcsvレコードを取得する
     *
     * @return list<list<string>>
     * @throws RuntimeException
     */
    private function readFormFileToCSV(Request $request, string $name): array
    {
        /** @var UploadedFileInterface $file */
        $file = $request->getUploadedFiles()[$name] ?? throw new RuntimeException($this->errNoFormFile);
        if ($file->getError() !== UPLOAD_ERR_OK) {
            throw new RuntimeException();
        }

        /** @var list<list<string>> $records */
        $records = [];
        foreach (explode("\n", $file->getStream()->getContents()) as $row) {
            if ($row === '') {
                continue;
            }
            $records[] = str_getcsv($row);
        }

        return $records;
    }

    /**
     * adminUser ユーザの詳細画面
     * GET /admin/user/{userID}
     */
    public function adminUser(Request $request, Response $response): Response
    {
        try {
            $userID = $this->getUserID($request);
        } catch (RuntimeException $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        $query = 'SELECT * FROM users WHERE id=?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request, $this->errUserNotFound);
        }
        $user = User::fromDBRow($row);

        $query = 'SELECT * FROM user_devices WHERE user_id=?';
        /** @var list<UserDevice> $devices */
        $devices = [];
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $devices[] = UserDevice::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $query = 'SELECT * FROM user_cards WHERE user_id=?';
        /** @var list<UserCard> $cards */
        $cards = [];
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $cards[] = UserCard::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $query = 'SELECT * FROM user_decks WHERE user_id=?';
        /** @var list<UserDeck> $decks */
        $decks = [];
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $decks[] = UserDeck::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $query = 'SELECT * FROM user_items WHERE user_id=?';
        /** @var list<UserItem> $items */
        $items = [];
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $items[] = UserItem::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $query = 'SELECT * FROM user_login_bonuses WHERE user_id=?';
        /** @var list<UserLoginBonus> $loginBonuses */
        $loginBonuses = [];
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $loginBonuses[] = UserLoginBonus::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $query = 'SELECT * FROM user_presents WHERE user_id=?';
        /** @var list<UserPresent> $presents */
        $presents = [];
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $presents[] = UserPresent::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $query = 'SELECT * FROM user_present_all_received_history WHERE user_id=?';
        /** @var list<UserPresentAllReceivedHistory> $presentHistory */
        $presentHistory = [];
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $presentHistory[] = UserPresentAllReceivedHistory::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        return $this->successResponse($response, new AdminUserResponse(
            user: $user,
            userDevices: $devices,
            userCards: $cards,
            userDecks: $decks,
            userItems: $items,
            userLoginBonuses: $loginBonuses,
            userPresents: $presents,
            userPresentAllReceivedHistory: $presentHistory,
        ));
    }

    /**
     * adminBanUser ユーザBAN処理
     * POST /admin/user/{userId}/ban
     */
    public function adminBanUser(Request $request, Response $response): Response
    {
        try {
            $userID = $this->getUserID($request);
        } catch (RuntimeException $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $query = 'SELECT * FROM users WHERE id=?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request, $this->errUserNotFound);
        }
        $user = User::fromDBRow($row);

        try {
            $banID = $this->generateID();
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        $query = 'INSERT user_bans(id, user_id, created_at, updated_at) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE updated_at = ?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $banID, PDO::PARAM_INT);
            $stmt->bindValue(2, $userID, PDO::PARAM_INT);
            $stmt->bindValue(3, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(4, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(5, $requestAt, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        return $this->successResponse($response, new AdminBanUserResponse(
            user: $user,
        ));
    }
}
