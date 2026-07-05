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
│   └── main.go
├── api/
│   └── v1alpha1/
│       ├── foo_types.go                  # CRD定義
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go
├── internal/
│   └── controller/
│       └── foo_controller.go             # 本体ロジック(Reconcile)
├── config/
│   ├── crd/
│   │   ├── bases/
│   │   │   └── *.yaml
│   │   └── kustomization.yaml
│   ├── rbac/
│   │   ├── role.yaml
│   │   ├── role_binding.yaml
│   │   └── service_account.yaml
│   ├── manager/
│   │   └── manager.yaml                   # ControllerのDeployment定義(standby数など)
│   ├── default/
│   │   └── kustomization.yaml
│   └── samples/
│       └── *.yaml
├── Makefile
└── Dockerfile
```