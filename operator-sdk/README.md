# operator-sdk
operator-sdkで開発したFoo Operator

# コンパイルからデプロイ手順
## CRD・RBAC・コード生成
```
make manifests
```
## コントローラーをコンテナイメージにする
```
make docker-build
```
## 作成したイメージをtarにする
```
docker save -o foo-operator.tar foo-operator:dev
```
## 作成したイメージをkubernetesに読み込ませる
```
sudo k3s ctr images import foo-operator.tar
```
- 同じタグのイメージがあると、古い`daigest`を使用してしまうため、一度削除
```
sudo k3s ctr images rm docker.io/library/foo-operator:dev
```
## kubernetesへデプロイ
```
make deploy
```
## 削除
```
make undeploy
```
# webhookの追加
## Validation Webhook
```sh
operator-sdk create webhook \
  --group samplecontroller \
  --version v1alpha1 \
  --kind Foo \
  --defaulting \
  --programmatic-validation
```
## cert-managerをインストール
```
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
```

# 主要コンポーネント
```
.
├── cmd/
│   └── main.go                            # Manager・Controller・Webhookの起動
├── api/
│   └── v1alpha1/
│       ├── foo_types.go                   # CRD定義
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go
├── internal/
│   ├── controller/
│   │   └── foo_controller.go              # Reconcile処理
│   └── webhook/
│       └── v1alpha1/
│           └── foo_webhook.go             # Default/Validateの実装
├── config/
│   ├── crd/
│   │   ├── bases/
│   │   │   └── *.yaml                     # CRD
│   │   └── kustomization.yaml
│   ├── webhook/
│   │   ├── manifests.yaml                 # Mutating/ValidatingWebhookConfiguration
│   │   ├── service.yaml                   # Webhook Service
│   │   └── kustomization.yaml
│   ├── certmanager/
│   │   ├── issuer.yaml                    # Issuer
│   │   ├── certificate-webhook.yaml       # Webhook用証明書
│   │   ├── certificate-metrics.yaml       # Metrics用証明書
│   │   └── kustomization.yaml
│   ├── rbac/
│   │   ├── role.yaml
│   │   ├── role_binding.yaml
│   │   └── service_account.yaml
│   ├── manager/
│   │   └── manager.yaml                   # Controller Deployment
│   ├── default/
│   │   ├── kustomization.yaml             # 全リソースをまとめてデプロイ
│   │   ├── manager_webhook_patch.yaml     # Webhook有効化
│   │   └── manager_metrics_patch.yaml
│   └── samples/
│       └── *.yaml                         # サンプルCR
├── Makefile
└── Dockerfile
```