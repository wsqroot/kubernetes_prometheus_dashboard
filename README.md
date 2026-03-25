# kubernetes_prometheus_dashboard

当前版本提供以下能力：

1. 创建和管理常见 Kubernetes 资源对象
2. 管理多个 Kubernetes 集群资源

## Docker 运行

1. 准备后端配置文件

```bash
cp login/configs/config.example.yaml login/configs/config.yaml
```

2. 按实际环境修改 `login/configs/config.yaml` 中的数据库和集群配置

3. 启动服务

```bash
docker compose up --build -d
```

4. 访问地址

- 前端: `http://localhost:8000`
- 后端: `http://localhost:8080`

## 说明

- 根目录 Windows 启动脚本仅用于本地机器调试，不再纳入 Git 仓库
- Docker 运行时，后端容器内会自动安装 `kubectl`
