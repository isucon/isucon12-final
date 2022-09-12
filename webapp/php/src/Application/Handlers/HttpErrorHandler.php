<?php

declare(strict_types=1);

namespace App\Application\Handlers;

use Psr\Http\Message\ResponseInterface as Response;
use Slim\Exception\HttpException;
use Slim\Handlers\ErrorHandler as SlimErrorHandler;

class HttpErrorHandler extends SlimErrorHandler
{
    /**
     * @inheritdoc
     */
    protected function respond(): Response
    {
        $exception = $this->exception;
        $statusCode = 500;

        if ($exception instanceof HttpException) {
            $statusCode = $exception->getCode();
        }

        $encodedPayload = json_encode([
            'status_code' => $statusCode,
            'message' => $exception->getMessage(),
        ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);

        $response = $this->responseFactory->createResponse($statusCode);
        $response->getBody()->write($encodedPayload);

        return $response->withHeader('Content-Type', 'application/json; charset=UTF-8');
    }
}
