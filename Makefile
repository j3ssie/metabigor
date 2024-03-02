TARGET   ?= metabigor
GO       ?= go
GOFLAGS  ?= 

build:
	go install
	rm -rf ./dist/*
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o dist/$(TARGET)
	zip -j dist/$(TARGET)-darwin.zip dist/$(TARGET)
	rm -rf ./dist/$(TARGET)
	# for linux build on mac
	GOOS=linux GOARCH=amd64 go build -o dist/$(TARGET)
	zip -j dist/$(TARGET)-linux.zip dist/$(TARGET)
	rm -rf ./dist/$(TARGET)

update:
	rm -rf $(GOPATH)/src/github.com/j3ssie/metabigor/modules/static/ip2asn-combined.tsv.gz
	wget -q https://iptoasn.com/data/ip2asn-combined.tsv.gz -O $(GOPATH)/src/github.com/j3ssie/metabigor/modules/static/ip2asn-combined.tsv.gz
	echo "Done."

run:
	$(GO) $(GOFLAGS) run *.go

test:
	$(GO) $(GOFLAGS) test ./... -v%