CREATE DATABASE IF NOT EXISTS `isucon`;
CREATE USER IF NOT EXISTS `isucon`@`%` IDENTIFIED WITH mysql_native_password BY 'isucon';
GRANT ALL ON `isucon`.* TO `isucon`@`%`;
GRANT FILE ON *.* to 'isucon'@'%';
