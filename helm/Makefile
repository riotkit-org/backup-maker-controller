move-image-to-registry:
	docker tag ghcr.io/riotkit-org/backup-maker-operator:latest bm-registry.localhost:5000/backup-maker-operator:latest
	docker push bm-registry.localhost:5000/backup-maker-operator:latest

install:
	kubectl create ns backup-maker-operator || true
	helm upgrade --install -n backup-maker-operator bmo ./ --set image.repository=bm-registry:5000/backup-maker-operator --set image.tag=latest
