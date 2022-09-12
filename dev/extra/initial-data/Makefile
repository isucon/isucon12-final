.PHONY: dump-master
dump-master:
	mysqldump -uisucon -pisucon --host 127.0.0.1 isucon \
	gacha_item_masters \
	gacha_masters \
	id_generator \
	item_masters \
	login_bonus_masters \
	login_bonus_reward_masters \
	present_all_masters \
	version_masters \
	admin_users \
	--no-tablespaces > 80_master_data.sql

.PHONY: dump-not-receive-presents
dump-not-receive-presents:
	# echo "SELECT * FROM user_presents where deleted_at is null ;" | mysql -uisucon -pisucon --host 127.0.0.1 isucon > 85_present_not_receive_data.tsv

	echo "SELECT id,user_id,sent_at,item_type,item_id,amount,present_message,created_at,updated_at,'|N' as deleted_at FROM user_presents where deleted_at is null ;" | mysql -uisucon -pisucon --host 127.0.0.1 isucon > 85_present_not_receive_data.tsv

.PHONEY: dump-all-exclude-presents
dump-all-exclude-presents:
	mysqldump -uisucon -pisucon --host 127.0.0.1 isucon \
		admin_sessions                    \
		admin_users                       \
		gacha_item_masters                \
		gacha_masters                     \
		item_masters                      \
		login_bonus_masters               \
		login_bonus_reward_masters        \
		present_all_masters               \
		user_bans                         \
		user_cards                        \
		user_decks                        \
		user_devices                      \
		user_items                        \
		user_login_bonuses                \
		user_one_time_tokens              \
		user_present_all_received_history \
		user_sessions                     \
		users                             \
		version_masters  				  \
		--no-tablespaces --no-create-info > 88_exclude_user_presents_data.sql

.PHONY: setup-master
setup-master:
	mysql -uisucon -pisucon --host 127.0.0.1 isucon < ./80_master_data.sql

.PHONEY: setup-presents
setup-presents:
	echo "LOAD DATA INFILE '/docker-entrypoint-initdb.d/85_present_not_receive_data.tsv' REPLACE INTO TABLE user_presents FIELDS ESCAPED BY '|' IGNORE 1 LINES ;" | mysql -uisucon -pisucon --host 127.0.0.1 isucon 

.PHONY: reset
reset:
	mysql -uisucon -pisucon --host 127.0.0.1 < ../../../webapp/sql/setup/1_schema.sql

.PHONY: clean
clean:
	rm -rf *.sql
	rm -rf *.json

.PHONY: build
build: reset setup-master
	@go run generate.go 10
	@mysqldump -uisucon -pisucon --host 127.0.0.1 --databases isucon --no-create-db --no-tablespaces --no-create-info > 90_data.sql

.PHONY: build-production
build-production: reset setup-master
	@go run generate.go 3333
	@mysqldump -uisucon -pisucon --host 127.0.0.1 --databases isucon --no-create-db --no-tablespaces --no-create-info > 90_data.sql

.PHONY: install
install: dump-all-exclude-presents dump-not-receive-presents
	cp 90_data.sql ../../../webapp/sql/setup/2_init.sql
	cp 88_exclude_user_presents_data.sql ../../../webapp/sql/4_alldata_exclude_user_presents.sql
	cp 85_present_not_receive_data.tsv ../../../webapp/sql/5_user_presents_not_receive_data.tsv
	cp *.json ../../../benchmarker/dump/

