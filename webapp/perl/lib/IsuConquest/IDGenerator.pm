package IsuConquest::IDGenerator;
use v5.36;
use experimental qw(try);

use Exporter 'import';

our @EXPORT_OK = qw(
    generate_id
    generate_uuid
);

use Carp qw(croak);
use Data::UUID;

use IsuConquest::DBConnector qw(connect_db);

sub generate_id() {
    my $dbh = connect_db;
    my $update_err;
    for my $i (0 .. 100) {
        try {
            $dbh->query("UPDATE id_generator SET id=LAST_INSERT_ID(id+1)");
        }
        catch ($e) {
            no warnings qw(once);
            if ($DBI::err == 1213) {
                $update_err = $e;
                next;
            }
            croak($e);
        }

        my $id = $dbh->last_insert_id;
        return $id;
    }
    croak(sprintf('failed to generate id: %s', $update_err));
}

sub generate_uuid() {
    my $ug = Data::UUID->new;
    return $ug->create_str;
}

1;
