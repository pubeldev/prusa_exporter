.PHONY = up build build-docker provision

up:
	./scripts/start_docker.sh

build:
	./scripts/build.sh

build-docker:
	docker buildx build -t pubeldev/prusa_exporter .

provision:
	# Root permission is required to chown to user 'grafana'
	sudo ./scripts/download_grafana_plugins.sh
