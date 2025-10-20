SHELL := /bin/zsh
GO_MOD_REMOTE := $$(git remote get-url origin | sed 's/git@//g;s/:/\//g;s/\.git$$//g')
APP_NAME := $(shell printf $(GO_MOD_REMOTE) | awk '{split($$0,a,"/");print a[3]}')


.PHONY: help # Show this Help
help:
	@echo -e "\033[1;37mAvailable Targets\033[0m"
	@cat Makefile | grep ".PHONY" | grep -v ".PHONY: _" | sed 's/.PHONY: //g' | sed 's/ # /\t/' | awk -F'\t' '{printf "\033[36m%-20s\033[0m%s\n", $$1, $$2}'


.PHONY: build # Compile the Code
build: go.mod main.go
	go build -o bin/$(APP_NAME) .


.PHONY: run # Run the App
run: build
	./bin/$(APP_NAME)


go.mod:
	@go mod init $(GO_MOD_REMOTE)/v2


main.go:
	@echo "package main%%import (%#\"fmt\"%#\"io\"%#\"os\"%)%%func _main(in io.Reader, out io.Writer) error {%#fmt.Fprintln(out, \"Hello World!\")%#return nil%}%%func main() {%#err := _main(os.Stdin, os.Stdout)%#if err != nil {%##panic(err)%#}%}%" | tr '#' '\t' | tr '%' '\n' | tee main.go


opt/eng-web_usfx.zip: opt/.gitignore
	curl -sL https://ebible.org/Scriptures/eng-web_usfx.zip > opt/eng-web_usfx.zip


opt/eng-web_usfx.zip.sha512sum: opt/eng-web_usfx.zip
	shasum -a 512 opt/eng-web_usfx.zip | tee opt/eng-web_usfx.zip.sha512sum


opt/usfx/eng-web/signature.txt.asc:
	make opt/eng-web_usfx.zip.sha512sum
	mkdir -p opt/usfx/eng-web
	cd opt/usfx/eng-web \
		&& unzip -u -o ../../eng-web_usfx.zip \
		&& sha256sum -c signature.txt.asc \
	;
