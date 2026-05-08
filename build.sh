manager=1.0.0
hub=1.0.2

# Build manager
cd /root/impahub/impa-hub-manager
docker build -t ghcr.io/adfastltda/impahub-manager:$manager \
             -t ghcr.io/adfastltda/impahub-manager:latest \
             -f /root/impahub/impa-hub-manager/Dockerfile .

docker push ghcr.io/adfastltda/impahub-manager:$manager
docker push ghcr.io/adfastltda/impahub-manager:latest


# Verificações de sintaxe antes do build (opcionais)
echo "=== Verificando sintaxe Go (opcional) ==="
cd /root/impahub/impa-hub

# Verifica se Go está disponível
if command -v go &> /dev/null; then
    # Verifica formatação
    echo "Verificando formatação..."
    gofmt -w . 2>/dev/null || echo "AVISO: gofmt falhou"
    
    # Verifica sintaxe com go vet
    echo "Verificando sintaxe com go vet..."
    go vet ./... 2>/dev/null || echo "AVISO: go vet encontrou problemas"
    
    # Tenta compilação local (não bloqueia se falhar)
    echo "Testando compilação local..."
    if go build -o /tmp/impa-hub-test ./cmd/impa-hub 2>/dev/null; then
        rm -f /tmp/impa-hub-test
        echo "✓ Compilação local OK"
    else
        echo "AVISO: Compilação local falhou (Go pode não estar disponível)"
        echo "Continuando com build Docker..."
    fi
else
    echo "AVISO: Go não encontrado localmente, pulando verificações"
fi

# Build hub
echo "=== Iniciando build Docker ==="
docker build -t ghcr.io/adfastltda/impahub:$hub \
             -t ghcr.io/adfastltda/impahub:latest \
             -f /root/impahub/impa-hub/Dockerfile .

docker push ghcr.io/adfastltda/impahub:$hub
docker push ghcr.io/adfastltda/impahub:latest


curl https://easypanel.adfast.com.br/api/deploy/b49283631db92f3c85c206224b31429aab861342876ca4d8