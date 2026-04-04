APP_NAME=selfmailmerge

build:
	@echo "Building $(APP_NAME) for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o dist/$(APP_NAME)
