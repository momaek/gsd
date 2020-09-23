generate:
	cd static && go generate

dev:
	reflex -s -R 'Makefile' -R '.zip$$' -R docs -R '.log$$' -R '_test.go$$'\
		-- go run cmd/gsd/main.go -http=:3000 $$args

# watch static assets, automatic generate
watch:
	cd static && reflex -s -R '.go$$'\
		-- go generate

serve:
	cd docs && python -m SimpleHTTPServer 8000

build:
	cd cmd/gsd; go install -trimpath

build_darwin:
	docker run --rm \
		-v "$$PWD":/gsd \
		-v "$$PWD"/.mod:/go/pkg/mod \
		-w /gsd \
		-e GOOS=darwin \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=1 \
		-e GOPROXY='https://goproxy.cn,direct' \
		docker.elastic.co/beats-dev/golang-crossbuild:1.14.7-darwin \
		--build-cmd "go build -o gsd-darwin-amd64 cmd/gsd/main.go" \
		-p 'darwin/amd64'
	docker run --rm \
		-v "$$PWD":/gsd \
		-v "$$PWD"/.mod:/go/pkg/mod \
		-w /gsd \
		-e GOOS=darwin \
		-e GOARCH=386 \
		-e CGO_ENABLED=1 \
		-e GOPROXY='https://goproxy.cn,direct' \
		docker.elastic.co/beats-dev/golang-crossbuild:1.14.7-darwin \
		--build-cmd "go build -o gsd-darwin-386 cmd/gsd/main.go" \
		-p 'darwin/386'

build_linux:
	docker run --rm \
		-v "$$PWD":/gsd \
		-v "$$PWD"/.mod:/go/pkg/mod \
		-w /gsd \
		-e GOOS=linux \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=1 \
		-e GOPROXY='https://goproxy.cn,direct' \
		golang:1.15 \
		go build -o gsd-linux-amd64 cmd/gsd/main.go
	docker run --rm \
		-v "$$PWD":/gsd \
		-v "$$PWD"/.mod:/go/pkg/mod \
		-w /gsd \
		-e GOOS=linux \
		-e GOARCH=386 \
		-e CGO_ENABLED=1 \
		-e GOPROXY='https://goproxy.cn,direct' \
		golang:1.15 \
		go build -o gsd-linux-386 cmd/gsd/main.go

build_windows:
	docker run --rm \
		-v "$$PWD":/gsd \
		-v "$$PWD"/.mod:/go/pkg/mod \
		-w /gsd \
		-e GOOS=windows \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=1 \
		-e GOPROXY='https://goproxy.cn,direct' \
		docker.elastic.co/beats-dev/golang-crossbuild:1.14.7-main \
		--build-cmd "go build -o gsd-windows-amd64 cmd/gsd/main.go" \
		-p 'windows/amd64'
	docker run --rm \
		-v "$$PWD":/gsd \
		-v "$$PWD"/.mod:/go/pkg/mod \
		-w /gsd \
		-e GOOS=windows \
		-e GOARCH=386 \
		-e CGO_ENABLED=1 \
		-e GOPROXY='https://goproxy.cn,direct' \
		docker.elastic.co/beats-dev/golang-crossbuild:1.14.7-main \
		--build-cmd "go build -o gsd-windows-386 cmd/gsd/main.go" \
		-p 'windows/386'

build_all: build_darwin build_windows build_linux
