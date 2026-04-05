APP_NAME=selfmailmerge

build:
	@echo "Building $(APP_NAME)"
	GOOS=darwin GOARCH=amd64 go build -o dist/$(APP_NAME)
	GOOS=windows GOARCH=amd64 go build -o dist/$(APP_NAME).exe
