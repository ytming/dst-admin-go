# dst-admin-go
> 饥荒联机版管理后台

## 部署
注意目录必须要有读写权限。

点击查看 [部署文档](docs/install.md)

## 运行

**修改config.yml**
```
#端口
port: 8082
database: dst-db
```


运行
```
go mod tidy
go run main.go
```

## 打包


### window 打包

window 下打包 Linux 二进制

```
打开 cmd
set GOARCH=amd64
set GOOS=linux

go build
```
