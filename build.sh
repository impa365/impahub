manager=1.0.0
hub=1.0.0

# Build manager
cd /root/impahub/impa-hub-manager
docker build -t ghcr.io/adfastltda/impahub-manager:$manager \
             -t ghcr.io/adfastltda/impahub-manager:latest \
             -f /root/impahub/impa-hub-manager/Dockerfile .

docker push ghcr.io/adfastltda/impahub-manager:$manager
docker push ghcr.io/adfastltda/impahub-manager:latest


# Build hub
cd /root/impahub/impa-hub
docker build -t ghcr.io/adfastltda/impahub:$hub \
             -t ghcr.io/adfastltda/impahub:latest \
             -f /root/impahub/impa-hub/Dockerfile .

docker push ghcr.io/adfastltda/impahub:$hub
docker push ghcr.io/adfastltda/impahub:latest


curl https://easypanel.adfast.com.br/api/deploy/b49283631db92f3c85c206224b31429aab861342876ca4d8