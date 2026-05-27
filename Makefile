BINARY    = heic-to-jpg
PLIST     = com.panphora.heic-to-jpg.plist
PLIST_DIR = $(HOME)/Library/LaunchAgents

.PHONY: build install uninstall

build:
	go build -o $(BINARY)

install: build
	cp $(PLIST) $(PLIST_DIR)/$(PLIST)
	launchctl unload $(PLIST_DIR)/$(PLIST) 2>/dev/null; true
	launchctl load $(PLIST_DIR)/$(PLIST)
	@echo "installed and running"

uninstall:
	launchctl unload $(PLIST_DIR)/$(PLIST) 2>/dev/null; true
	rm -f $(PLIST_DIR)/$(PLIST)
	@echo "uninstalled"
