# GAE/Go Compatibility Check

Test

```console
go test ./...
```

Test ([act](https://github.com/nektos/act))

```console
act -j test
```

Deploy (Go 1.11)

```console
gcloud app deploy --project ${YOUR_PROJECT} app_go111.yaml
gcloud app deploy --project ${YOUR_PROJECT} queue.yaml
```

Deploy (Go 1.15)

```console
gcloud beta app deploy --project ${YOUR_PROJECT} app_go115.yaml
gcloud app deploy --project ${YOUR_PROJECT} queue.yaml
```

Deploy (Go 1.16)

```console
gcloud beta app deploy --project ${YOUR_PROJECT} app_go116.yaml
gcloud app deploy --project ${YOUR_PROJECT} queue.yaml
```

Japanese article
[Go 1.15 で GAE 独自 API を利用できるのか？](https://qiita.com/sg0hsmt/items/341265b485bbc7ccef28)
