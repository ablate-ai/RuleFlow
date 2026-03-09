FROM golang:1.24-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -o ruleflow ./cmd/ruleflow

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/ruleflow .
COPY --from=builder /app/web ./web
COPY --from=builder /app/rules ./rules

# 暴露端口
EXPOSE 8080

# 运行
CMD ["./ruleflow"]
