# weave-project

Start the server using below command. We need the Github token to start the server.

```
export GITHUB_TOKEN="your_token"
make run-grpc
```

Example curl req:
Using only search:

```
grpcurl --plaintext -d '{"search_term" : "hello"}' localhost:8080  weave.api.v1.GithubSearchService.Search
```

Using search and user:
```
grpcurl --plaintext -d '{"search_term" : "test", "user" : "abh2k"}' localhost:8080  weave.api.v1.GithubSearchService.Search
```