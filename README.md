# [OCHaCafe Season5 #1 「Kubernetes Operator超入門」](https://ochacafe.connpass.com/event/232810/)の"任意のDeploymentを作成してみるOperator"

## 1. このOperatorが行うこと

![img/001.png](img/001.png)

- 補足
  - Update時にimageは変更されないので、ご注意ください。(最初に指定したイメージのままになります)

## 2. Operatorのデプロイ

### 共通手順

```sh
vim Makefile
```

32行目の`IMAGE_TAG_BASE`を任意のコンテナ・レジストリパスに変更

```sh
IMAGE_TAG_BASE ?= oracle.com/ochacafe-operator-intro
```

例：

```sh
IMAGE_TAG_BASE ?= nrt.ocir.io/orasejapan/ochacafe_sample_operator
```

以下のコマンドを実行し、Operatorのビルドとコンテナレジストリへのプッシュを実行

```sh
make docker-build docker-push
```

### OLM(Operator Lifecyle Management)を利用したデプロイ

- 前提条件
  - operator-sdkコマンドがインストールされていること
    - [こちらを参照](https://sdk.operatorframework.io/docs/installation/)
  - Kubernetesクラスタに対して`kubectl`コマンドでアクセスできること

#### 1.OLMのインストール

```sh
operator-sdk olm install
```

#### 2.バンドルイメージのビルドとプッシュ

```sh
make bundle bundle-build bundle-push
```

#### 3.デプロイ

```sh
operator-sdk run bundle <共通手順で定義したイメージのフルパス>-bundle:v0.0.1
```

例：

```sh
operator-sdk run bundle nrt.ocir.io/orasejapan/ochacafe_sample_operator-bundle:v0.0.1
```

### ダイレクトデプロイ

#### 1.デプロイ

```sh
make deploy 
```

## 3.CR(カスタム・リソース)のデプロイ

```sh
kubectl apply -f config/samples/ochacafe_v1alpha1_ochacafe.yaml
```

## 4.クリーンアップ

#### OLM(Operator Lifecyle Management)を利用した場合

```sh
operator-sdk olm uninstall
```

#### ダイレクトデプロイを利用した場合

```sh
make undeploy
```
