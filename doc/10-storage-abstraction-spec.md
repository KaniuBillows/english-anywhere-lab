# English Anywhere Lab - 文件存储抽象规范（Local FS / S3）

## 1. 目标
同一套业务代码支持两类文件存储后端：
- 本地文件系统（默认，低成本）
- S3 协议对象存储（可配置切换）

无需数据迁移能力（按你的要求）。

## 2. 适用文件类型
- 口语录音（webm/m4a）
- 生成音频（mp3）
- 词卡插图（png/jpg/webp）
- 导出报告（pdf/json）

## 3. 统一接口（Go）
```go
type ObjectStore interface {
    Put(ctx context.Context, req PutRequest) (PutResult, error)
    GetPresignedURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
    Delete(ctx context.Context, objectKey string) error
    Stat(ctx context.Context, objectKey string) (ObjectMeta, error)
}

type PutRequest struct {
    ObjectKey   string
    ContentType string
    SizeBytes   int64
    Reader      io.Reader
    Metadata    map[string]string
}
```

## 4. 对象键命名规范
- 音频：`user/{user_id}/audio/{yyyy}/{mm}/{uuid}.webm`
- 图片：`pack/{pack_id}/image/{uuid}.webp`
- 导出：`user/{user_id}/export/{yyyy}/{mm}/{uuid}.pdf`

要求：
- 不使用原始文件名作为 key（避免注入与冲突）
- 所有 key 小写

## 5. Provider 配置

### 5.1 Local FS
- `FILES_PROVIDER=local`
- `FILES_LOCAL_ROOT=./data/files`
- URL 映射：`/static/files/{object_key}`（由 API 服务只读暴露）

### 5.2 S3 Compatible
- `FILES_PROVIDER=s3`
- `FILES_S3_ENDPOINT`
- `FILES_S3_REGION`
- `FILES_S3_BUCKET`
- `FILES_S3_ACCESS_KEY`
- `FILES_S3_SECRET_KEY`
- `FILES_S3_FORCE_PATH_STYLE=true|false`
- `FILES_S3_PUBLIC_BASE_URL`（可选）

## 6. 上传与下载规则
- 最大单文件大小：
  - 音频：20MB
  - 图片：5MB
  - 其他：10MB
- Content-Type 白名单校验。
- 下载默认使用预签名 URL（S3）或受控静态路由（Local）。

## 7. 安全规范
- 禁止可执行脚本类型上传（`.exe`, `.sh`, `.js`）。
- 上传后写入数据库元数据，删除逻辑先删 DB 再删对象（失败可重试补偿）。
- 所有文件路径必须经过 `path.Clean` 和前缀检查。

## 8. 失败补偿
- `Put` 成功但 DB 失败：写补偿任务删除孤儿对象。
- DB 成功但 `Delete` 失败：写重试队列，后台清理。

## 9. 监控指标
- `storage_put_success_rate`
- `storage_put_p95_ms`
- `storage_get_url_p95_ms`
- `storage_orphan_objects_count`

## 10. 验收标准
- 在同一业务代码下，切换 provider 仅依赖配置文件。
- Local 与 S3 两模式均可通过上传/读取/删除回归测试。
