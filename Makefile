serve:
	reflex -s -R 'Makefile' -R '.log$$' -R '_test.go$$'\
		-- go run cmd/godoc/*.go -v