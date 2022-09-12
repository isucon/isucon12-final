<?php

declare(strict_types=1);

use App\Application\Settings\Settings;
use App\Application\Settings\SettingsInterface;
use DI\ContainerBuilder;
use Monolog\Level as LogLevel;

return function (ContainerBuilder $containerBuilder) {

    // Global Settings Object
    $containerBuilder->addDefinitions([
        SettingsInterface::class => function () {
            return new Settings([
                'logError'            => true,
                'logErrorDetails'     => true,
                'logger' => [
                    'name' => 'isu-conquest',
                    'path' => 'php://stdout',
                    'level' => LogLevel::Debug,
                ],
                'database' => [
                    'host' => getenv('ISUCON_DB_HOST') ?: '127.0.0.1',
                    'database' => getenv('ISUCON_DB_NAME') ?: 'isucon',
                    'port' => getenv('ISUCON_DB_PORT') ?: '3306',
                    'user' => getenv('ISUCON_DB_USER') ?: 'isucon',
                    'password' => getenv('ISUCON_DB_PASSWORD') ?: 'isucon',
                ],
            ]);
        }
    ]);
};
