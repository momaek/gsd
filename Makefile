generate:
	cd static && go generate

dev:
	reflex -s -R 'Makefile' -R '.zip$$' -R docs -R '.log$$' -R '_test.go$$'\
		-- go run cmd/gsd/main.go -http=:3000

# watch static assets, automatic generate
watch:
	cd static && reflex -s -R '.go$$'\
		-- go generate

serve:
	cd docs && python -m SimpleHTTPServer 8000

build:
	cd cmd/gsd; go install -trimpath
