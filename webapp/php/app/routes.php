<?php

declare(strict_types=1);

use App\IsuConquest\AdminHandler;
use App\IsuConquest\Handler;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\App;
use Slim\Routing\RouteCollectorProxy;

return function (App $app) {
    $app->options('/{routes:.*}', function (Request $request, Response $response) {
        // CORS Pre-Flight OPTIONS Request Handler
        return $response;
    });

    // utility
    $app->post('/initialize', Handler::class . ':initialize');
    $app->get('/health', Handler::class . ':health');

    // feature
    $app->group('', function (RouteCollectorProxy $api) {
        $api->post('/user', Handler::class . ':createUser');
        $api->post('/login', Handler::class . ':login');
        $api->group('', function (RouteCollectorProxy $sessCheckAPI) {
            $sessCheckAPI->get('/user/{userID}/gacha/index', Handler::class . ':listGacha');
            $sessCheckAPI->post('/user/{userID}/gacha/draw/{gachaID}/{n}', Handler::class . ':drawGacha');
            $sessCheckAPI->get('/user/{userID}/present/index/{n}', Handler::class . ':listPresent');
            $sessCheckAPI->post('/user/{userID}/present/receive', Handler::class . ':receivePresent');
            $sessCheckAPI->get('/user/{userID}/item', Handler::class . ':listItem');
            $sessCheckAPI->post('/user/{userID}/card/addexp/{cardID}', Handler::class . ':addExpToCard');
            $sessCheckAPI->post('/user/{userID}/card', Handler::class . ':updateDeck');
            $sessCheckAPI->post('/user/{userID}/reward', Handler::class . ':reward');
            $sessCheckAPI->get('/user/{userID}/home', Handler::class . ':home');
        })->add(Handler::class . ':checkSessionMiddleware');
    })->add(Handler::class . ':apiMiddleware');

    //admin
    $app->group('', function (RouteCollectorProxy $adminAPI) {
        $adminAPI->post('/admin/login', AdminHandler::class . ':adminLogin');
        $adminAPI->group('', function (RouteCollectorProxy $adminAuthAPI) {
            $adminAuthAPI->delete('/admin/logout', AdminHandler::class . ':adminLogout');
            $adminAuthAPI->get('/admin/master', AdminHandler::class . ':adminListMaster');
            $adminAuthAPI->put('/admin/master', AdminHandler::class . ':adminUpdateMaster');
            $adminAuthAPI->get('/admin/user/{userID}', AdminHandler::class . ':adminUser');
            $adminAuthAPI->post('/admin/user/{userID}/ban', AdminHandler::class . ':adminBanUser');
        })->add(AdminHandler::class . ':adminSessionCheckMiddleware');
    })->add(AdminHandler::class . ':adminMiddleware');
};
