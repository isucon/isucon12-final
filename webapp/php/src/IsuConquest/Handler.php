<?php

declare(strict_types=1);

namespace App\IsuConquest;

use DateTimeImmutable;
use Exception;
use Fig\Http\Message\StatusCodeInterface;
use PDO;
use PDOException;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\RequestHandlerInterface as RequestHandler;
use Psr\Log\LoggerInterface as Logger;
use RuntimeException;
use Slim\Exception\HttpBadRequestException;
use Slim\Exception\HttpException;
use Slim\Exception\HttpForbiddenException;
use Slim\Exception\HttpInternalServerErrorException;
use Slim\Exception\HttpNotFoundException;
use Slim\Exception\HttpUnauthorizedException;

final class Handler
{
    use Common;

    private const DECK_CARD_NUMBER = 3;
    private const PRESENT_COUNT_PER_PAGE = 100;

    private const SQL_DIRECTORY = __DIR__ . '/../../../sql/';

    public function __construct(
        private readonly PDO $db,
        private readonly Logger $logger,
    ) {
    }

    /**
     * apiMiddleware
     */
    public function apiMiddleware(Request $request, RequestHandler $handler): Response
    {
        $requestAt = DateTimeImmutable::createFromFormat(DATE_RFC1123, $request->getHeader('x-isu-date')[0] ?? '');
        if ($requestAt === false) {
            $requestAt = new DateTimeImmutable();
        }
        $request = $request->withAttribute('requestTime', $requestAt->getTimestamp());

        // マスタ確認
        $query = 'SELECT * FROM version_masters WHERE status=1';
        try {
            $stmt = $this->db->query($query);
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request, 'active master version is not found');
        }
        $masterVersion = VersionMaster::fromDBRow($row);

        if (!$request->hasHeader('x-master-version') || $masterVersion->masterVersion !== $request->getHeader('x-master-version')[0]) {
            throw new HttpException($request, $this->errInvalidMasterVersion, StatusCodeInterface::STATUS_UNPROCESSABLE_ENTITY);
        }

        // check ban
        try {
            $userID = $this->getUserID($request);
        } catch (RuntimeException) {
            $userID = null;
        }
        if (!is_null($userID) && $userID !== 0) {
            try {
                $isBan = $this->checkBan($userID);
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
            if ($isBan) {
                throw new HttpUnauthorizedException($request, $this->errUnauthorized);
            }
        }

        // next
        return $handler->handle($request);
    }

    /**
     * checkSessionMiddleware
     */
    public function checkSessionMiddleware(Request $request, RequestHandler $handler): Response
    {
        $sessID = $request->getHeader('x-session')[0] ?? '';
        if ($sessID === '') {
            throw new HttpUnauthorizedException($request, $this->errUnauthorized);
        }

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

        $query = 'SELECT * FROM user_sessions WHERE session_id=? AND deleted_at IS NULL';
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
        $userSession = Session::fromDBRow($row);

        if ($userSession->userID !== $userID) {
            throw new HttpForbiddenException($request, $this->errForbidden);
        }

        if ($userSession->expiredAt < $requestAt) {
            $query = 'UPDATE user_sessions SET deleted_at=? WHERE session_id=?';
            try {
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
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
     * checkOneTimeToken
     *
     * @throws PDOException
     * @throws RuntimeException
     */
    private function checkOneTimeToken(string $token, int $tokenType, int $requestAt): void
    {
        $query = 'SELECT * FROM user_one_time_tokens WHERE token=? AND token_type=? AND deleted_at IS NULL';
        $stmt = $this->db->prepare($query);
        $stmt->bindValue(1, $token);
        $stmt->bindValue(2, $tokenType, PDO::PARAM_INT);
        $stmt->execute();
        $row = $stmt->fetch();
        if ($row === false) {
            throw new RuntimeException($this->errInvalidToken);
        }
        $tk = UserOneTimeToken::fromDBRow($row);

        if ($tk->expiredAt < $requestAt) {
            $query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?';
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(2, $token);
            $stmt->execute();
            throw new RuntimeException($this->errInvalidToken);
        }

        // 使ったトークンを失効する
        $query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE token=?';
        $stmt = $this->db->prepare($query);
        $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
        $stmt->bindValue(2, $token);
        $stmt->execute();
    }

    /**
     * checkViewerID
     *
     * @throws PDOException
     * @throws RuntimeException
     */
    private function checkViewerID(int $userID, string $viewerID): void
    {
        $query = 'SELECT * FROM user_devices WHERE user_id=? AND platform_id=?';

        $stmt = $this->db->prepare($query);
        $stmt->bindValue(1, $userID, PDO::PARAM_INT);
        $stmt->bindValue(2, $viewerID);
        $stmt->execute();
        if ($stmt->fetch() === false) {
            throw new RuntimeException($this->errUserDeviceNotFound);
        }
    }

    /**
     * checkBan
     *
     * @throws PDOException
     */
    private function checkBan(int $userID): bool
    {
        $query = 'SELECT * FROM user_bans WHERE user_id=?';
        $stmt = $this->db->prepare($query);
        $stmt->bindValue(1, $userID, PDO::PARAM_INT);
        $stmt->execute();
        $row = $stmt->fetch();
        if ($row === false) {
            return false;
        }

        return true;
    }

    /**
     * loginProcess ログイン処理
     *
     * @return array{User, list<UserLoginBonus>, list<UserPresent>}
     * @throws PDOException
     * @throws RuntimeException
     */
    private function loginProcess(int $userID, int $requestAt): array
    {
        $query = 'SELECT * FROM users WHERE id=?';
        $stmt = $this->db->prepare($query);
        $stmt->bindValue(1, $userID, PDO::PARAM_INT);
        $stmt->execute();
        $row = $stmt->fetch();
        if ($row === false) {
            throw new RuntimeException($this->errUserNotFound);
        }
        $user = User::fromDBRow($row);

        // ログインボーナス処理
        $loginBonuses = $this->obtainLoginBonus($userID, $requestAt);

        // 全員プレゼント取得
        $allPresents = $this->obtainPresent($userID, $requestAt);

        $stmt = $this->db->prepare('SELECT isu_coin FROM users WHERE id=?');
        $stmt->bindValue(1, $user->id, PDO::PARAM_INT);
        $stmt->execute();
        $isuCoin = $stmt->fetchColumn();
        if ($isuCoin === false) {
            throw new RuntimeException($this->errUserNotFound);
        }
        $user->isuCoin = $isuCoin;

        $user->updatedAt = $requestAt;
        $user->lastActivatedAt = $requestAt;

        $query = 'UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?';
        $stmt = $this->db->prepare($query);
        $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
        $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
        $stmt->bindValue(3, $userID, PDO::PARAM_INT);

        return [$user, $loginBonuses, $allPresents];
    }

    /**
     * isCompleteTodayLogin ログイン処理が終わっているか
     */
    private function isCompleteTodayLogin(int $lastActivatedAt, int $requestAt): bool
    {
        return date(format: 'Ymd', timestamp: $lastActivatedAt) === date(format: 'Ymd', timestamp: $requestAt);
    }

    /**
     * obtainLoginBonus
     *
     * @return list<UserLoginBonus>
     * @throws PDOException
     * @throws RuntimeException
     */
    private function obtainLoginBonus(int $userID, int $requestAt): array
    {
        // login bonus masterから有効なログインボーナスを取得
        /** @var list<LoginBonusMaster> $loginBonuses */
        $loginBonuses = [];
        $query = 'SELECT * FROM login_bonus_masters WHERE start_at <= ? AND end_at >= ?';
        $stmt = $this->db->prepare($query);
        $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
        $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
        $stmt->execute();
        while ($row = $stmt->fetch()) {
            $loginBonuses[] = LoginBonusMaster::fromDBRow($row);
        }

        /** @var list<UserLoginBonus> $sendLoginBonuses */
        $sendLoginBonuses = [];

        foreach ($loginBonuses as $bonus) {
            $initBonus = false;
            // ボーナスの進捗取得
            $query = 'SELECT * FROM user_login_bonuses WHERE user_id=? AND login_bonus_id=?';
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->bindValue(2, $bonus->id, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
            if ($row === false) {
                $initBonus = true;
                $ubID = $this->generateID();
                $userBonus = new UserLoginBonus( // ボーナス初期化
                    id: $ubID,
                    userID: $userID,
                    loginBonusID: $bonus->id,
                    lastRewardSequence: 0,
                    loopCount: 1,
                    createdAt: $requestAt,
                    updatedAt: $requestAt,
                );
            } else {
                $userBonus = UserLoginBonus::fromDBRow($row);
            }

            // ボーナス進捗更新
            if ($userBonus->lastRewardSequence < $bonus->columnCount) {
                $userBonus->lastRewardSequence++;
            } else {
                if ($bonus->looped) {
                    $userBonus->loopCount += 1;
                    $userBonus->lastRewardSequence = 1;
                } else {
                    // 上限まで付与完了
                    continue;
                }
            }
            $userBonus->updatedAt = $requestAt;

            // 今回付与するリソース取得
            $query = 'SELECT * FROM login_bonus_reward_masters WHERE login_bonus_id=? AND reward_sequence=?';
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $bonus->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $userBonus->lastRewardSequence, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
            if ($row === false) {
                throw new RuntimeException($this->errLoginBonusRewardNotFound);
            }
            $rewardItem = LoginBonusRewardMaster::fromDBRow($row);

            $this->obtainItem($userID, $rewardItem->itemID, $rewardItem->itemType, $rewardItem->amount, $requestAt);

            // 進捗の保存
            if ($initBonus) {
                $query = 'INSERT INTO user_login_bonuses(id, user_id, login_bonus_id, last_reward_sequence, loop_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)';
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $userBonus->id, PDO::PARAM_INT);
                $stmt->bindValue(2, $userBonus->userID, PDO::PARAM_INT);
                $stmt->bindValue(3, $userBonus->loginBonusID, PDO::PARAM_INT);
                $stmt->bindValue(4, $userBonus->lastRewardSequence, PDO::PARAM_INT);
                $stmt->bindValue(5, $userBonus->loopCount, PDO::PARAM_INT);
                $stmt->bindValue(6, $userBonus->createdAt, PDO::PARAM_INT);
                $stmt->bindValue(7, $userBonus->updatedAt, PDO::PARAM_INT);
                $stmt->execute();
            } else {
                $query = 'UPDATE user_login_bonuses SET last_reward_sequence=?, loop_count=?, updated_at=? WHERE id=?';
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $userBonus->lastRewardSequence, PDO::PARAM_INT);
                $stmt->bindValue(2, $userBonus->loopCount, PDO::PARAM_INT);
                $stmt->bindValue(3, $userBonus->updatedAt, PDO::PARAM_INT);
                $stmt->bindValue(4, $userBonus->id, PDO::PARAM_INT);
                $stmt->execute();
            }

            $sendLoginBonuses[] = $userBonus;
        }

        return $sendLoginBonuses;
    }

    /**
     * obtainPresent プレゼント付与処理
     *
     * @return list<UserPresent>
     * @throws PDOException
     * @throws RuntimeException
     */
    private function obtainPresent(int $userID, int $requestAt): array
    {
        /** @var list<PresentAllMaster> $normalPresents */
        $normalPresents = [];
        $query = 'SELECT * FROM present_all_masters WHERE registered_start_at <= ? AND registered_end_at >= ?';
        $stmt = $this->db->prepare($query);
        $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
        $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
        $stmt->execute();
        while ($row = $stmt->fetch()) {
            $normalPresents[] = PresentAllMaster::fromDBRow($row);
        }

        // 全員プレゼント取得情報更新
        /** @var list<UserPresent> $obtainPresents */
        $obtainPresents = [];
        foreach ($normalPresents as $np) {
            $query = 'SELECT * FROM user_present_all_received_history WHERE user_id=? AND present_all_id=?';
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->bindValue(2, $np->id, PDO::PARAM_INT);
            $stmt->execute();
            if ($stmt->fetch() !== false) {
                // プレゼント配布済み
                continue;
            }

            // user present boxに入れる
            $pID = $this->generateID();
            $up = new UserPresent(
                id: $pID,
                userID: $userID,
                sentAt: $requestAt,
                itemType: $np->itemType,
                itemID: $np->itemID,
                amount: $np->amount,
                presentMessage: $np->presentMessage,
                createdAt: $requestAt,
                updatedAt: $requestAt,
            );
            $query = 'INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)';
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $up->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $up->userID, PDO::PARAM_INT);
            $stmt->bindValue(3, $up->sentAt, PDO::PARAM_INT);
            $stmt->bindValue(4, $up->itemType, PDO::PARAM_INT);
            $stmt->bindValue(5, $up->itemID, PDO::PARAM_INT);
            $stmt->bindValue(6, $up->amount, PDO::PARAM_INT);
            $stmt->bindValue(7, $up->presentMessage);
            $stmt->bindValue(8, $up->createdAt, PDO::PARAM_INT);
            $stmt->bindValue(9, $up->updatedAt, PDO::PARAM_INT);
            $stmt->execute();

            // historyに入れる
            $phID = $this->generateID();
            $history = new UserPresentAllReceivedHistory(
                id: $phID,
                userID: $userID,
                presentAllID: $np->id,
                receivedAt: $requestAt,
                createdAt: $requestAt,
                updatedAt: $requestAt,
            );
            $query = 'INSERT INTO user_present_all_received_history(id, user_id, present_all_id, received_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)';
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $history->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $history->userID, PDO::PARAM_INT);
            $stmt->bindValue(3, $history->presentAllID, PDO::PARAM_INT);
            $stmt->bindValue(4, $history->receivedAt, PDO::PARAM_INT);
            $stmt->bindValue(5, $history->createdAt, PDO::PARAM_INT);
            $stmt->bindValue(6, $history->updatedAt, PDO::PARAM_INT);
            $stmt->execute();

            $obtainPresents[] = $up;
        }

        return $obtainPresents;
    }

    /**
     * obtainItem アイテム付与処理
     *
     * @return array{list<int>, list<UserCard>, list<UserItem>}
     * @throws PDOException
     * @throws RuntimeException
     */
    private function obtainItem(int $userID, int $itemID, int $itemType, int $obtainAmount, int $requestAt): array
    {
        /** @var list<int> $obtainCoins */
        $obtainCoins = [];
        /** @var list<UserCard> $obtainCards */
        $obtainCards = [];
        /** @var list<UserItem> $obtainItems */
        $obtainItems = [];

        switch ($itemType) {
            case 1: // coin
                $query = 'SELECT * FROM users WHERE id=?';
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $userID, PDO::PARAM_INT);
                $stmt->execute();
                $row = $stmt->fetch();
                if ($row === false) {
                    throw new RuntimeException($this->errUserNotFound);
                }
                $user = User::fromDBRow($row);

                $query = 'UPDATE users SET isu_coin=? WHERE id=?';
                $totalCoin = $user->isuCoin + $obtainAmount;
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $totalCoin, PDO::PARAM_INT);
                $stmt->bindValue(2, $userID, PDO::PARAM_INT);
                $stmt->execute();
                $obtainCoins[] = $obtainAmount;
                break;

            case 2: // card(ハンマー)
                $query = 'SELECT * FROM item_masters WHERE id=? AND item_type=?';
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $itemID, PDO::PARAM_INT);
                $stmt->bindValue(2, $itemType, PDO::PARAM_INT);
                $stmt->execute();
                $row = $stmt->fetch();
                if ($row === false) {
                    throw new RuntimeException($this->errItemNotFound);
                }
                $item = ItemMaster::fromDBRow($row);
                $cID = $this->generateID();
                $card = new UserCard(
                    id: $cID,
                    userID: $userID,
                    cardID: $item->id,
                    amountPerSec: $item->amountPerSec,
                    level: 1,
                    totalExp: 0,
                    createdAt: $requestAt,
                    updatedAt: $requestAt,
                );
                $query = 'INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)';
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $card->id, PDO::PARAM_INT);
                $stmt->bindValue(2, $card->userID, PDO::PARAM_INT);
                $stmt->bindValue(3, $card->cardID, PDO::PARAM_INT);
                $stmt->bindValue(4, $card->amountPerSec, PDO::PARAM_INT);
                $stmt->bindValue(5, $card->level, PDO::PARAM_INT);
                $stmt->bindValue(6, $card->totalExp, PDO::PARAM_INT);
                $stmt->bindValue(7, $card->createdAt, PDO::PARAM_INT);
                $stmt->bindValue(8, $card->updatedAt, PDO::PARAM_INT);
                $stmt->execute();
                $obtainCards[] = $card;
                break;

            case 3:
            case 4: // 強化素材
                $query = 'SELECT * FROM item_masters WHERE id=? AND item_type=?';
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $itemID, PDO::PARAM_INT);
                $stmt->bindValue(2, $itemType, PDO::PARAM_INT);
                $stmt->execute();
                $row = $stmt->fetch();
                if ($row === false) {
                    throw new RuntimeException($this->errItemNotFound);
                }
                $item = ItemMaster::fromDBRow($row);
                // 所持数取得
                $query = 'SELECT * FROM user_items WHERE user_id=? AND item_id=?';
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $userID, PDO::PARAM_INT);
                $stmt->bindValue(2, $item->id, PDO::PARAM_INT);
                $stmt->execute();
                $row = $stmt->fetch();
                if ($row === false) { // 新規作成
                    $uitemID = $this->generateID();
                    $uitem = new UserItem(
                        id: $uitemID,
                        userID: $userID,
                        itemType: $item->itemType,
                        itemID: $item->id,
                        amount: $obtainAmount,
                        createdAt: $requestAt,
                        updatedAt: $requestAt,
                    );
                    $query = 'INSERT INTO user_items(id, user_id, item_id, item_type, amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)';
                    $stmt = $this->db->prepare($query);
                    $stmt->bindValue(1, $uitem->id, PDO::PARAM_INT);
                    $stmt->bindValue(2, $userID, PDO::PARAM_INT);
                    $stmt->bindValue(3, $uitem->itemID, PDO::PARAM_INT);
                    $stmt->bindValue(4, $uitem->itemType, PDO::PARAM_INT);
                    $stmt->bindValue(5, $uitem->amount, PDO::PARAM_INT);
                    $stmt->bindValue(6, $requestAt, PDO::PARAM_INT);
                    $stmt->bindValue(7, $requestAt, PDO::PARAM_INT);
                    $stmt->execute();
                } else { // 更新
                    $uitem = UserItem::fromDBRow($row);
                    $uitem->amount += $obtainAmount;
                    $uitem->updatedAt = $requestAt;
                    $query = 'UPDATE user_items SET amount=?, updated_at=? WHERE id=?';
                    $stmt = $this->db->prepare($query);
                    $stmt->bindValue(1, $uitem->amount, PDO::PARAM_INT);
                    $stmt->bindValue(2, $uitem->updatedAt, PDO::PARAM_INT);
                    $stmt->bindValue(3, $uitem->id, PDO::PARAM_INT);
                    $stmt->execute();
                }

                $obtainItems[] = $uitem;
                break;

            default:
                throw new RuntimeException($this->errInvalidItemType);
        }

        return [$obtainCoins, $obtainCards, $obtainItems];
    }

    /**
     * initialize 初期化処理
     * POST /initialize
     */
    public function initialize(Request $request, Response $response): Response
    {
        $fp = fopen('php://temp', 'w+');
        $descriptorSpec = [
            1 => $fp,
            2 => $fp,
        ];

        $process = proc_open(['/bin/sh', '-c', self::SQL_DIRECTORY . 'init.sh'], $descriptorSpec, $_);
        if ($process === false) {
            throw new HttpInternalServerErrorException($request, 'Failed to initialize: cannot open process');
        }

        if (proc_close($process) !== 0) {
            rewind($fp);
            $out = stream_get_contents($fp);
            throw new HttpInternalServerErrorException($request, sprintf('Failed to initialize: %s', $out));
        }

        return $this->successResponse($response, new InitializeResponse(
            language: 'php',
        ));
    }

    /**
     * createUser ユーザの作成
     * POST /user
     */
    public function createUser(Request $request, Response $response): Response
    {
        // parse body
        try {
            $req = new CreateUserRequest($request->getBody()->getContents());
        } catch (Exception $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        if ($req->viewerID === '' || $req->platformType < 1 || $req->platformType > 3) {
            throw new HttpBadRequestException($request, $this->errInvalidRequestBody);
        }

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $this->db->beginTransaction();

        // ユーザ作成
        try {
            $uID = $this->generateID();
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        $user = new User(
            id: $uID,
            isuCoin: 0,
            lastGetRewardAt: $requestAt,
            lastActivatedAt: $requestAt,
            registeredAt: $requestAt,
            createdAt: $requestAt,
            updatedAt: $requestAt,
        );
        $query = 'INSERT INTO users(id, last_activated_at, registered_at, last_getreward_at, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?)';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $user->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $user->lastActivatedAt, PDO::PARAM_INT);
            $stmt->bindValue(3, $user->registeredAt, PDO::PARAM_INT);
            $stmt->bindValue(4, $user->lastGetRewardAt, PDO::PARAM_INT);
            $stmt->bindValue(5, $user->createdAt, PDO::PARAM_INT);
            $stmt->bindValue(6, $user->updatedAt, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        try {
            $udID = $this->generateID();
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        $userDevice = new UserDevice(
            id: $udID,
            userID: $user->id,
            platformID: $req->viewerID,
            platformType: $req->platformType,
            createdAt: $requestAt,
            updatedAt: $requestAt,
        );
        $query = 'INSERT INTO user_devices(id, user_id, platform_id, platform_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userDevice->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $user->id, PDO::PARAM_INT);
            $stmt->bindValue(3, $req->viewerID);
            $stmt->bindValue(4, $req->platformType, PDO::PARAM_INT);
            $stmt->bindValue(5, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(6, $requestAt, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        // 初期デッキ付与
        $query = 'SELECT * FROM item_masters WHERE id=?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, 2, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request, $this->errItemNotFound);
        }
        $initCard = ItemMaster::fromDBRow($row);

        /** @var list<UserCard> $initCards */
        $initCards = [];
        for ($i = 0; $i < 3; $i++) {
            try {
                $cID = $this->generateID();
            } catch (Exception $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
            $card = new UserCard(
                id: $cID,
                userID: $user->id,
                cardID: $initCard->id,
                amountPerSec: $initCard->amountPerSec,
                level: 1,
                totalExp: 0,
                createdAt: $requestAt,
                updatedAt: $requestAt,
            );
            $query = 'INSERT INTO user_cards(id, user_id, card_id, amount_per_sec, level, total_exp, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)';
            try {
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $card->id, PDO::PARAM_INT);
                $stmt->bindValue(2, $card->userID, PDO::PARAM_INT);
                $stmt->bindValue(3, $card->cardID, PDO::PARAM_INT);
                $stmt->bindValue(4, $card->amountPerSec, PDO::PARAM_INT);
                $stmt->bindValue(5, $card->level, PDO::PARAM_INT);
                $stmt->bindValue(6, $card->totalExp, PDO::PARAM_INT);
                $stmt->bindValue(7, $card->createdAt, PDO::PARAM_INT);
                $stmt->bindValue(8, $card->updatedAt, PDO::PARAM_INT);
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
            $initCards[] = $card;
        }

        try {
            $deckID = $this->generateID();
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        $initDeck = new UserDeck(
            id: $deckID,
            userID: $user->id,
            cardID1: $initCards[0]->id,
            cardID2: $initCards[1]->id,
            cardID3: $initCards[2]->id,
            createdAt: $requestAt,
            updatedAt: $requestAt,
        );
        $query = 'INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $initDeck->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $initDeck->userID, PDO::PARAM_INT);
            $stmt->bindValue(3, $initDeck->cardID1, PDO::PARAM_INT);
            $stmt->bindValue(4, $initDeck->cardID2, PDO::PARAM_INT);
            $stmt->bindValue(5, $initDeck->cardID3, PDO::PARAM_INT);
            $stmt->bindValue(6, $initDeck->createdAt, PDO::PARAM_INT);
            $stmt->bindValue(7, $initDeck->updatedAt, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        // ログイン処理
        try {
            [$user, $loginBonuses, $presents] = $this->loginProcess($user->id, $requestAt);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errUserNotFound || $err === $this->errItemNotFound || $err === $this->errLoginBonusRewardNotFound) {
                throw new HttpNotFoundException($request, $err, $e);
            } elseif ($err === $this->errInvalidItemType) {
                throw new HttpBadRequestException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        // generate session
        try {
            $sID = $this->generateID();
            $sessID = $this->generateUUID();
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        $sess = new Session(
            id: $sID,
            userID: $user->id,
            sessionID: $sessID,
            createdAt: $requestAt,
            updatedAt: $requestAt,
            expiredAt: $requestAt + 86400,
        );
        $query = 'INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)';
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

        return $this->successResponse($response, new CreateUserResponse(
            userID: $user->id,
            viewerID: $req->viewerID,
            sessionID: $sess->sessionID,
            createdAt: $requestAt,
            updatedResources: new UpdatedResource($requestAt, $user, $userDevice, $initCards, [$initDeck], null, $loginBonuses, $presents),
        ));
    }

    /**
     * login ログイン
     * POST /login
     */
    public function login(Request $request, Response $response): Response
    {
        try {
            $req = new LoginRequest($request->getBody()->getContents());
        } catch (Exception $e) {
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
            $stmt->bindValue(1, $req->userID, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request, $this->errUserNotFound);
        }
        $user = User::fromDBRow($row);

        // check ban
        try {
            $isBan = $this->checkBan($user->id);
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($isBan) {
            throw new HttpForbiddenException($request, $this->errForbidden);
        }

        // viewer id check
        try {
            $this->checkViewerID($user->id, $req->viewerID);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errUserDeviceNotFound) {
                throw new HttpNotFoundException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        $this->db->beginTransaction();

        // sessionを更新
        $query = 'UPDATE user_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(2, $req->userID, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
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
        $query = 'INSERT INTO user_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)';
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

        // すでにログインしているユーザはログイン処理をしない
        if ($this->isCompleteTodayLogin($user->lastActivatedAt, $requestAt)) {
            $user->updatedAt = $requestAt;
            $user->lastActivatedAt = $requestAt;

            $query = 'UPDATE users SET updated_at=?, last_activated_at=? WHERE id=?';
            try {
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
                $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
                $stmt->bindValue(3, $req->userID, PDO::PARAM_INT);
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }

            $this->db->commit();

            return $this->successResponse($response, new LoginResponse(
                viewerID: $req->viewerID,
                sessionID: $sess->sessionID,
                updatedResources: new UpdatedResource($requestAt, $user, null, null, null, null, null, null),
            ));
        }

        // login process
        try {
            [$user, $loginBonuses, $presents] = $this->loginProcess($req->userID, $requestAt);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errUserNotFound || $err === $this->errItemNotFound || $err === $this->errLoginBonusRewardNotFound) {
                throw new HttpNotFoundException($request, $err, $e);
            } elseif ($err === $this->errInvalidItemType) {
                throw new HttpBadRequestException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        $this->db->commit();

        return $this->successResponse($response, new LoginResponse(
            viewerID: $req->viewerID,
            sessionID: $sess->sessionID,
            updatedResources: new UpdatedResource($requestAt, $user, null, null, null, null, $loginBonuses, $presents),
        ));
    }

    /**
     * listGacha ガチャ一覧
     * GET /user/{userID}/gacha/index
     */
    public function listGacha(Request $request, Response $response): Response
    {
        try {
            $userID = $this->getUserID($request);
        } catch (RuntimeException $e) {
            throw new HttpBadRequestException($request, 'invalid userID parameter', $e);
        }

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        /** @var list<GachaMaster> $gachaMasterList */
        $gachaMasterList = [];
        $query = 'SELECT * FROM gacha_masters WHERE start_at <= ? AND end_at >= ? ORDER BY display_order ASC';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $gachaMasterList[] = GachaMaster::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if (count($gachaMasterList) === 0) {
            return $this->successResponse($response, new ListGachaResponse( // 0 件
                oneTimeToken: '',
                gachas: [],
            ));
        }

        // ガチャ排出アイテム取得
        /** @var list<GachaData> $gachaDataList */
        $gachaDataList = [];
        $query = 'SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC';
        try {
            $stmt = $this->db->prepare($query);
            foreach ($gachaMasterList as $v) {
                /** @var list<GachaItemMaster> $gachaItem */
                $gachaItem = [];
                $stmt->bindValue(1, $v->id, PDO::PARAM_INT);
                $stmt->execute();
                while ($row = $stmt->fetch()) {
                    $gachaItem[] = GachaItemMaster::fromDBRow($row);
                }
                if (count($gachaItem) === 0) {
                    throw new HttpNotFoundException($request, 'not found gacha item');
                }
                $gachaDataList[] = new GachaData(
                    gacha: $v,
                    gachaItem: $gachaItem,
                );
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        // generate one time token
        $query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(2, $userID, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        try {
            $tID = $this->generateID();
            $tk = $this->generateUUID();
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        $token = new UserOneTimeToken(
            id: $tID,
            userID: $userID,
            token: $tk,
            tokenType: 1,
            createdAt: $requestAt,
            updatedAt: $requestAt,
            expiredAt: $requestAt + 600,
        );
        $query = 'INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?, ?)';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $token->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $token->userID, PDO::PARAM_INT);
            $stmt->bindValue(3, $token->token);
            $stmt->bindValue(4, $token->tokenType, PDO::PARAM_INT);
            $stmt->bindValue(5, $token->createdAt, PDO::PARAM_INT);
            $stmt->bindValue(6, $token->updatedAt, PDO::PARAM_INT);
            $stmt->bindValue(7, $token->expiredAt, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        return $this->successResponse($response, new ListGachaResponse(
            oneTimeToken: $token->token,
            gachas: $gachaDataList,
        ));
    }

    /**
     * drawGacha ガチャを引く
     * POST /user/{userID}/gacha/draw/{gachaTypeID}/{n}
     */
    public function drawGacha(Request $request, Response $response, array $params): Response
    {
        try {
            $userID = $this->getUserID($request);
        } catch (RuntimeException $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        $gachaIDStr = $params['gachaID'] ?? '';
        $gachaID = filter_var($gachaIDStr, FILTER_VALIDATE_INT);
        if (!is_int($gachaID)) {
            throw new HttpBadRequestException($request, 'invalid gachaID');
        }

        $gachaCountStr = $params['n'] ?? '';
        $gachaCount = filter_var($gachaCountStr, FILTER_VALIDATE_INT);
        if (!is_int($gachaCount)) {
            throw new HttpBadRequestException($request);
        }
        if ($gachaCount !== 1 && $gachaCount !== 10) {
            throw new HttpBadRequestException($request, 'invalid draw gacha times');
        }

        try {
            $req = new DrawGachaRequest($request->getBody()->getContents());
        } catch (Exception $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        try {
            $this->checkOneTimeToken($req->oneTimeToken, 1, $requestAt);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errInvalidToken) {
                throw new HttpBadRequestException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        try {
            $this->checkViewerID($userID, $req->viewerID);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errUserDeviceNotFound) {
                throw new HttpNotFoundException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        $consumedCoin = $gachaCount * 1000;

        // userのisuconが足りるか
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
        if ($user->isuCoin < $consumedCoin) {
            throw new HttpException($request, 'not enough isucoin', StatusCodeInterface::STATUS_CONFLICT);
        }

        // gachaIDからガチャマスタの取得
        $query = 'SELECT * FROM gacha_masters WHERE id=? AND start_at <= ? AND end_at >= ?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $gachaID, PDO::PARAM_INT);
            $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(3, $requestAt, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request, 'not found gacha');
        }
        $gachaInfo = GachaMaster::fromDBRow($row);

        // gachaItemMasterからアイテムリスト取得
        /** @var list<GachaItemMaster> $gachaItemList */
        $gachaItemList = [];
        try {
            $stmt = $this->db->prepare('SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id ASC');
            $stmt->bindValue(1, $gachaID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $gachaItemList[] = GachaItemMaster::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if (count($gachaItemList) === 0) {
            throw new HttpNotFoundException($request, 'not found gacha item');
        }

        // weightの合計値を算出
        $sum = 0;
        try {
            $stmt = $this->db->prepare('SELECT SUM(weight) FROM gacha_item_masters WHERE gacha_id=?');
            $stmt->bindValue(1, $gachaID, PDO::PARAM_INT);
            $stmt->execute();
            $sum = $stmt->fetchColumn();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($sum === false) {
            throw new HttpNotFoundException($request);
        }

        // random値の導出 & 抽選
        /** @var list<GachaItemMaster> $result */
        $result = [];
        for ($i = 0; $i < $gachaCount; $i++) {
            $random = random_int(0, (int)$sum);
            $boundary = 0;
            foreach ($gachaItemList as $v) {
                $boundary += $v->weight;
                if ($random < $boundary) {
                    $result[] = $v;
                    break;
                }
            }
        }

        $this->db->beginTransaction();

        // 直付与 => プレゼントに入れる
        /** @var list<UserPresent> $presents */
        $presents = [];
        foreach ($result as $v) {
            try {
                $pID = $this->generateID();
            } catch (Exception $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
            $present = new UserPresent(
                id: $pID,
                userID: $userID,
                sentAt: $requestAt,
                itemType: $v->itemType,
                itemID: $v->itemID,
                amount: $v->amount,
                presentMessage: sprintf('%sの付与アイテムです', $gachaInfo->name),
                createdAt: $requestAt,
                updatedAt: $requestAt,
            );
            $query = 'INSERT INTO user_presents(id, user_id, sent_at, item_type, item_id, amount, present_message, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)';
            try {
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $present->id, PDO::PARAM_INT);
                $stmt->bindValue(2, $present->userID, PDO::PARAM_INT);
                $stmt->bindValue(3, $present->sentAt, PDO::PARAM_INT);
                $stmt->bindValue(4, $present->itemType, PDO::PARAM_INT);
                $stmt->bindValue(5, $present->itemID, PDO::PARAM_INT);
                $stmt->bindValue(6, $present->amount, PDO::PARAM_INT);
                $stmt->bindValue(7, $present->presentMessage);
                $stmt->bindValue(8, $present->createdAt, PDO::PARAM_INT);
                $stmt->bindValue(9, $present->updatedAt, PDO::PARAM_INT);
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }

            $presents[] = $present;
        }

        // isuconをへらす
        $query = 'UPDATE users SET isu_coin=? WHERE id=?';
        $totalCoin = $user->isuCoin - $consumedCoin;
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $totalCoin, PDO::PARAM_INT);
            $stmt->bindValue(2, $userID, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $this->db->commit();

        return $this->successResponse($response, new DrawGachaResponse(
            presents: $presents,
        ));
    }

    /**
     * listPresent プレゼント一覧
     * GET /user/{userID}/present/index/{n}
     */
    public function listPresent(Request $request, Response $response, array $params): Response
    {
        $nStr = $params['n'] ?? '';
        $n = filter_var($nStr, FILTER_VALIDATE_INT);
        if (!is_int($n)) {
            throw new HttpBadRequestException($request, 'invalid n parameter');
        }
        if ($n === 0) {
            throw new HttpBadRequestException($request, 'index number is more than 1');
        }

        try {
            $userID = $this->getUserID($request);
        } catch (RuntimeException $e) {
            throw new HttpBadRequestException($request, 'invalid userID parameter', $e);
        }

        $offset = self::PRESENT_COUNT_PER_PAGE * ($n - 1);
        /** @var list<UserPresent> $presentList */
        $presentList = [];
        $query = <<<'SQL'
            SELECT * FROM user_presents
            WHERE user_id = ? AND deleted_at IS NULL
            ORDER BY created_at DESC, id
            LIMIT ? OFFSET ?
        SQL;
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->bindValue(2, self::PRESENT_COUNT_PER_PAGE, PDO::PARAM_INT);
            $stmt->bindValue(3, $offset, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $presentList[] = UserPresent::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        try {
            $stmt = $this->db->prepare('SELECT COUNT(*) FROM user_presents WHERE user_id = ? AND deleted_at IS NULL');
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            $presentCount = $stmt->fetchColumn();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $isNext = false;
        if ($presentCount > ($offset + self::PRESENT_COUNT_PER_PAGE)) {
            $isNext = true;
        }

        return $this->successResponse($response, new ListPresentResponse(
            presents: $presentList,
            isNext: $isNext,
        ));
    }

    /**
     * receivePresent プレゼント受け取り
     * POST /user/{userID}/present/receive
     */
    public function receivePresent(Request $request, Response $response): Response
    {
        // read body
        try {
            $req = new ReceivePresentRequest($request->getBody()->getContents());
        } catch (Exception $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

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

        if (count($req->presentIDs) === 0) {
            throw new HttpException($request, 'presentIds is empty', StatusCodeInterface::STATUS_UNPROCESSABLE_ENTITY);
        }

        try {
            $this->checkViewerID($userID, $req->viewerID);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errUserDeviceNotFound) {
                throw new HttpNotFoundException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        // user_presentsに入っているが未取得のプレゼント取得
        $inClause = str_repeat('?, ', count($req->presentIDs) - 1) . '?';
        $query = 'SELECT * FROM user_presents WHERE id IN (' . $inClause . ') AND deleted_at IS NULL';
        /** @var list<UserPresent> $obtainPresent */
        $obtainPresent = [];
        try {
            $stmt = $this->db->prepare($query);
            foreach ($req->presentIDs as $i => $presentID) {
                $stmt->bindValue($i + 1, $presentID, PDO::PARAM_INT);
            }
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $obtainPresent[] = UserPresent::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if (count($obtainPresent) === 0) {
            return $this->successResponse($response, new ReceivePresentResponse(
                updatedResources: new UpdatedResource($requestAt, null, null, null, null, null, null, []),
            ));
        }

        $this->db->beginTransaction();

        // 配布処理
        for ($i = 0; $i < count($obtainPresent); $i++) {
            if (!is_null($obtainPresent[$i]->deletedAt)) {
                throw new HttpInternalServerErrorException($request, 'received present');
            }

            $obtainPresent[$i]->updatedAt = $requestAt;
            $obtainPresent[$i]->deletedAt = $requestAt;
            $v = $obtainPresent[$i];
            $query = 'UPDATE user_presents SET deleted_at=?, updated_at=? WHERE id=?';
            try {
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
                $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
                $stmt->bindValue(3, $v->id, PDO::PARAM_INT);
                $stmt->execute();
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }

            try {
                $this->obtainItem($v->userID, $v->itemID, $v->itemType, $v->amount, $requestAt);
            } catch (Exception $e) {
                $err = $e->getMessage();
                if ($err === $this->errUserNotFound || $err === $this->errItemNotFound) {
                    throw new HttpNotFoundException($request, $err, $e);
                } elseif ($err === $this->errInvalidItemType) {
                    throw new HttpBadRequestException($request, $err, $e);
                }
                throw new HttpInternalServerErrorException($request, $err, $e);
            }
        }

        $this->db->commit();

        return $this->successResponse($response, new ReceivePresentResponse(
            updatedResources: new UpdatedResource($requestAt, null, null, null, null, null, null, $obtainPresent),
        ));
    }

    /**
     * listItem アイテムリスト
     * GET /user/{userID}/item
     */
    public function listItem(Request $request, Response $response): Response
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

        /** @var list<UserItem> $itemList */
        $itemList = [];
        $query = 'SELECT * FROM user_items WHERE user_id = ?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $itemList[] = UserItem::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        /** @var list<UserCard> $cardList */
        $cardList = [];
        $query = 'SELECT * FROM user_cards WHERE user_id=?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $cardList[] = UserCard::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        // generate one time token
        $query = 'UPDATE user_one_time_tokens SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(2, $userID, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        try {
            $tID = $this->generateID();
            $tk = $this->generateUUID();
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        $token = new UserOneTimeToken(
            id: $tID,
            userID: $userID,
            token: $tk,
            tokenType: 2,
            createdAt: $requestAt,
            updatedAt: $requestAt,
            expiredAt: $requestAt + 600,
        );
        $query = 'INSERT INTO user_one_time_tokens(id, user_id, token, token_type, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?, ?)';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $token->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $token->userID, PDO::PARAM_INT);
            $stmt->bindValue(3, $token->token);
            $stmt->bindValue(4, $token->tokenType, PDO::PARAM_INT);
            $stmt->bindValue(5, $token->createdAt, PDO::PARAM_INT);
            $stmt->bindValue(6, $token->updatedAt, PDO::PARAM_INT);
            $stmt->bindValue(7, $token->expiredAt, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        return $this->successResponse($response, new ListItemResponse(
            oneTimeToken: $token->token,
            items: $itemList,
            user: $user,
            cards: $cardList,
        ));
    }

    /**
     * addExpToCard 装備強化
     * POST /user/{userID}/card/addexp/{cardID}
     */
    public function addExpToCard(Request $request, Response $response, array $params): Response
    {
        $cardIDStr = $params['cardID'] ?? '';
        $cardID = filter_var($cardIDStr, FILTER_VALIDATE_INT);
        if (!is_int($cardID)) {
            throw new HttpBadRequestException($request);
        }

        try {
            $userID = $this->getUserID($request);
        } catch (RuntimeException $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        // read body
        try {
            $req = new AddExpToCardRequest($request->getBody()->getContents());
        } catch (Exception $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        try {
            $this->checkOneTimeToken($req->oneTimeToken, 2, $requestAt);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errInvalidToken) {
                throw new HttpBadRequestException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        try {
            $this->checkViewerID($userID, $req->viewerID);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errUserDeviceNotFound) {
                throw new HttpNotFoundException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        // get target card
        $query = <<<'SQL'
            SELECT uc.id , uc.user_id , uc.card_id , uc.amount_per_sec , uc.level, uc.total_exp, im.amount_per_sec as 'base_amount_per_sec', im.max_level , im.max_amount_per_sec , im.base_exp_per_level
            FROM user_cards as uc
            INNER JOIN item_masters as im ON uc.card_id = im.id
            WHERE uc.id = ? AND uc.user_id=?
        SQL;
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $cardID, PDO::PARAM_INT);
            $stmt->bindValue(2, $userID, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request);
        }
        $card = TargetUserCardData::fromDBRow($row);

        if ($card->level === $card->maxLevel) {
            throw new HttpBadRequestException($request, 'target card is max level');
        }

        // 消費アイテムの所持チェック
        /** @var list<ConsumeUserItemData> $items */
        $items = [];
        $query = <<<'SQL'
            SELECT ui.id, ui.user_id, ui.item_id, ui.item_type, ui.amount, ui.created_at, ui.updated_at, im.gained_exp
            FROM user_items as ui
            INNER JOIN item_masters as im ON ui.item_id = im.id
            WHERE ui.item_type = 3 AND ui.id=? AND ui.user_id=?
        SQL;
        try {
            $stmt = $this->db->prepare($query);
            foreach ($req->items as $v) {
                $stmt->bindValue(1, $v->id, PDO::PARAM_INT);
                $stmt->bindValue(2, $userID, PDO::PARAM_INT);
                $stmt->execute();
                $row = $stmt->fetch();
                if ($row === false) {
                    throw new HttpNotFoundException($request);
                }
                $item = ConsumeUserItemData::fromDBRow($row);

                if ($v->amount > $item->amount) {
                    throw new HttpBadRequestException($request, 'item not enough');
                }
                $item->consumeAmount = $v->amount;
                $items[] = $item;
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        // 経験値付与
        // 経験値をカードに付与
        foreach ($items as $v) {
            $card->totalExp += $v->gainedExp * $v->consumeAmount;
        }

        // lvup判定(lv upしたら生産性を加算)
        while (true) {
            $nextLvThreshold = $card->baseExpPerLevel * pow(1.2, $card->level - 1);
            if ($nextLvThreshold > $card->totalExp) {
                break;
            }

            // lv up処理
            $card->level += 1;
            $card->amountPerSec += ($card->maxAmountPerSec - $card->baseAmountPerSec) / ($card->maxLevel - 1);
        }

        $this->db->beginTransaction();

        // cardのlvと経験値の更新、itemの消費
        $query = 'UPDATE user_cards SET amount_per_sec=?, level=?, total_exp=?, updated_at=? WHERE id=?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $card->amountPerSec, PDO::PARAM_INT);
            $stmt->bindValue(2, $card->level, PDO::PARAM_INT);
            $stmt->bindValue(3, $card->totalExp, PDO::PARAM_INT);
            $stmt->bindValue(4, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(5, $card->id, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $query = 'UPDATE user_items SET amount=?, updated_at=? WHERE id=?';
        try {
            $stmt = $this->db->prepare($query);
            foreach ($items as $v) {
                $stmt->bindValue(1, $v->amount - $v->consumeAmount, PDO::PARAM_INT);
                $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
                $stmt->bindValue(3, $v->id, PDO::PARAM_INT);
                $stmt->execute();
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        // get response data
        $query = 'SELECT * FROM user_cards WHERE id=?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $cardID, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request, 'not found card');
        }
        $resultCard = UserCard::fromDBRow($row);
        /** @var list<UserItem> $resultItems */
        $resultItems = [];
        foreach ($items as $v) {
            $resultItems[] = new UserItem(
                id: $v->id,
                userID: $v->userID,
                itemID: $v->itemID,
                itemType: $v->itemType,
                amount: $v->amount - $v->consumeAmount,
                createdAt: $v->createdAt,
                updatedAt: $requestAt,
            );
        }

        $this->db->commit();

        return $this->successResponse($response, new AddExpToCardResponse(
            updatedResources: new UpdatedResource($requestAt, null, null, [$resultCard], null, $resultItems, null, null),
        ));
    }

    /**
     * updateDeck 装備変更
     * POST /user/{userID}/card
     */
    public function updateDeck(Request $request, Response $response): Response
    {
        try {
            $userID = $this->getUserID($request);
        } catch (RuntimeException $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        // read body
        try {
            $req = new UpdateDeckRequest($request->getBody()->getContents());
        } catch (Exception $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        if (count($req->cardIDs) !== self::DECK_CARD_NUMBER) {
            throw new HttpBadRequestException($request, 'invalid number of cards');
        }

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        try {
            $this->checkViewerID($userID, $req->viewerID);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errUserDeviceNotFound) {
                throw new HttpNotFoundException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        // カード所持情報のバリデーション
        /** @var list<UserCard> $cards */
        $cards = [];
        $query = 'SELECT * FROM user_cards WHERE id IN (?, ?, ?)';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $req->cardIDs[0], PDO::PARAM_INT);
            $stmt->bindValue(2, $req->cardIDs[1], PDO::PARAM_INT);
            $stmt->bindValue(3, $req->cardIDs[2], PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $cards[] = UserCard::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if (count($cards) !== self::DECK_CARD_NUMBER) {
            throw new HttpBadRequestException($request, 'invalid card ids');
        }

        $this->db->beginTransaction();

        // update data
        $query = 'UPDATE user_decks SET updated_at=?, deleted_at=? WHERE user_id=? AND deleted_at IS NULL';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(2, $requestAt, PDO::PARAM_INT);
            $stmt->bindValue(3, $userID, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        try {
            $udID = $this->generateID();
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        $newDeck = new UserDeck(
            id: $udID,
            userID: $userID,
            cardID1: $req->cardIDs[0],
            cardID2: $req->cardIDs[1],
            cardID3: $req->cardIDs[2],
            createdAt: $requestAt,
            updatedAt: $requestAt,
        );
        $query = 'INSERT INTO user_decks(id, user_id, user_card_id_1, user_card_id_2, user_card_id_3, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $newDeck->id, PDO::PARAM_INT);
            $stmt->bindValue(2, $newDeck->userID, PDO::PARAM_INT);
            $stmt->bindValue(3, $newDeck->cardID1, PDO::PARAM_INT);
            $stmt->bindValue(4, $newDeck->cardID2, PDO::PARAM_INT);
            $stmt->bindValue(5, $newDeck->cardID3, PDO::PARAM_INT);
            $stmt->bindValue(6, $newDeck->createdAt, PDO::PARAM_INT);
            $stmt->bindValue(7, $newDeck->updatedAt, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        $this->db->commit();

        return $this->successResponse($response, new UpdateDeckResponse(
            updatedResources: new UpdatedResource($requestAt, null, null, null, [$newDeck], null, null, null),
        ));
    }

    /**
     * reward ゲーム報酬受取
     * POST /user/{userID}/reward
     */
    public function reward(Request $request, Response $response): Response
    {
        try {
            $userID = $this->getUserID($request);
        } catch (RuntimeException $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        // parse body
        try {
            $req = new RewardRequest($request->getBody()->getContents());
        } catch (Exception $e) {
            throw new HttpBadRequestException($request, $e->getMessage(), $e);
        }

        try {
            $requestAt = $this->getRequestTime($request);
        } catch (Exception $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        try {
            $this->checkViewerID($userID, $req->viewerID);
        } catch (Exception $e) {
            $err = $e->getMessage();
            if ($err === $this->errUserDeviceNotFound) {
                throw new HttpNotFoundException($request, $err, $e);
            }
            throw new HttpInternalServerErrorException($request, $err, $e);
        }

        // 最後に取得した報酬時刻取得
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

        // 使っているデッキの取得
        $query = 'SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row === false) {
            throw new HttpNotFoundException($request);
        }
        $deck = UserDeck::fromDBRow($row);

        /** @var list<UserCard> $cards */
        $cards = [];
        $query = 'SELECT * FROM user_cards WHERE id IN (?, ?, ?)';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $deck->cardID1, PDO::PARAM_INT);
            $stmt->bindValue(2, $deck->cardID2, PDO::PARAM_INT);
            $stmt->bindValue(3, $deck->cardID3, PDO::PARAM_INT);
            $stmt->execute();
            while ($row = $stmt->fetch()) {
                $cards[] = UserCard::fromDBRow($row);
            }
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if (count($cards) !== 3) {
            throw new HttpBadRequestException($request, 'invalid cards length');
        }

        // 経過時間*生産性のcoin (1椅子 = 1coin)
        $pastTime = $requestAt - $user->lastGetRewardAt;
        $getCoin = $pastTime * ($cards[0]->amountPerSec + $cards[1]->amountPerSec + $cards[2]->amountPerSec);

        // 報酬の保存(ゲームない通貨を保存)(users)
        $user->isuCoin += $getCoin;
        $user->lastGetRewardAt = $requestAt;

        $query = 'UPDATE users SET isu_coin=?, last_getreward_at=? WHERE id=?';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $user->isuCoin, PDO::PARAM_INT);
            $stmt->bindValue(2, $user->lastGetRewardAt, PDO::PARAM_INT);
            $stmt->bindValue(3, $user->id, PDO::PARAM_INT);
            $stmt->execute();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }

        return $this->successResponse($response, new RewardResponse(
            updateResources: new UpdatedResource($requestAt, $user, null, null, null, null, null, null),
        ));
    }

    /**
     * home ホーム取得
     * GET /user/{userID}/home
     */
    public function home(Request $request, Response $response): Response
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

        // 装備情報
        $deck = null;
        $query = 'SELECT * FROM user_decks WHERE user_id=? AND deleted_at IS NULL';
        try {
            $stmt = $this->db->prepare($query);
            $stmt->bindValue(1, $userID, PDO::PARAM_INT);
            $stmt->execute();
            $row = $stmt->fetch();
        } catch (PDOException $e) {
            throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
        }
        if ($row !== false) {
            $deck = UserDeck::fromDBRow($row);
        }

        // 生産性
        /** @var list<UserCard> $cards */
        $cards = [];
        if (!is_null($deck)) {
            $query = 'SELECT * FROM user_cards WHERE id IN (?, ?, ?)';
            try {
                $stmt = $this->db->prepare($query);
                $stmt->bindValue(1, $deck->cardID1, PDO::PARAM_INT);
                $stmt->bindValue(2, $deck->cardID2, PDO::PARAM_INT);
                $stmt->bindValue(3, $deck->cardID3, PDO::PARAM_INT);
                $stmt->execute();
                while ($row = $stmt->fetch()) {
                    $cards[] = UserCard::fromDBRow($row);
                }
            } catch (PDOException $e) {
                throw new HttpInternalServerErrorException($request, $e->getMessage(), $e);
            }
        }
        $totalAmountPerSec = 0;
        foreach ($cards as $v) {
            $totalAmountPerSec += $v->amountPerSec;
        }

        // 経過時間
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
        $pastTime = $requestAt - $user->lastGetRewardAt;

        return $this->successResponse($response, new HomeResponse(
            now: $requestAt,
            user: $user,
            deck: $deck,
            totalAmountPerSec: $totalAmountPerSec,
            pastTime: $pastTime,
        ));
    }

    // //////////////////////////////////////
    // util

    /**
     * health ヘルスチェック
     */
    public function health(Request $request, Response $response): Response
    {
        $response->getBody()->write('OK');

        return $response;
    }
}
