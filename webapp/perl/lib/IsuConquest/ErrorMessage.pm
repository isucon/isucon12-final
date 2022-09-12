package IsuConquest::ErrorMessage;
use v5.36;

use Exporter 'import';

our @EXPORT_OK = qw(
    ErrInvalidRequestBody
    ErrInvalidMasterVersion
    ErrInvalidItemType
    ErrInvalidToken
    ErrGetRequestTime
    ErrExpiredSession
    ErrUserNotFound
    ErrUserDeviceNotFound
    ErrItemNotFound
    ErrLoginBonusRewardNotFound
    ErrNoFormFile
    ErrUnauthorized
    ErrForbidden
    ErrGeneratePassword
);

use constant {
    ErrInvalidRequestBody       => 'invalid request body',
    ErrInvalidMasterVersion     => 'invalid master version',
    ErrInvalidItemType          => 'invalid item type',
    ErrInvalidToken             => 'invalid token',
    ErrGetRequestTime           => 'failed to get request time',
    ErrExpiredSession           => 'session expired',
    ErrUserNotFound             => 'not found user',
    ErrUserDeviceNotFound       => 'not found user device',
    ErrItemNotFound             => 'not found item',
    ErrLoginBonusRewardNotFound => 'not found login bonus reward',
    ErrNoFormFile               => 'no such file',
    ErrUnauthorized             => 'unauthorized user',
    ErrForbidden                => 'forbidden',
    ErrGeneratePassword         => 'failed to password hash',
};

1;
