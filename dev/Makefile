.PHONY: get-initial-data
get-initial-data:
	wget https://github.com/isucon/isucon12-final/releases/download/initial_data_20220912/initial_data.tar.gz
	mv initial_data.tar.gz ../

.PHONY: initial-data
initial-data: get-initial-data
	tar -C .. -xvf ../initial_data.tar.gz
	rm ../initial_data.tar.gz

packer-copy-app: initial-data
	cp ../webapp/sql/4_*.sql ../provisioning/packer/ansible/roles/webapp/files/home/isucon/webapp/sql/
	cp ../webapp/sql/5_*.tsv ../provisioning/packer/ansible/roles/webapp/files/home/isucon/webapp/sql/
	cp ../webapp/sql/setup/2_init.sql ../provisioning/packer/ansible/roles/webapp/files/home/isucon/webapp/sql/setup/

packer-copy-bench: initial-data
	cp ../benchmarker/dump/*.json ../provisioning/packer/ansible/roles/benchmarker/files/dump/
