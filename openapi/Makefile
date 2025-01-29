.PHONY: generate-api clean

generate-api:
	@echo "Generating OpenAPI code..."
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=models.config.yaml api.yaml
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=server.config.yaml api.yaml
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=client.config.yaml api.yaml

clean:
	rm -f models.gen.go server.gen.go client.gen.go
