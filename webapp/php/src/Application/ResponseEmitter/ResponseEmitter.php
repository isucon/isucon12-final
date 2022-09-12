<?php

declare(strict_types=1);

namespace App\Application\ResponseEmitter;

use Psr\Http\Message\ResponseInterface;
use Slim\ResponseEmitter as SlimResponseEmitter;

class ResponseEmitter extends SlimResponseEmitter
{
    /**
     * {@inheritdoc}
     */
    public function emit(ResponseInterface $response): void
    {
        $response = $response
            ->withHeader('Access-Control-Allow-Origin', '*')
            ->withHeader(
                'Access-Control-Allow-Headers',
                'Content-Type, x-master-version, x-session',
            )
            ->withHeader('Access-Control-Allow-Methods', 'GET, POST')
            ->withHeader('Cache-Control', 'no-store, no-cache, must-revalidate, max-age=0')
            ->withAddedHeader('Cache-Control', 'post-check=0, pre-check=0')
            ->withHeader('Pragma', 'no-cache');

        if (ob_get_contents()) {
            ob_clean();
        }

        parent::emit($response);
    }
}
