<?php

declare(strict_types=1);

namespace App\IsuConquest;

use JsonException;
use JsonSerializable;
use TypeError;

class InitializeResponse implements JsonSerializable
{
    public function __construct(public string $language)
    {
    }

    public function jsonSerialize(): array
    {
        return ['language' => $this->language];
    }
}

class CreateUserRequest
{
    public readonly string $viewerID;
    public readonly int $platformType;

    /**
     * @throws JsonException
     * @throws TypeError
     */
    public function __construct(string $json)
    {
        $data = json_decode($json, flags: JSON_THROW_ON_ERROR);

        $this->viewerID = $data->viewerId;
        $this->platformType = $data->platformType;
    }
}

class CreateUserResponse implements JsonSerializable
{
    public function __construct(
        public int $userID,
        public string $viewerID,
        public string $sessionID,
        public int $createdAt,
        public UpdatedResource $updatedResources,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'userId' => $this->userID,
            'viewerId' => $this->viewerID,
            'sessionId' => $this->sessionID,
            'createdAt' => $this->createdAt,
            'updatedResources' => $this->updatedResources,
        ];
    }
}

class LoginRequest
{
    public readonly string $viewerID;
    public readonly int $userID;

    /**
     * @throws JsonException
     * @throws TypeError
     */
    public function __construct(string $json)
    {
        $data = json_decode($json, flags: JSON_THROW_ON_ERROR);

        $this->viewerID = $data->viewerId;
        $this->userID = $data->userId;
    }
}

class LoginResponse implements JsonSerializable
{
    public function __construct(
        public string $viewerID,
        public string $sessionID,
        public UpdatedResource $updatedResources,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'viewerId' => $this->viewerID,
            'sessionId' => $this->sessionID,
            'updatedResources' => $this->updatedResources,
        ];
    }
}

class ListGachaResponse implements JsonSerializable
{
    /**
     * @param list<GachaData> $gachas
     */
    public function __construct(
        public string $oneTimeToken,
        public array $gachas,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'oneTimeToken' => $this->oneTimeToken,
            'gachas' => $this->gachas,
        ];
    }
}

class GachaData implements JsonSerializable
{
    /**
     * @param list<GachaItemMaster> $gachaItem
     */
    public function __construct(
        public GachaMaster $gacha,
        public array $gachaItem,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'gacha' => $this->gacha,
            'gachaItemList' => $this->gachaItem,
        ];
    }
}

class DrawGachaRequest
{
    public readonly string $viewerID;
    public readonly string $oneTimeToken;

    /**
     * @throws JsonException
     * @throws TypeError
     */
    public function __construct(string $json)
    {
        $data = json_decode($json, flags: JSON_THROW_ON_ERROR);

        $this->viewerID = $data->viewerId;
        $this->oneTimeToken = $data->oneTimeToken;
    }
}

class DrawGachaResponse implements JsonSerializable
{
    /**
     * @param list<UserPresent> $presents
     */
    public function __construct(public array $presents)
    {
    }

    public function jsonSerialize(): array
    {
        return ['presents' => $this->presents];
    }
}

class ListPresentResponse implements JsonSerializable
{
    /**
     * @param list<UserPresent> $presents
     */
    public function __construct(
        public array $presents,
        public bool $isNext
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'presents' => $this->presents,
            'isNext' => $this->isNext,
        ];
    }
}

class ReceivePresentRequest
{
    public readonly string $viewerID;
    /** @var list<int> */
    public readonly array $presentIDs;

    /**
     * @throws JsonException
     * @throws TypeError
     */
    public function __construct(string $json)
    {
        $data = json_decode($json, flags: JSON_THROW_ON_ERROR);

        $this->viewerID = $data->viewerId;
        $this->presentIDs = $data->presentIds;
    }
}

class ReceivePresentResponse implements JsonSerializable
{
    public function __construct(public UpdatedResource $updatedResources)
    {
    }

    public function jsonSerialize(): array
    {
        return ['updatedResources' => $this->updatedResources];
    }
}

class ListItemResponse implements JsonSerializable
{
    /**
     * @param list<UserItem> $items
     * @param list<UserCard> $cards
     */
    public function __construct(
        public string $oneTimeToken,
        public User $user,
        public array $items,
        public array $cards,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'oneTimeToken' => $this->oneTimeToken,
            'user' => $this->user,
            'items' => $this->items,
            'cards' => $this->cards,
        ];
    }
}

class AddExpToCardRequest
{
    public readonly string $viewerID;
    public readonly string $oneTimeToken;
    /** @var list<ConsumeItem> */
    public readonly array $items;

    /**
     * @throws JsonException
     * @throws TypeError
     */
    public function __construct(string $json)
    {
        $data = json_decode($json, flags: JSON_THROW_ON_ERROR);

        $this->viewerID = $data->viewerId;
        $this->oneTimeToken = $data->oneTimeToken;
        $this->items = $data->items;
    }
}

class AddExpToCardResponse implements JsonSerializable
{
    public function __construct(public UpdatedResource $updatedResources)
    {
    }

    public function jsonSerialize(): array
    {
        return ['updatedResources' => $this->updatedResources];
    }
}

class ConsumeItem
{
    public function __construct(
        public readonly int $id,
        public readonly int $amount,
    ) {
    }
}

class ConsumeUserItemData
{
    public function __construct(
        public int $id,
        public int $userID,
        public int $itemID,
        public int $itemType,
        public int $amount,
        public int $createdAt,
        public int $updatedAt,
        public int $gainedExp,
        public int $consumeAmount = 0, // 消費量
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['item_id'],
            $row['item_type'],
            $row['amount'],
            $row['created_at'],
            $row['updated_at'],
            $row['gained_exp'],
        );
    }
}

class TargetUserCardData
{
    public function __construct(
        public int $id,
        public int $userID,
        public int $cardID,
        public int $amountPerSec,
        public int $level,
        public int $totalExp,
        // lv1のときの生産性
        public int $baseAmountPerSec,
        // 最高レベル
        public int $maxLevel,
        // lv maxのときの生産性
        public int $maxAmountPerSec,
        // lv1 -> lv2に上がるときのexp
        public int $baseExpPerLevel,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['card_id'],
            $row['amount_per_sec'],
            $row['level'],
            $row['total_exp'],
            $row['base_amount_per_sec'],
            $row['max_level'],
            $row['max_amount_per_sec'],
            $row['base_exp_per_level'],
        );
    }
}

class UpdateDeckRequest
{
    public readonly string $viewerID;
    /** @var list<int> */
    public readonly array $cardIDs;

    /**
     * @throws JsonException
     * @throws TypeError
     */
    public function __construct(string $json)
    {
        $data = json_decode($json, flags: JSON_THROW_ON_ERROR);

        $this->viewerID = $data->viewerId;
        $this->cardIDs = $data->cardIds;
    }
}

class UpdateDeckResponse implements JsonSerializable
{
    public function __construct(public UpdatedResource $updatedResources)
    {
    }

    public function jsonSerialize(): array
    {
        return ['updatedResources' => $this->updatedResources];
    }
}

class RewardRequest
{
    public readonly string $viewerID;

    /**
     * @throws JsonException
     * @throws TypeError
     */
    public function __construct(string $json)
    {
        $data = json_decode($json, flags: JSON_THROW_ON_ERROR);

        $this->viewerID = $data->viewerId;
    }
}

class RewardResponse implements JsonSerializable
{
    public function __construct(public UpdatedResource $updateResources)
    {
    }

    public function jsonSerialize(): array
    {
        return ['updatedResources' => $this->updateResources];
    }
}

class HomeResponse implements JsonSerializable
{
    public function __construct(
        public int $now,
        public User $user,
        public ?UserDeck $deck,
        public int $totalAmountPerSec,
        public int $pastTime, // 経過時間を秒単位で
    ) {
    }

    public function jsonSerialize(): array
    {
        $data = [
            'now' => $this->now,
            'user' => $this->user,
            'totalAmountPerSec' => $this->totalAmountPerSec,
            'pastTime' => $this->pastTime,
        ];

        if (!is_null($this->deck)) {
            $data['deck'] = $this->deck;
        }

        return $data;
    }
}

// //////////////////////////////////////
// util

class UpdatedResource implements JsonSerializable
{
    /**
     * @param ?list<UserCard> $userCards
     * @param ?list<UserDeck> $userDecks
     * @param ?list<UserItem> $userItems
     * @param ?list<UserLoginBonus> $userLoginBonuses
     * @param ?list<UserPresent> $userPresents
     */
    public function __construct(
        public int $now,
        public ?User $user,
        public ?UserDevice $userDevice,
        public ?array $userCards,
        public ?array $userDecks,
        public ?array $userItems,
        public ?array $userLoginBonuses,
        public ?array $userPresents,
    ) {
    }

    public function jsonSerialize(): array
    {
        $data = [
            'now' => $this->now,
            'user' => $this->user,
            'userDevice' => $this->userDevice,
            'userCards' => $this->userCards,
            'userDecks' => $this->userDecks,
            'userItems' => $this->userItems,
            'userLoginBonuses' => $this->userLoginBonuses,
            'userPresents' => $this->userPresents,
        ];

        $optionalPropertyNames = [
            'user',
            'userDevice',
            'userCards',
            'userDecks',
            'userItems',
            'userLoginBonuses',
            'userPresents',
        ];

        foreach ($optionalPropertyNames as $propertyName) {
            if (is_null($data[$propertyName])) {
                unset($data[$propertyName]);
            }
        }

        return $data;
    }
}

// //////////////////////////////////////
// entity

class User implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $isuCoin,
        public int $lastGetRewardAt,
        public int $lastActivatedAt,
        public int $registeredAt,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['isu_coin'],
            $row['last_getreward_at'],
            $row['last_activated_at'],
            $row['registered_at'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }

    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'isuCoin' => $this->isuCoin,
            'lastGetRewardAt' => $this->lastGetRewardAt,
            'lastActivatedAt' => $this->lastActivatedAt,
            'registeredAt' => $this->registeredAt,
            'createdAt' => $this->createdAt,
            'updatedAt' => $this->updatedAt,
        ];

        if (!is_null($this->deletedAt)) {
            $data['deletedAt'] = $this->deletedAt;
        }

        return $data;
    }
}

class UserDevice implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $userID,
        public string $platformID,
        public int $platformType,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['platform_id'],
            $row['platform_type'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }

    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'userId' => $this->userID,
            'platformId' => $this->platformID,
            'platformType' => $this->platformType,
            'createdAt' => $this->createdAt,
            'updatedAt' => $this->updatedAt,
        ];

        if (!is_null($this->deletedAt)) {
            $data['deletedAt'] = $this->deletedAt;
        }

        return $data;
    }
}

class UserBan
{
    public function __construct(
        public int $id,
        public int $userID,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }
}

class UserCard implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $userID,
        public int $cardID,
        public int $amountPerSec,
        public int $level,
        public int $totalExp,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['card_id'],
            $row['amount_per_sec'],
            $row['level'],
            $row['total_exp'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }

    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'userId' => $this->userID,
            'cardId' => $this->cardID,
            'amountPerSec' => $this->amountPerSec,
            'level' => $this->level,
            'totalExp' => $this->totalExp,
            'createdAt' => $this->createdAt,
            'updatedAt' => $this->updatedAt,
        ];

        if (!is_null($this->deletedAt)) {
            $data['deletedAt'] = $this->deletedAt;
        }

        return $data;
    }
}

class UserDeck implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $userID,
        public int $cardID1,
        public int $cardID2,
        public int $cardID3,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['user_card_id_1'],
            $row['user_card_id_2'],
            $row['user_card_id_3'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }

    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'userId' => $this->userID,
            'cardId1' => $this->cardID1,
            'cardId2' => $this->cardID2,
            'cardId3' => $this->cardID3,
            'createdAt' => $this->createdAt,
            'updatedAt' => $this->updatedAt,
        ];

        if (!is_null($this->deletedAt)) {
            $data['deletedAt'] = $this->deletedAt;
        }

        return $data;
    }
}

class UserItem implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $userID,
        public int $itemType,
        public int $itemID,
        public int $amount,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['item_type'],
            $row['item_id'],
            $row['amount'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }

    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'userId' => $this->userID,
            'itemType' => $this->itemType,
            'itemId' => $this->itemID,
            'amount' => $this->amount,
            'createdAt' => $this->createdAt,
            'updatedAt' => $this->updatedAt,
        ];

        if (!is_null($this->deletedAt)) {
            $data['deletedAt'] = $this->deletedAt;
        }

        return $data;
    }
}

class UserLoginBonus implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $userID,
        public int $loginBonusID,
        public int $lastRewardSequence,
        public int $loopCount,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['login_bonus_id'],
            $row['last_reward_sequence'],
            $row['loop_count'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }

    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'userId' => $this->userID,
            'loginBonusId' => $this->loginBonusID,
            'lastRewardSequence' => $this->lastRewardSequence,
            'loopCount' => $this->loopCount,
            'createdAt' => $this->createdAt,
            'updatedAt' => $this->updatedAt,
        ];

        if (!is_null($this->deletedAt)) {
            $data['deletedAt'] = $this->deletedAt;
        }

        return $data;
    }
}

class UserPresent implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $userID,
        public int $sentAt,
        public int $itemType,
        public int $itemID,
        public int $amount,
        public string $presentMessage,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['sent_at'],
            $row['item_type'],
            $row['item_id'],
            $row['amount'],
            $row['present_message'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }

    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'userId' => $this->userID,
            'sentAt' => $this->sentAt,
            'itemType' => $this->itemType,
            'itemId' => $this->itemID,
            'amount' => $this->amount,
            'presentMessage' => $this->presentMessage,
            'createdAt' => $this->createdAt,
            'updatedAt' => $this->updatedAt,
        ];

        if (!is_null($this->deletedAt)) {
            $data['deletedAt'] = $this->deletedAt;
        }

        return $data;
    }
}

class UserPresentAllReceivedHistory implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $userID,
        public int $presentAllID,
        public int $receivedAt,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['present_all_id'],
            $row['received_at'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'userId' => $this->userID,
            'presentAllId' => $this->presentAllID,
            'receivedAt' => $this->receivedAt,
            'createdAt' => $this->createdAt,
            'updatedAt' => $this->updatedAt,
            'deletedAt' => $this->deletedAt,
        ];
    }
}

class Session implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $userID,
        public string $sessionID,
        public int $createdAt,
        public int $updatedAt,
        public int $expiredAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['session_id'],
            $row['created_at'],
            $row['updated_at'],
            $row['expired_at'],
            $row['deleted_at'],
        );
    }

    public function jsonSerialize(): array
    {
        $data = [
            'id' => $this->id,
            'userId' => $this->userID,
            'sessionId' => $this->sessionID,
            'createdAt' => $this->createdAt,
            'updatedAt' => $this->updatedAt,
            'expiredAt' => $this->expiredAt,
        ];

        if (!is_null($this->deletedAt)) {
            $data['deletedAt'] = $this->deletedAt;
        }

        return $data;
    }
}

class UserOneTimeToken
{
    public function __construct(
        public int $id,
        public int $userID,
        public string $token,
        public int $tokenType,
        public int $createdAt,
        public int $updatedAt,
        public int $expiredAt,
        public ?int $deletedAt = null,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['user_id'],
            $row['token'],
            $row['token_type'],
            $row['created_at'],
            $row['updated_at'],
            $row['expired_at'],
            $row['deleted_at'],
        );
    }
}

// //////////////////////////////////////
// master

class GachaMaster implements JsonSerializable
{
    public function __construct(
        public int $id,
        public string $name,
        public int $startAt,
        public int $endAt,
        public int $displayOrder,
        public int $createdAt,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['name'],
            $row['start_at'],
            $row['end_at'],
            $row['display_order'],
            $row['created_at'],
        );
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'name' => $this->name,
            'startAt' => $this->startAt,
            'endAt' => $this->endAt,
            'displayOrder' => $this->displayOrder,
            'createdAt' => $this->createdAt,
        ];
    }
}

class GachaItemMaster implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $gachaID,
        public int $itemType,
        public int $itemID,
        public int $amount,
        public int $weight,
        public int $createdAt,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['gacha_id'],
            $row['item_type'],
            $row['item_id'],
            $row['amount'],
            $row['weight'],
            $row['created_at'],
        );
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'gachaId' => $this->gachaID,
            'itemType' => $this->itemType,
            'itemId' => $this->itemID,
            'amount' => $this->amount,
            'weight' => $this->weight,
            'createdAt' => $this->createdAt,
        ];
    }
}

class ItemMaster implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $itemType,
        public string $name,
        public string $description,
        public ?int $amountPerSec,
        public ?int $maxLevel,
        public ?int $maxAmountPerSec,
        public ?int $baseExpPerLevel,
        public ?int $gainedExp,
        public ?int $shorteningMin,
        // public int $createdAt,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['item_type'],
            $row['name'],
            $row['description'],
            $row['amount_per_sec'],
            $row['max_level'],
            $row['max_amount_per_sec'],
            $row['base_exp_per_level'],
            $row['gained_exp'],
            $row['shortening_min'],
            // $row['created_at'],
        );
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'itemType' => $this->itemType,
            'name' => $this->name,
            'description' => $this->description,
            'amountPerSec' => $this->amountPerSec,
            'maxLevel' => $this->maxLevel,
            'maxAmountPerSec' => $this->maxAmountPerSec,
            'baseExpPerLevel' => $this->baseExpPerLevel,
            'gainedExp' => $this->gainedExp,
            'shorteningMin' => $this->shorteningMin,
            // 'createdAt' => $this->createdAt,
        ];
    }
}

class LoginBonusMaster implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $startAt,
        public int $endAt,
        public int $columnCount,
        public bool $looped,
        public int $createdAt,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['start_at'],
            $row['end_at'],
            $row['column_count'],
            (bool)$row['looped'],
            $row['created_at'],
        );
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'startAt' => $this->startAt,
            'endAt' => $this->endAt,
            'columnCount' => $this->columnCount,
            'looped' => $this->looped,
            'createdAt' => $this->createdAt,
        ];
    }
}

class LoginBonusRewardMaster implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $loginBonusID,
        public int $rewardSequence,
        public int $itemType,
        public int $itemID,
        public int $amount,
        public int $createdAt,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['login_bonus_id'],
            $row['reward_sequence'],
            $row['item_type'],
            $row['item_id'],
            $row['amount'],
            $row['created_at'],
        );
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'loginBonusId' => $this->loginBonusID,
            'rewardSequence' => $this->rewardSequence,
            'itemType' => $this->itemType,
            'itemId' => $this->itemID,
            'amount' => $this->amount,
            'createdAt' => $this->createdAt,
        ];
    }
}

class PresentAllMaster implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $registeredStartAt,
        public int $registeredEndAt,
        public int $itemType,
        public int $itemID,
        public int $amount,
        public string $presentMessage,
        public int $createdAt,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['registered_start_at'],
            $row['registered_end_at'],
            $row['item_type'],
            $row['item_id'],
            $row['amount'],
            $row['present_message'],
            $row['created_at'],
        );
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'registeredStartAt' => $this->registeredStartAt,
            'registeredEndAt' => $this->registeredEndAt,
            'itemType' => $this->itemType,
            'itemID' => $this->itemID,
            'amount' => $this->amount,
            'presentMessage' => $this->presentMessage,
            'createdAt' => $this->createdAt,
        ];
    }
}

class VersionMaster implements JsonSerializable
{
    public function __construct(
        public int $id,
        public int $status,
        public string $masterVersion,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['status'],
            $row['master_version'],
        );
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'status' => $this->status,
            'masterVersion' => $this->masterVersion,
        ];
    }
}
