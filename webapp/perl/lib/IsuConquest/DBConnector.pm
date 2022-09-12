package IsuConquest::DBConnector;
use v5.36;

use Exporter 'import';

use DBIx::Sunny;

our @EXPORT_OK = qw(connect_db);

sub connect_db() {
    my $user     = $ENV{ISUCON_DB_USER}       || 'isucon';
    my $password = $ENV{ISUCON_DB_PASSWORD}   || 'isucon';
    my $host     = $ENV{ISUCON_DB_HOST}       || '127.0.0.1';
    my $port     = $ENV{ISUCON_DB_PORT}       || '3306';
    my $dbname   = $ENV{ISUCON_DB_NAME}       || 'isucon';

    my $dsn = "dbi:mysql:database=$dbname;host=$host;port=$port";
    my $dbh = DBIx::Sunny->connect($dsn, $user, $password, {
        mysql_enable_utf8mb4 => 1,
        mysql_auto_reconnect => 1,
    });
    return $dbh;
}

1;
