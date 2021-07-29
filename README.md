# GAE/Go Compatibility Check

Test

```console
go test ./...
```

Deploy (Go 1.11)

```console
gcloud app deploy --project ${YOUR_PROJECT} app_go111.yaml
gcloud app deploy --project ${YOUR_PROJECT} queue.yaml
```

Deploy (Go 1.15)

```console
gcloud app deploy --project ${YOUR_PROJECT} app_go115.yaml
gcloud app deploy --project ${YOUR_PROJECT} queue.yaml
```
