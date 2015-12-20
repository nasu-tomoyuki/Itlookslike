It looks like ...
==========

# はじめに
定期的に [Yahoo! の気象情報API](http://developer.yahoo.co.jp/webapi/map/openlocalplatform/v1/weather.html) を取得して少し先の天気をなんとなく予報します。

# 動作環境
Google App Engine を想定しています。

# API
デプロイ先のルートアクセスでフィードを返します。
/update で更新をします。

# デプロイ方法
GAE のアプリケーションキーを取得して app.yaml を書いてください。
[Yahoo! デベロッパーネットワーク](http://developer.yahoo.co.jp/webapi/map/) でアカウントを取得しアプリケーション ID を取得して config.go を書いてください。

その後、
```
goapp deploy
```
でデプロイされるはずです。10 分ごとに cron で取得に行きます。
