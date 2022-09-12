require_relative 'app'

require 'rack/cors'

use Rack::Cors do
  allow do
    origins '*'
    resource '/*', methods: [:get, :post], headers: ['content-type', 'x-master-version', 'x-session']
  end
end

run Isuconquest::App
