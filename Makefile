SERVICE_NAME = linkServce

# ====================== GENERATION ======================

.PHONY: proto
proto: ## Generate gRPC + Gateway code
	@echo "Generating protobuf code..."
	./bin/buf.exe generate -v ./api/proto --template=buf.gen.yaml
	git add generated
	@echo "Done."