# build.mk
# This file contains make tasks for building.

.PHONY=watch-web
watch-web: prepare-frontend
	cd web && npx ng serve -c dev


.PHONY=build-web
build-web: prepare-frontend
	cd web && npx ng build --output-path ../dist -c prod