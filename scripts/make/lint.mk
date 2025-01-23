.PHONY=lint-web
lint-web: prepare-frontend
	cd web && npx ng lint && npx stylelint "*/**.sass"

.PHONY=lint-go
lint-go:
	go vet ./...

.PHONY=format-go
format-go:
	gofmt -s -w .

.PHONY=format-web
format-web: prepare-frontend
	cd web && npx prettier --ignore-path .gitignore --write "./**/*.+(scss|ts|json|html)"

.PHONY=check-format-go
check-format-go:
	test -z `gofmt -l .`

.PHONY=check-format-web
check-format-web: prepare-frontend
	cd web && npx prettier --ignore-path .gitignore --check "./**/*.+(scss|ts|json|html)"