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