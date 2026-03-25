# Login Service

Go 后端登录服务，用户名和密码从 `192.168.10.201` 上的 MySQL 读取。

## 目录结构

- `cmd/server`: 程序入口
- `configs/config.yaml`: 独立配置文件
- `internal/config`: 配置加载
- `internal/database`: MySQL 连接
- `internal/repository`: 用户数据访问
- `internal/service`: 登录校验
- `internal/handler`: HTTP 接口

## 启动

```powershell
cd C:\Users\Dell\Desktop\developer-kubernetes\login
go run .\cmd\server
```

或指定配置文件：

```powershell
$env:LOGIN_CONFIG='configs/config.yaml'
go run .\cmd\server
```

## 接口

- `GET /healthz`
- `POST /api/login`

请求体：

```json
{
  "username": "admin",
  "password": "Admin@123"
}
```

## 表结构假设

默认 SQL 在 `configs/config.yaml` 中配置，当前假设：

- `kubernetes_user.id`
- `kubernetes_user.username`
- `kubernetes_passwd.user_id`
- `kubernetes_passwd.password`

如果你的真实字段名不同，直接修改配置文件里的 `auth.user_lookup_sql` 即可，无需改 Go 代码。
