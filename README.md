# vrc-daily-uploader

Flickrにアップロードしている写真からランダムに3枚取得し, S3経由でCloudFront配信するツールです.   
GitHub Actionsで毎日自動実行され, VRChat Worldに日替わりで写真を公開する用途を想定しています.

## Flickr APIの利用について

- 取得対象は運営者自身のFlickrアカウントの公開写真のみです.
- APIへのアクセスは1日1回, GitHub Actionsの定時実行（[.github/workflows/worker.yml](.github/workflows/worker.yml)）でのみ発生します.
- リクエストの`User-Agent`にはツール名とこのリポジトリへのリンクを含めており, 利用状況について確認したい場合はリポジトリのIssueからご連絡いただけます.
