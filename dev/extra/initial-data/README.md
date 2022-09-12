# 初期データ生成マニュアル

## 動作確認用のデータ生成方法

次のコマンドで、必要なファイルを生成し、指定のディレクトリへファイルを配置します

```
make build && make install
```

## 本番と同等のデータの生成

かなり時間がかかりますが、ローカルで再作成できます
（通常は、GitHubのReleaseページから取得してください）

```
make build-production && make install
```


