#! /bin/bash

# gitのルートディレクトリに移動
cd "$(git rev-parse --show-toplevel)"

# デプロイ先のサーバーを定義する
servers=("isucon12-1" "isucon12-2" "isucon12-3" "isucon12-4" "isucon12-5")

user=isucon

cd webapp/go
go build -o isuconquest || { echo 'ビルド失敗' ; exit 1 ; }
cd "$(git rev-parse --show-toplevel)"


# ローカルマシン上のWebアプリケーションのパスを定義する
app_path="./webapp"

# リモートサーバー上のWebアプリケーションのパスを定義する
remote_path="/home/isucon/webapp"

# SCPを使用して、Webアプリケーションを各サーバーにコピーする
for server in "${servers[@]}"
do
    echo サービス停止
    ssh "$user@$server" "sudo systemctl stop isuconquest.go.service"
    if [ $? -ne 0 ]; then
        echo "Error restarting web server on $server"
        continue
    fi

    echo アプリをコピーします。
    scp -r "$app_path/go" "$user@$server:$remote_path"
    if [ $? -ne 0 ]; then
        echo "Error copying files to $server"
        continue
    fi

    echo 初期化用SQLをコピーします。
    scp -r "$app_path/sql" "$user@$server:$remote_path"
    if [ $? -ne 0 ]; then
        echo "Error copying files to $server"
        continue
    fi

    echo サービス起動
    ssh "$user@$server" "sudo systemctl restart isuconquest.go.service"
    if [ $? -ne 0 ]; then
        echo "Error restarting web server on $server"
        continue
    fi

    echo "Application deployed to $server"
done

#echo プロファイラーとか起動
#./bench.sh
