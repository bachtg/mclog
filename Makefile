gen-proto:
	protoc api/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.

# Makefile
.PHONY: gen-ca gen-server gen-client gen-all clean

CA_CONFIG_DIR = authority/ca-config
CERTS_DIR = authority/certs

gen-ca:
	cfssl gencert -initca $(CA_CONFIG_DIR)/ca-csr.json | cfssljson -bare $(CERTS_DIR)/ca

gen-server: gen-ca
	cfssl gencert \
		-ca=$(CERTS_DIR)/ca.pem \
		-ca-key=$(CERTS_DIR)/ca-key.pem \
		-config=$(CA_CONFIG_DIR)/ca-config.json \
		-profile=server \
		$(CA_CONFIG_DIR)/server.json | cfssljson -bare $(CERTS_DIR)/server

gen-client: gen-ca
	cfssl gencert \
		-ca=$(CERTS_DIR)/ca.pem \
		-ca-key=$(CERTS_DIR)/ca-key.pem \
		-config=$(CA_CONFIG_DIR)/ca-config.json \
		-profile=client \
		$(CA_CONFIG_DIR)/client.json | cfssljson -bare $(CERTS_DIR)/client

gen-all-cert: gen-ca gen-server gen-client

clean-certs:
	rm -rf $(CERTS_DIR)/*.pem $(CERTS_DIR)/*.key