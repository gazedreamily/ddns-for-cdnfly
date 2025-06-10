# 简易Cdnfly客户端

主要用于解决部分cdnfly系统dns服务器失效，填写后端域名报错的问题。

## 用法

将可执行文件上传到服务器，然后执行

```bash
./cdnfly-ddns
```

### 可选参数

```bash
-c # 配置文件路径，默认为当前目录下的 config.json
```

### 配置文件示例

API-KEY和API-SECRET可以在Cdnfly的控制台中获取
```text
{
  "api": "https://api.your_service.com", // Cdnfly的基础URL
  "api_key": "your_api_key_here", // API-KEY
  "api_secret": "your_api_secret_here", // API-SECRET
  "site_domain": "your_site.com", // 你的站点域名
  "ip_set_count": 1 // IP地址数量，默认为1，如果服务器本身有多个IP地址，可以设置为2或更多
}
```
