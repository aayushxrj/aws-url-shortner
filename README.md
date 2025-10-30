# Commands

```
go mod init github.com/aayushxrj/aws-url-shortner
```

```
go get github.com/joho/godotenv
```

```
protoc `
  -I proto `
  --go_out=proto/gen --go_opt=paths=source_relative `
  --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative `
  proto/main.proto
```

```
go get github.com/aws/aws-sdk-go-v2@v1.23.0 github.com/aws/aws-sdk-go-v2/config@v1.18.0 github.com/aws/aws-sdk-go-v2/service/dynamodb@v1.21.0; go mod tidy
```

go get github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue
