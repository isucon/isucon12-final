use File::Basename;
use Plack::Builder;

use IsuConquest::API;
use IsuConquest::Admin;

my $root_dir = File::Basename::dirname(__FILE__);

my $api   = IsuConquest::API->psgi($root_dir);
my $admin = IsuConquest::Admin->psgi($root_dir);

builder {
    enable 'ReverseProxy';
    enable 'CrossOrigin',
         origins => '*',
         methods => [qw/GET POST/],
         headers => [qw/Content-Type x-master-version x-session/];

    mount '/admin/' => $admin;
    mount '/' => $api;
}
