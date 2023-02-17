preinstall:
	-rm -rf $(HOME)/.config/systemd/user/repo-indexer.service

install: preinstall
	$(warning "NOTE! Do not forget to add $GO/bin to PATH")
	go install cmd/indexer/repo-indexer.go
	go install cmd/lister/rofi-repos.go
	mkdir -p $(HOME)/.config/systemd/user
	ln -s $(shell pwd)/repo-indexer.service $(HOME)/.config/systemd/user/repo-indexer.service

reinstall:
	go install cmd/indexer/repo-indexer.go
	go install cmd/lister/rofi-repos.go