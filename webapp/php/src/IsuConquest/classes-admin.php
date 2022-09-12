<?php

declare(strict_types=1);

namespace App\IsuConquest;

use JsonException;
use JsonSerializable;
use TypeError;

// //////////////////////////////////////
// admin

class AdminLoginRequest
{
    public readonly int $userID;
    public readonly string $password;

    /**
     * @throws JsonException
     * @throws TypeError
     */
    public function __construct(string $json)
    {
        $data = json_decode($json, flags: JSON_THROW_ON_ERROR);

        $this->userID = $data->userId;
        $this->password = $data->password;
    }
}

class AdminLoginResponse implements JsonSerializable
{
    public function __construct(public Session $adminSession)
    {
    }

    public function jsonSerialize(): array
    {
        return ['session' => $this->adminSession];
    }
}

class AdminListMasterResponse implements JsonSerializable
{
    /**
     * @param list<VersionMaster> $versionMaster
     * @param list<ItemMaster> $items
     * @param list<GachaMaster> $gachas
     * @param list<GachaItemMaster> $gachaItems
     * @param list<PresentAllMaster> $presentAlls
     * @param list<LoginBonusRewardMaster> $loginBonusRewards
     * @param list<LoginBonusMaster> $loginBonuses
     */
    public function __construct(
        public array $versionMaster,
        public array $items,
        public array $gachas,
        public array $gachaItems,
        public array $presentAlls,
        public array $loginBonusRewards,
        public array $loginBonuses,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'versionMaster' => $this->versionMaster,
            'items' => $this->items,
            'gachas' => $this->gachas,
            'gachaItems' => $this->gachaItems,
            'presentAlls' => $this->presentAlls,
            'loginBonusRewards' => $this->loginBonusRewards,
            'loginBonuses' => $this->loginBonuses,
        ];
    }
}

class AdminUpdateMasterResponse implements JsonSerializable
{
    public function __construct(public VersionMaster $versionMaster)
    {
    }

    public function jsonSerialize(): array
    {
        return ['versionMaster' => $this->versionMaster];
    }
}

class AdminUserResponse implements JsonSerializable
{
    /**
     * @param list<UserDevice> $userDevices
     * @param list<UserCard> $userCards
     * @param list<UserDeck> $userDecks
     * @param list<UserItem> $userItems
     * @param list<UserLoginBonus> $userLoginBonuses
     * @param list<UserPresent> $userPresents
     * @param list<UserPresentAllReceivedHistory> $userPresentAllReceivedHistory
     */
    public function __construct(
        public User $user,
        public array $userDevices,
        public array $userCards,
        public array $userDecks,
        public array $userItems,
        public array $userLoginBonuses,
        public array $userPresents,
        public array $userPresentAllReceivedHistory,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'user' => $this->user,
            'userDevices' => $this->userDevices,
            'userCards' => $this->userCards,
            'userDecks' => $this->userDecks,
            'userItems' => $this->userItems,
            'userLoginBonuses' => $this->userLoginBonuses,
            'userPresents' => $this->userPresents,
            'userPresentAllReceivedHistory' => $this->userPresentAllReceivedHistory,
        ];
    }
}

class AdminBanUserResponse implements JsonSerializable
{
    public function __construct(public User $user)
    {
    }

    public function jsonSerialize(): array
    {
        return ['user' => $this->user];
    }
}

class AdminUser
{
    public function __construct(
        public int $id,
        public string $password,
        public int $lastActivatedAt,
        public int $createdAt,
        public int $updatedAt,
        public ?int $deletedAt,
    ) {
    }

    public static function fromDBRow(array $row): self
    {
        return new self(
            $row['id'],
            $row['password'],
            $row['last_activated_at'],
            $row['created_at'],
            $row['updated_at'],
            $row['deleted_at'],
        );
    }
}
