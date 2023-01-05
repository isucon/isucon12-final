# 環境構築手順

git cloneしたディレクトリで動かすのではなく、本番と同様の環境構築する手順を記載します

## ansibleを用いて環境構築

### 本番の構成の補足

本番環境では、サーバーのOSは Ubuntu 22.04 LTS を使用していました。
台数は 2 vCPU, 4GB RAM のサーバーを 5 台、ベンチマーカー用に 4 vCPU, 8GB RAM のサーバーを 1 台の計 6 台となります。

### 手順
`inventory/hosts` に構築ターゲットとなるホストを記載ください（benchmarker,application）

初期状態では次のようになっています

```
[benchmarker]
127.0.0.1
[application]
127.0.0.1
```

### ベンチマーカーサーバー
次のコマンドで実行して構築します

```
ansible-playbook base.yml benchmarker.yml -i inventory/hosts
```
`/home/isucon`配下に必要なファイルが配置されます


### アプリケーションサーバー
次のコマンドで実行して構築します

```
ansible-playbook base.yml application.yml -i inventory/hosts
```

`/home/isucon`配下に必要なファイルが配置されます

## アプリケーションの起動方法

[ISUCON12本選 当日マニュアル](https://gist.github.com/shirai-suguru/770d30d16688a07ba78e0a188cd99f9f)
の内容を参照ください

たとえば、goで動作確認するには次のコマンドでアプリケーションを起動させます

```
sudo systemctl restart nginx
sudo systemctl start isuconquest.go
```

## ベンチマークのかけ方
本番と同様の環境構築した場合は、手動でベンチマーカーを起動する必要があります

ベンチマークをかける対象のホストを次のように環境変数で設定します

```
export ISUXBENCH_TARGET=127.0.0.1
```

本番と同様の設定でベンチマークをかけるには次のオプションでベンチマーカーを実行します

`/home/isucon`にて実行

```
./bin/benchmarker --stage=prod --request-timeout=10s --initialize-request-timeout=60s
```

## 1台のサーバーでベンチマーカーもアプリケーションも構築する場合

次のコマンドで実行して構築します

```
ansible-playbook base.yml benchmarker.yml application.yml -i inventory/hosts
```
