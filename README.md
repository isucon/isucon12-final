# isucon12-final

## ディレクトリ構成

```
.
+- webapp                 # 各言語の参考実装
+- benchmarker            # ベンチマーカー
+- docs                   # Webフロント用の静的ファイル
+- dev/extra/initial-data # 初期データ生成
+- probisioning           # セットアップ用

```

## 概要

競技中は下記2つを行うことを想定しています。

1. サーバーサイドを起動する
2. ベンチマーカーを起動する

さらにオプショナルで管理画面とゲーム本体の画面を起動できます。

## 事前準備

サーバーサイドとベンチマーカーの初期化の際に使用するマスターデータを事前にダウンロードします。(ghコマンドなしでも実行できるように修正しました)

```
$ cd dev
$ make initial-data
```

### 1. Adminの管理画面の確認方法


次のURLをブラウザでアクセスすると、Adminの管理画面のログイン画面が表示されます
（ローカルに環境を作った場合）

```
http://localhost/
```

ID: 123456
PASS: password

でログインできます

### 2. ゲーム本体の起動

ゲーム本体はUnityで開発されています。
そのためビルドするためにはUnityエディタとUnityライセンスが必要になります。
そこで本リポジトリにビルド済みの成果物が含まれており、またGitHub Pagesを通じて配信も行っています。
下記のいずれかの方法にてゲーム画面を起動できます。

#### 2.1. ローカルでの起動

リポジトリのルートでdocker runでnginxサーバーを起動します。
nginxサーバーが起動完了したら、ブラウザから `http://localhost:8888/` にアクセスします。
APIサーバーの接続先を指定して起動ボタンを押下すると、ゲーム画面が起動します。

```sh
$ docker run --name isucon12-frontend-nginx -v $(pwd)/docs:/usr/share/nginx/html:ro -p 8888:80 -d --rm nginx:stable-alpine
```

#### 2.2. GitHub Pagesを利用

[GitHub Pages](https://isucon.github.io/isucon12-final/)にアクセスします。
APIサーバーの接続先を指定して起動ボタンを押下すると、ゲーム画面が起動します。

> **Warning**
セキュリティ上の理由により、Chrome（運営の確認ではChrome 104）では、httpsなページからローカルネットワークに対するhttpアクセスがブロックされるようになりました。
ローカルで動作確認する場合は、ngrokなどのhttpsトンネリングサービスを利用するか、5.1で紹介した配信用のnginxサーバーの起動を推奨します。

### 初期データについて
https://github.com/isucon/isucon12-final/releases/tag/initial_data_20220912

こちらに初期データがあります

（事前準備の手順では、こちらのデータをダウンロードしています）

## 環境構築について

provisioningディレクトリにあるansibleを用いて構築します

- [Ansible を用いた環境構築手順](https://github.com/isucon/isucon12-final/blob/main/provisioning/packer/ansible/README.md)
- スペック: 本選当日は 2 vCPU, 4GB RAM のマシンを 5 台提供しました。別途ベンチマーカーとして 4 vCPU, 8GB RAM のマシンを用意しました。


## Links

- [ISUCON12本選 当日マニュアル](https://gist.github.com/shirai-suguru/770d30d16688a07ba78e0a188cd99f9f)
- [ISUCON12本選 アプリケーションマニュアル](https://gist.github.com/shirai-suguru/accb96c5f86200b5c16e1d2a8b533cc1)
- [ISUCON12本選 問題の解説と講評](https://isucon.net/archives/56959385.html)
