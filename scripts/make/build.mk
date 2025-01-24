# build.mk
# This file contains make tasks for building.

.PHONY=watch-web
watch-web: prepare-frontend
	cd web && NG_APP_BACKEND_URL_PREFIX="http://localhost:8080" npx ng serve -c dev


.PHONY=build-web
build-web: prepare-frontend
	cd web &&NG_APP_VERSION="$(VERSION)"  npx ng build --output-path ../dist -c prod