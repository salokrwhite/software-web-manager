import { Layout, Menu, Typography, Card, Space, Tag, Button, message } from 'antd'
import {
  CodeOutlined,
  ApiOutlined,
  FileTextOutlined,
  CopyOutlined,
  RocketOutlined
} from '@ant-design/icons'
import { useMemo, useState, type CSSProperties } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

const { Title, Text, Paragraph } = Typography
const { Content } = Layout

const codeExamples: Record<string, Record<string, string>> = {
  go: {
    install: `go get software-web-manager/sdk-go`,
    init: `package main

import (
    "context"
    "fmt"
    "log"

    swmsdk "software-web-manager/sdk-go"
)

func main() {
    client := swmsdk.New("https://dashscope-internal-swmapi.anteasy.com", "your-app-id", "your-app-secret")
    client.Channel = "stable"
    client.Platform = "windows"
    client.Arch = "amd64"
    client.DeviceID = "device-unique-id"
    client.UserID = "user-1001"
    client.Attributes = map[string]interface{}{
        "os_version": "Windows 11",
    }

    resp, err := client.CheckUpdate(context.Background(), "1.0.0", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("update_available:", resp.UpdateAvailable)
}`,
    checkUpdate: `resp, err := client.CheckUpdate(context.Background(), "1.0.0", nil)
if err != nil {
    log.Fatal(err)
}

if resp.UpdateAvailable {
    fmt.Printf("发现新版本: %s\n", resp.Version)
    fmt.Printf("下载地址: %s\n", resp.DownloadURL)
    fmt.Printf("校验值: %s\n", resp.ChecksumSHA256)
    fmt.Printf("签名: %s\n", resp.Signature)
    fmt.Printf("强制更新: %v\n", resp.Mandatory)
}`,
    download: `err := client.Download(
    context.Background(),
    resp.DownloadURL,
    "/path/to/download/file.zip",
    resp.ChecksumSHA256,
    resp.Signature,
    func(written, total int64) {
        progress := float64(written) / float64(total) * 100
        fmt.Printf("下载进度: %.2f%%\n", progress)
    },
)`,
    reportEvent: `err := client.ReportEvent(context.Background(), "app_launch", map[string]interface{}{
    "os_version": "Windows 11",
    "screen_resolution": "1920x1080",
})`,
    reportHeartbeat: `err := client.ReportHeartbeat(context.Background(), "1.0.0")`,
    reportFeedback: `rating := 5
err := client.ReportFeedback(context.Background(), "反馈内容", &rating, "user@example.com", []string{
    "/path/to/screenshot.png",
}, map[string]interface{}{
    "app_version": "1.0.0",
    "os": "Windows 11",
    "locale": "zh-CN",
})
if errors.Is(err, swmsdk.ErrFeedbackDisabled) {
    fmt.Println("服务端已关闭用户反馈")
}`
  },
  java: {
    install: `<dependency>
    <groupId>com.swm</groupId>
    <artifactId>swm-sdk</artifactId>
    <version>0.1.0</version>
</dependency>`,
    init: `import com.swm.sdk.Client;

public class Main {
    public static void main(String[] args) {
        Client client = new Client("https://dashscope-internal-swmapi.anteasy.com", "your-app-id", "your-app-secret");
        client.setChannel("stable");
        client.setPlatform("windows");
        client.setArch("x64");
        client.setDeviceId("device-unique-id");
    }
}`,
    checkUpdate: `UpdateCheckResponse resp = client.checkUpdate("1.0.0", null);

if (resp.update_available) {
    System.out.println("发现新版本: " + resp.version);
    System.out.println("下载地址: " + resp.download_url);
    System.out.println("强制更新: " + resp.mandatory);
}`,
    download: `client.download(
    resp.download_url,
    Paths.get("/path/to/download/file.zip"),
    resp.checksum_sha256
);`,
    reportEvent: `Map<String, Object> props = new HashMap<>();
props.put("os_version", "Windows 11");
props.put("screen_resolution", "1920x1080");

client.reportEvent("app_launch", props);`,
    reportHeartbeat: `client.reportHeartbeat("1.0.0");`,
    reportFeedback: `client.reportFeedback(
    "反馈内容",
    5,
    "user@example.com",
    List.of(Paths.get("/path/to/screenshot.png")),
    Map.of("app_version", "1.0.0", "os", "Windows 11")
);`
  },
  python: {
    install: `pip install swm-sdk`,
    init: `from swm_sdk import Client

client = Client(
    base_url="https://dashscope-internal-swmapi.anteasy.com",
    app_id="your-app-id",
    app_secret="your-app-secret",
    timeout=30
)
client.channel = "stable"
client.platform = "windows"
client.arch = "x64"
client.device_id = "device-unique-id"`,
    checkUpdate: `resp = client.check_update("1.0.0")

if resp.update_available:
    print(f"发现新版本: {resp.version}")
    print(f"下载地址: {resp.download_url}")
    print(f"强制更新: {resp.mandatory}")`,
    download: `client.download(
    url=resp.download_url,
    dest_path="/path/to/download/file.zip",
    checksum_sha256=resp.checksum_sha256,
    chunk_size=32768
)`,
    reportEvent: `client.report_event("app_launch", {
    "os_version": "Windows 11",
    "screen_resolution": "1920x1080"
})`,
    reportHeartbeat: `client.report_heartbeat(app_version="1.0.0")`,
    reportFeedback: `client.report_feedback(
    "反馈内容",
    rating=5,
    contact="user@example.com",
    attachments=["/path/to/screenshot.png"],
    metadata={"app_version": "1.0.0", "os": "Windows 11"}
)`
  },
  node: {
    install: `npm install swm-sdk`,
    init: `import { Client } from 'swm-sdk';

const client = new Client('https://dashscope-internal-swmapi.anteasy.com', 'your-app-id', 'your-app-secret');
client.channel = 'stable';
client.platform = 'windows';
client.arch = 'x64';
client.deviceId = 'device-unique-id';`,
    checkUpdate: `const resp = await client.checkUpdate('1.0.0');

if (resp.updateAvailable) {
    console.log('发现新版本:', resp.version);
    console.log('下载地址:', resp.downloadURL);
    console.log('强制更新:', resp.mandatory);
}`,
    download: `await client.download(
    resp.downloadURL,
    '/path/to/download/file.zip',
    resp.checksumSHA256,
    (written, total) => {
        const progress = (written / total * 100).toFixed(2);
        console.log('下载进度:', progress + '%');
    }
);`,
    reportEvent: `await client.reportEvent('app_launch', {
    os_version: 'Windows 11',
    screen_resolution: '1920x1080'
});`,
    reportHeartbeat: `await client.reportHeartbeat('1.0.0');`,
    reportFeedback: `await client.reportFeedback('反馈内容', {
    rating: 5,
    contact: 'user@example.com',
    attachments: ['/path/to/screenshot.png'],
    metadata: { app_version: '1.0.0', os: 'Windows 11' }
});`
  },
  cpp: {
    install: `# 使用 vcpkg
vcpkg install swm-sdk

# 或使用 CMake FetchContent
include(FetchContent)
FetchContent_Declare(
    swm-sdk
    GIT_REPOSITORY https://github.com/yourorg/swm-sdk-cpp.git
    GIT_TAG v0.1.0
)`,
    init: `#include <swm_sdk/client.hpp>

int main() {
    swm::Client client("https://dashscope-internal-swmapi.anteasy.com", "your-app-id", "your-app-secret");
    client.channel = "stable";
    client.platform = "windows";
    client.arch = "x64";
    client.device_id = "device-unique-id";
}`,
    checkUpdate: `auto resp = client.check_update("1.0.0");

if (resp.update_available) {
    std::cout << "发现新版本: " << resp.version << std::endl;
    std::cout << "下载地址: " << resp.download_url << std::endl;
    std::cout << "强制更新: " << resp.mandatory << std::endl;
}`,
    download: `client.download(
    resp.download_url,
    "/path/to/download/file.zip",
    resp.checksum_sha256
);`,
    reportEvent: `client.report_event("app_launch", {
    {"os_version", "Windows 11"},
    {"screen_resolution", "1920x1080"}
});`,
    reportHeartbeat: `client.report_heartbeat("1.0.0");`,
    reportFeedback: `client.report_feedback(
    "反馈内容",
    5,
    "user@example.com",
    {"/path/to/screenshot.png"},
    {{"app_version", "1.0.0"}, {"os", "Windows 11"}}
);`
  },
  csharp: {
    install: `dotnet add reference ./sdk/csharp/SwmSdk.csproj
# 或（已发布到 NuGet 时）
# dotnet add package SwmSdk`,
    init: `using SwmSdk;

var client = new Client("https://dashscope-internal-swmapi.anteasy.com", "your-app-id", "your-app-secret");
client.Channel = "stable";
client.Platform = "windows";
client.Arch = "amd64";
client.DeviceId = "device-unique-id";`,
    checkUpdate: `UpdateCheckResponse resp;
try
{
    resp = await client.CheckUpdateAsync("1.0.0", versionCode: 100, userId: "user-1001");
}
catch (SwmDeviceBlockedException)
{
    Console.WriteLine("设备已被封禁");
    return;
}
catch (SwmUpdateRegionBlockedException)
{
    Console.WriteLine("当前地区不允许更新");
    return;
}

await client.ReportEventAsync("check_update", new Dictionary<string, object> {
    ["version"] = "1.0.0",
    ["update_available"] = resp.UpdateAvailable,
    ["release_id"] = resp.ReleaseId ?? ""
});

if (resp.UpdateAvailable) {
    Console.WriteLine($"发现新版本: {resp.Version}");
    Console.WriteLine($"下载地址: {resp.DownloadUrl}");
    Console.WriteLine($"强制更新: {resp.Mandatory}");
}`,
    download: `await client.DownloadAsync(
    resp.DownloadUrl!,
    "/path/to/download/file.zip",
    resp.ChecksumSha256,
    resp.Signature,
    (written, total) => {
        var progress = (double)written / total * 100;
        Console.WriteLine($"下载进度: {progress:F2}%");
    }
);`,
    reportEvent: `await client.ReportEventAsync("app_started", new Dictionary<string, object?> {
    ["version"] = "1.0.0",
    ["platform"] = "windows",
    ["arch"] = "amd64"
});`,
    reportHeartbeat: `await client.ReportHeartbeatAsync("1.0.0");`,
    reportFeedback: `await client.ReportFeedbackAsync(
    "反馈内容",
    5,
    "user@example.com",
    new[] { "/path/to/screenshot.png" },
    new Dictionary<string, object?> {
        ["app_version"] = "1.0.0",
        ["os"] = "Windows 11",
        ["locale"] = "zh-CN"
    }
);`
  },
  rust: {
    install: `cargo add swm-sdk`,
    init: `use swm_sdk::Client;

let client = Client::new("https://dashscope-internal-swmapi.anteasy.com", "your-app-id", "your-app-secret");
client.channel = "stable".to_string();
client.platform = "windows".to_string();
client.arch = "x64".to_string();
client.device_id = "device-unique-id".to_string();`,
    checkUpdate: `let resp = client.check_update("1.0.0", None)?;

if resp.update_available {
    println!("发现新版本: {}", resp.version);
    println!("下载地址: {}", resp.download_url);
    println!("强制更新: {}", resp.mandatory);
}`,
    download: `client.download(
    &resp.download_url,
    Path::new("/path/to/download/file.zip"),
    Some(&resp.checksum_sha256)
)?;`,
    reportEvent: `client.report_event(
    "app_launch",
    serde_json::json!({
        "os_version": "Windows 11",
        "screen_resolution": "1920x1080"
    })
)?;`,
    reportHeartbeat: `client.report_heartbeat(Some("1.0.0".to_string()), None)?;`,
    reportFeedback: `client.report_feedback(
    "反馈内容",
    Some(5),
    Some("user@example.com"),
    vec![Path::new("/path/to/screenshot.png")],
    Some(serde_json::json!({"app_version": "1.0.0", "os": "Windows 11"}))
)?;`
  }
}

type SupportedLang = 'go' | 'csharp'

type DocParam = {
  name: string
  type: string
  required: string
  description: string
}

type DocField = {
  name: string
  type: string
  description: string
}

type MethodDoc = {
  name: string
  signature: string
  endpoint: string
  description: string
  params: DocParam[]
  returns: DocField[]
  notes?: string[]
  example: string
}

type ManagementDoc = {
  method: string
  endpoint: string
  description: string
}

type SdkReferenceDoc = {
  languageName: string
  packageName: string
  install: string
  init: string
  fullFlow: string
  runtimeFields: DocField[]
  updateResponse: DocField[]
  streamOptions: DocField[]
  methods: MethodDoc[]
  errors: DocField[]
  managementAuthNote: string
  managementMethods: ManagementDoc[]
}

const languages: Array<{ key: SupportedLang; label: string; color: string }> = [
  { key: 'go', label: 'Go', color: '#00ADD8' },
  { key: 'csharp', label: 'C#', color: '#239120' },
]

const docTableStyle: CSSProperties = { width: '100%', borderCollapse: 'collapse' }
const docThStyle: CSSProperties = {
  padding: '12px',
  textAlign: 'left',
  border: '1px solid #e8e8e8',
  background: '#f5f5f5',
  verticalAlign: 'top'
}
const docTdStyle: CSSProperties = { padding: '12px', border: '1px solid #e8e8e8', verticalAlign: 'top' }

function DocTable({ headers, rows }: { headers: string[]; rows: string[][] }) {
  return (
    <table style={docTableStyle}>
      <thead>
        <tr>
          {headers.map((header) => (
            <th key={header} style={docThStyle}>{header}</th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.map((row, index) => (
          <tr key={`${row[0]}-${index}`}>
            {row.map((cell, cellIndex) => (
              <td key={`${row[0]}-${cellIndex}`} style={docTdStyle}>{cell}</td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  )
}

const sdkReferenceDocs: Record<SupportedLang, SdkReferenceDoc> = {
  go: {
    languageName: 'Go',
    packageName: 'software-web-manager/sdk-go',
    install: codeExamples.go.install,
    init: codeExamples.go.init,
    fullFlow: `ctx := context.Background()
client := swmsdk.New("https://dashscope-internal-swmapi.anteasy.com", "your-app-id", "your-app-secret")
client.Channel = "stable"
client.Platform = "windows"
client.Arch = "amd64"
client.DeviceID = "device-001"
client.VerifySignature = true
client.PublicKey = "base64_or_hex_public_key"

resp, err := client.CheckUpdate(ctx, "1.0.0", nil)
if err != nil {
    log.Fatal(err)
}
if resp.UpdateAvailable {
    err = client.Download(ctx, resp.DownloadURL, "./downloads/app.zip", resp.ChecksumSHA256, resp.Signature, nil)
    if err != nil {
        log.Fatal(err)
    }
}
_ = client.ReportHeartbeat(ctx, "1.0.0")`,
    runtimeFields: [
      { name: 'BaseURL/AppID/AppSecret', type: 'string', description: '由 New(baseURL, appID, appSecret) 初始化，三者均为签名所需。' },
      { name: 'Channel/Platform/Arch/DeviceID', type: 'string', description: '客户端上下文字段；更新检查、事件、心跳、反馈、流式更新均会使用。' },
      { name: 'UserID', type: 'string', description: '可选用户 ID。' },
      { name: 'Attributes', type: 'map[string]interface{}', description: '可选扩展属性，会并入请求体。' },
      { name: 'VerifySignature/PublicKey', type: 'bool/string', description: '开启后会对 checksum 与 signature 做 Ed25519 校验。' },
      { name: 'Retries/Backoff', type: 'int/time.Duration', description: '请求重试与退避参数，默认 2 次、500ms。' },
      { name: 'HTTPClient', type: '*http.Client', description: '可替换默认客户端（默认超时 30s）。' },
      { name: 'AuthToken', type: 'string', description: '管理端 API 鉴权 token，通过 SetAuthToken 设置。' }
    ],
    updateResponse: [
      { name: 'update_available/mandatory/version/release_id', type: 'bool/string', description: '更新可用性与目标版本基础信息。' },
      { name: 'download_url/checksum_sha256/signature/size', type: 'string/int64', description: '下载与校验字段。' },
      { name: 'notes/delivery_method/open_in_browser', type: 'string/bool', description: '展示与投递策略字段。' },
      { name: 'heartbeat_interval_seconds/rollback_allowed', type: 'int/bool', description: '心跳建议与回滚能力。' }
    ],
    streamOptions: [
      { name: 'ChannelCode/Platform/Arch/DeviceID', type: 'string', description: '必需字段；为空时尝试从 Client 对应字段回填。' },
      { name: 'CurrentVersion/VersionCode', type: 'string/*int', description: 'WatchUpdates 内二次调用 CheckUpdate 时使用。' },
      { name: 'Reconnect/ReconnectBackoff/ReconnectMaxBackoff/Jitter', type: 'bool/time.Duration', description: '重连行为控制（默认自动重连+抖动）。' },
      { name: 'OnError/OnControlEvent', type: 'func', description: '错误和控制事件回调。' }
    ],
    methods: [
      {
        name: 'CheckUpdate',
        signature: 'CheckUpdate(ctx context.Context, currentVersion string, versionCode *int) (UpdateCheckResponse, error)',
        endpoint: 'POST /api/client/update-check',
        description: '版本检查入口。请求体自动包含 Client 的 channel/platform/arch/device_id/user_id/attributes。',
        params: [
          { name: 'ctx', type: 'context.Context', required: '是', description: '请求上下文。' },
          { name: 'currentVersion', type: 'string', required: '是', description: '客户端当前版本。' },
          { name: 'versionCode', type: '*int', required: '否', description: '可选版本号 code。' }
        ],
        returns: [
          { name: 'UpdateCheckResponse', type: 'struct', description: '包含下载地址、签名、强更标记等字段。' },
          { name: 'error', type: 'error', description: '可能返回 ErrDeviceBlocked / ErrUpdateRegionBlocked。' }
        ],
        notes: ['当 VerifySignature=true 且响应携带 signature+checksum_sha256 时，会自动验签。'],
        example: codeExamples.go.checkUpdate
      },
      {
        name: 'Download',
        signature: 'Download(ctx context.Context, url, destPath, checksum, signature string, progress func(written, total int64)) error',
        endpoint: 'GET <download_url>',
        description: '下载更新文件并执行校验；destPath 目录不存在会自动创建。',
        params: [
          { name: 'url/destPath', type: 'string', required: '是', description: '下载地址与目标文件路径。' },
          { name: 'checksum', type: 'string', required: '否', description: 'SHA256；为空则跳过校验。' },
          { name: 'signature', type: 'string', required: '否', description: '签名值，配合 VerifySignature 使用。' },
          { name: 'progress', type: 'func(int64,int64)', required: '否', description: '下载进度回调。' }
        ],
        returns: [{ name: 'error', type: 'error', description: '下载失败或校验失败。' }],
        example: codeExamples.go.download
      },
      {
        name: 'ReportEvent / ReportEvents',
        signature: 'ReportEvent(ctx, eventName, props) / ReportEvents(ctx, events)',
        endpoint: 'POST /api/client/events',
        description: '单条与批量事件上报。批量模式用于本地缓存补偿。',
        params: [
          { name: 'eventName', type: 'string', required: '是(单条)', description: '事件名称。' },
          { name: 'props', type: 'map[string]interface{}', required: '否', description: '单条事件属性。' },
          { name: 'events', type: '[]Event', required: '是(批量)', description: '批量事件数组。' }
        ],
        returns: [{ name: 'error', type: 'error', description: '请求失败返回错误。' }],
        example: codeExamples.go.reportEvent
      },
      {
        name: 'ReportHeartbeat',
        signature: 'ReportHeartbeat(ctx context.Context, appVersion string) error',
        endpoint: 'POST /api/client/heartbeat',
        description: '设备在线心跳。DeviceID 不能为空，否则返回 device_id required。',
        params: [
          { name: 'appVersion', type: 'string', required: '否', description: '客户端版本号。' },
          { name: 'c.DeviceID', type: 'string', required: '是', description: '设备唯一 ID。' }
        ],
        returns: [{ name: 'error', type: 'error', description: '请求失败返回错误。' }],
        example: codeExamples.go.reportHeartbeat
      },
      {
        name: 'ReportFeedback',
        signature: 'ReportFeedback(ctx, content, rating, contact, attachments, metadata) error',
        endpoint: 'POST /api/client/feedback (multipart/form-data)',
        description: '反馈上报，支持评分、联系方式、附件、metadata。',
        params: [
          { name: 'content', type: 'string', required: '是', description: '反馈内容。' },
          { name: 'rating', type: '*int', required: '否', description: '评分。' },
          { name: 'contact', type: 'string', required: '否', description: '联系方式。' },
          { name: 'attachments', type: '[]string', required: '否', description: '本地文件路径列表。' },
          { name: 'metadata', type: 'map[string]interface{}', required: '否', description: '业务元数据。' }
        ],
        returns: [{ name: 'error', type: 'error', description: '参数、文件读取或请求失败。' }],
        example: codeExamples.go.reportFeedback
      },
      {
        name: 'StartUpdateStream',
        signature: 'StartUpdateStream(ctx context.Context, options UpdateStreamOptions, onEvent func(UpdatePushEvent)) (*UpdateWatchHandle, error)',
        endpoint: 'GET /api/client/updates/stream (SSE)',
        description: '建立更新推送流；handle.Stop() 主动关闭。',
        params: [
          { name: 'options', type: 'UpdateStreamOptions', required: '是', description: '流选项，包含重连和回调配置。' },
          { name: 'onEvent', type: 'func(UpdatePushEvent)', required: '是', description: '收到推送事件时触发。' }
        ],
        returns: [
          { name: '*UpdateWatchHandle', type: 'handle', description: '调用 Stop() 结束监听。' },
          { name: 'error', type: 'error', description: '参数不完整或连接初始化失败。' }
        ],
        example: `handle, err := client.StartUpdateStream(context.Background(), swmsdk.UpdateStreamOptions{
    CurrentVersion: "1.0.0",
}, func(evt swmsdk.UpdatePushEvent) {
    fmt.Println(evt.EventType, evt.ReleaseID)
})
defer handle.Stop()`
      },
      {
        name: 'WatchUpdates',
        signature: 'WatchUpdates(ctx context.Context, options UpdateStreamOptions, onUpdateAvailable func(UpdateCheckResponse)) (*UpdateWatchHandle, error)',
        endpoint: 'SSE + update-check',
        description: '收到推送后自动触发 CheckUpdate，发现更新才回调。',
        params: [
          { name: 'options.CurrentVersion', type: 'string', required: '建议必填', description: '二次检查使用的当前版本。' },
          { name: 'onUpdateAvailable', type: 'func(UpdateCheckResponse)', required: '是', description: '有更新时回调。' }
        ],
        returns: [{ name: '*UpdateWatchHandle/error', type: 'handle/error', description: '监听控制句柄与错误。' }],
        example: `handle, _ := client.WatchUpdates(context.Background(), swmsdk.UpdateStreamOptions{
    CurrentVersion: "1.0.0",
}, func(resp swmsdk.UpdateCheckResponse) {
    fmt.Println("new version:", resp.Version)
})`
      },
      {
        name: 'SetAuthToken',
        signature: 'SetAuthToken(token string)',
        endpoint: '管理端 API 共用',
        description: '设置 Bearer Token（管理端 API 必须先设置）。',
        params: [{ name: 'token', type: 'string', required: '是', description: '后台登录返回的 access token。' }],
        returns: [{ name: '无', type: '-', description: '仅更新 Client 状态。' }],
        example: `client.SetAuthToken(os.Getenv("SWM_ACCESS_TOKEN"))`
      }
    ],
    errors: [
      { name: 'ErrDeviceBlocked', type: 'error', description: 'error.code = device_blocked。' },
      { name: 'ErrUpdateRegionBlocked', type: 'error', description: 'error.code = update_region_blocked。' },
      { name: '其他 error', type: 'error', description: '网络错误、状态码错误、验签失败等。' }
    ],
    managementAuthNote: '调用管理端 API 前先执行 client.SetAuthToken(token)。',
    managementMethods: [
      { method: 'GetApp / UpdateApp', endpoint: 'GET/PATCH /api/apps/{appID}', description: '应用信息查询与更新。' },
      { method: 'ListChannels / CreateChannel', endpoint: 'GET/POST /api/apps/{appID}/channels', description: '渠道管理。' },
      { method: 'ListAppMembers / AddAppMember', endpoint: 'GET/POST /api/apps/{appID}/members', description: '成员管理。' },
      { method: 'List/Create/Update/Delete Release', endpoint: '/api/apps/{appID}/releases /api/releases/{releaseID}', description: '发布基础生命周期。' },
      { method: 'Submit/Approve/Reject/Publish/Rollback/Revoke Release', endpoint: 'POST /api/releases/{releaseID}/...', description: '发布审核与通道操作。' },
      { method: 'UploadArtifact / ListArtifacts / GetArtifactDownloadURL', endpoint: '/api/releases/{releaseID}/artifacts /api/artifacts/{artifactID}/download', description: '制品上传与下载地址。' },
      { method: 'Region/Geo/Template/AppSecret 系列', endpoint: '/api/apps/{id}/region-rules /api/geo/* /api/release-templates /api/apps/{id}/app-secrets', description: '区域规则、地理解析、模板与密钥管理。' }
    ]
  },
  csharp: {
    languageName: 'C#',
    packageName: 'SwmSdk',
    install: codeExamples.csharp.install,
    init: codeExamples.csharp.init,
    fullFlow: `var client = new Client("https://dashscope-internal-swmapi.anteasy.com", "your-app-id", "your-app-secret")
{
    Channel = "stable",
    Platform = "windows",
    Arch = "amd64",
    DeviceId = "device-001",
    VerifySignature = true,
    PublicKey = "base64_or_hex_public_key"
};

var resp = await client.CheckUpdateAsync("1.0.0", versionCode: 100);
if (resp.UpdateAvailable && !string.IsNullOrWhiteSpace(resp.DownloadUrl))
{
    await client.DownloadAsync(resp.DownloadUrl!, "./downloads/app.zip", resp.ChecksumSha256, resp.Signature);
}
await client.ReportHeartbeatAsync("1.0.0");`,
    runtimeFields: [
      { name: 'BaseUrl/AppId/AppSecret', type: 'string', description: '由构造函数传入，签名必需。' },
      { name: 'Channel/Platform/Arch/DeviceId', type: 'string', description: '客户端上下文。' },
      { name: 'UserId', type: 'string', description: '默认用户 ID，可被方法参数覆盖。' },
      { name: 'Attributes', type: 'Dictionary<string, object?>', description: '扩展属性。' },
      { name: 'VerifySignature/PublicKey/SignatureVerifier', type: 'bool/string/delegate', description: '验签开关、公钥和自定义验签实现。' },
      { name: 'Retries/Backoff', type: 'int/TimeSpan', description: '重试参数，默认 2 次 + 500ms。' },
      { name: 'HttpClient', type: 'HttpClient', description: '可注入外部 HttpClient。' },
      { name: 'AuthToken', type: 'string', description: '管理端 Bearer Token。' }
    ],
    updateResponse: [
      { name: 'UpdateAvailable/Mandatory/Version/ReleaseId', type: 'bool/string?', description: '更新判定与目标版本信息。' },
      { name: 'DownloadUrl/ChecksumSha256/Signature/Size', type: 'string?/long', description: '下载与校验字段。' },
      { name: 'HeartbeatIntervalSeconds/RollbackAllowed', type: 'int/bool', description: '心跳建议与回滚能力。' },
      { name: 'Notes/DeliveryMethod/OpenInBrowser/ReleaseNotesUrl', type: 'string?/bool', description: '展示和投递相关字段。' }
    ],
    streamOptions: [
      { name: 'ChannelCode/Platform/Arch/DeviceId', type: 'string?', description: '为空时回退到 Client 对应字段。' },
      { name: 'CurrentVersion/VersionCode', type: 'string?/int?', description: 'WatchUpdates 二次检查参数。' },
      { name: 'Reconnect/Backoff/MaxBackoff/Jitter', type: 'bool/TimeSpan', description: '重连与退避控制。' },
      { name: 'OnError/OnControlEvent', type: 'Action', description: '错误与控制事件回调。' }
    ],
    methods: [
      {
        name: 'Client(...)',
        signature: 'Client(string baseUrl, string appId, string appSecret, HttpClient? httpClient = null)',
        endpoint: '客户端+管理端通用',
        description: '创建 SDK 客户端实例。',
        params: [
          { name: 'baseUrl', type: 'string', required: '是', description: 'API 根地址。' },
          { name: 'appId/appSecret', type: 'string', required: '是', description: '应用身份与密钥。' },
          { name: 'httpClient', type: 'HttpClient?', required: '否', description: '可选外部注入。' }
        ],
        returns: [{ name: 'Client', type: 'class', description: '可复用实例。' }],
        example: codeExamples.csharp.init
      },
      {
        name: 'CheckUpdateAsync',
        signature: 'Task<UpdateCheckResponse> CheckUpdateAsync(string currentVersion, int? versionCode = null, string? userId = null, CancellationToken cancellationToken = default)',
        endpoint: 'POST /api/client/update-check',
        description: '执行更新检查。',
        params: [
          { name: 'currentVersion', type: 'string', required: '是', description: '当前版本。' },
          { name: 'versionCode', type: 'int?', required: '否', description: '可选数值版本号。' },
          { name: 'userId', type: 'string?', required: '否', description: '覆盖 client.UserId。' },
          { name: 'cancellationToken', type: 'CancellationToken', required: '否', description: '取消令牌。' }
        ],
        returns: [{ name: 'UpdateCheckResponse', type: 'DTO', description: '更新结果对象。' }],
        notes: ['失败时抛出 SwmApiException 派生类型。'],
        example: codeExamples.csharp.checkUpdate
      },
      {
        name: 'DownloadAsync',
        signature: 'Task DownloadAsync(string url, string destPath, string? checksum = null, string? signature = null, Action<long,long>? progress = null, CancellationToken cancellationToken = default)',
        endpoint: 'GET <download_url>',
        description: '下载并校验文件。',
        params: [
          { name: 'url/destPath', type: 'string', required: '是', description: '下载 URL 与本地路径。' },
          { name: 'checksum/signature', type: 'string?', required: '否', description: '可选 checksum 和签名。' },
          { name: 'progress', type: 'Action<long,long>?', required: '否', description: '下载进度回调。' }
        ],
        returns: [{ name: 'Task', type: 'async', description: '失败时抛出异常。' }],
        example: codeExamples.csharp.download
      },
      {
        name: 'ReportEventAsync / ReportEventsAsync',
        signature: 'Task ReportEventAsync(...) / Task ReportEventsAsync(List<EventIngestItem> events, ...)',
        endpoint: 'POST /api/client/events',
        description: '单条与批量事件上报。',
        params: [
          { name: 'eventName/properties', type: 'string + Dictionary', required: '单条必填事件名', description: '单条上报参数。' },
          { name: 'events', type: 'List<EventIngestItem>', required: '批量必填', description: '批量事件列表。' }
        ],
        returns: [{ name: 'Task', type: 'async', description: '失败抛异常。' }],
        example: codeExamples.csharp.reportEvent
      },
      {
        name: 'ReportHeartbeatAsync',
        signature: 'Task ReportHeartbeatAsync(string? appVersion = null, string? userId = null, CancellationToken cancellationToken = default)',
        endpoint: 'POST /api/client/heartbeat',
        description: '设备在线心跳。',
        params: [
          { name: 'appVersion/userId', type: 'string?', required: '否', description: '版本号和用户 ID。' },
          { name: 'DeviceId', type: 'client field', required: '是', description: 'DeviceId 为空会抛 SwmValidationException。' }
        ],
        returns: [{ name: 'Task', type: 'async', description: '失败抛异常。' }],
        example: codeExamples.csharp.reportHeartbeat
      },
      {
        name: 'ReportFeedbackAsync',
        signature: 'Task ReportFeedbackAsync(string content, int? rating = null, string? contact = null, IEnumerable<string>? attachments = null, Dictionary<string, object?>? metadata = null, CancellationToken cancellationToken = default)',
        endpoint: 'POST /api/client/feedback (multipart/form-data)',
        description: '反馈上报（含附件）。',
        params: [
          { name: 'content', type: 'string', required: '是', description: '反馈正文。' },
          { name: 'rating/contact', type: 'int?/string?', required: '否', description: '可选评分和联系方式。' },
          { name: 'attachments', type: 'IEnumerable<string>?', required: '否', description: '本地文件路径集合。' },
          { name: 'metadata', type: 'Dictionary<string, object?>?', required: '否', description: '扩展元数据。' }
        ],
        returns: [{ name: 'Task', type: 'async', description: '失败抛异常。' }],
        example: codeExamples.csharp.reportFeedback
      },
      {
        name: 'StartUpdateStream / WatchUpdates',
        signature: 'UpdateWatchHandle StartUpdateStream(...) / UpdateWatchHandle WatchUpdates(...)',
        endpoint: 'GET /api/client/updates/stream (+ update-check)',
        description: '实时更新推送与自动二次检查。',
        params: [
          { name: 'options', type: 'UpdateStreamOptions', required: '是', description: '流配置。' },
          { name: 'onEvent/onUpdateAvailable', type: 'Action', required: '是', description: '事件回调。' }
        ],
        returns: [{ name: 'UpdateWatchHandle', type: 'handle', description: 'Stop()/Dispose() 停止监听。' }],
        example: `var handle = client.WatchUpdates(
    new UpdateStreamOptions { CurrentVersion = "1.0.0", VersionCode = 100 },
    resp => Console.WriteLine(resp.Version)
);`
      },
      {
        name: 'SetAuthToken',
        signature: 'void SetAuthToken(string token)',
        endpoint: '管理端 API 共用',
        description: '设置 Bearer Token。',
        params: [{ name: 'token', type: 'string', required: '是', description: '后台登录令牌。' }],
        returns: [{ name: 'void', type: '-', description: '仅更新客户端状态。' }],
        example: `client.SetAuthToken(Environment.GetEnvironmentVariable("SWM_ACCESS_TOKEN") ?? "");`
      }
    ],
    errors: [
      { name: 'SwmDeviceBlockedException', type: 'Exception', description: 'error.code = device_blocked。' },
      { name: 'SwmUpdateRegionBlockedException', type: 'Exception', description: 'error.code = update_region_blocked。' },
      { name: 'SwmUnauthorizedException', type: 'Exception', description: '401/403 或缺少 AuthToken。' },
      { name: 'SwmValidationException', type: 'Exception', description: '4xx 参数类错误。' },
      { name: 'SwmApiException', type: 'Exception', description: '通用 SDK 异常基类。' }
    ],
    managementAuthNote: '调用管理端 *Async 方法前必须执行 client.SetAuthToken(token)。',
    managementMethods: [
      { method: 'GetAppAsync / UpdateAppAsync', endpoint: 'GET/PATCH /api/apps/{appId}', description: '应用查询与更新。' },
      { method: 'ListChannelsAsync / CreateChannelAsync', endpoint: 'GET/POST /api/apps/{appId}/channels', description: '渠道管理。' },
      { method: 'ListAppMembersAsync / AddAppMemberAsync', endpoint: 'GET/POST /api/apps/{appId}/members', description: '成员管理。' },
      { method: 'Release 系列（List/Create/Update/Delete/Submit/Approve/Reject/Publish/Rollback/Revoke）', endpoint: '/api/apps/{appId}/releases /api/releases/{releaseId}/...', description: '发布全生命周期。' },
      { method: 'Artifact 系列（Upload/List/GetDownloadUrl）', endpoint: '/api/releases/{releaseId}/artifacts /api/artifacts/{artifactId}/download', description: '制品管理。' },
      { method: 'ReleaseChannel + Metrics', endpoint: '/api/apps/{appId}/release-channels /metrics', description: '发布通道与指标。' },
      { method: 'RegionRules + Geo + Templates + AppSecrets', endpoint: '/api/apps/{appId}/region-rules /api/geo/* /api/release-templates /api/apps/{appId}/app-secrets', description: '区域、地理、模板、密钥。' }
    ]
  }
}

type DocsLanguage = 'zh' | 'en'

type DocTableData = {
  headers: string[]
  rows: string[][]
}

const DOCS_LANGUAGE_STORAGE_KEY = 'swm_landing_lang'

const zhToEnDictionary: Array<[string, string]> = [
  ['请求 Header 需包含：X-App-Id、X-Timestamp、X-Nonce、X-Signature、X-Sign-Version(v1)。', 'Required request headers: X-App-Id, X-Timestamp, X-Nonce, X-Signature, X-Sign-Version(v1).'],
  ['SWM (Software Web Manager) 当前主要维护 Go 与 C# 两个官方 SDK。本文档示例和方法签名已与仓库内最新 SDK 对齐，可直接按示例调用。', 'SWM (Software Web Manager) currently maintains two official SDKs: Go and C#. Examples and method signatures in this document are aligned with the latest repository source and can be used directly.'],
  ['使用 API 基础 URL、`app_id` 与 `app_secret` 初始化客户端。您可以在', 'Initialize the client using API base URL, `app_id`, and `app_secret`. You can get these credentials on the '],
  ['页面获取这两个凭据。', ' page.'],
  ['调用 checkUpdate 方法检查是否有新版本可用。', 'Call checkUpdate to check whether a new version is available.'],
  ['如果检测到更新，使用 download 方法下载更新包。SDK 会自动验证文件完整性。', 'If an update is found, use download to fetch the update package. The SDK will verify file integrity automatically.'],
  ['使用 reportEvent 方法上报应用使用事件，帮助分析用户行为。', 'Use reportEvent to report app usage events for behavior analysis.'],
  ['使用 reportHeartbeat 方法定期上报心跳，便于统计实时在线设备。', 'Use reportHeartbeat to report heartbeats periodically for real-time online device statistics.'],
  ['使用 reportFeedback 方法上报用户反馈，支持携带截图附件。', 'Use reportFeedback to submit user feedback with optional screenshot attachments.'],
  ['SWM 提供 RESTful API 接口，您可以直接调用 API 或使用 SDK 进行集成。', 'SWM provides RESTful APIs. You can call APIs directly or integrate via SDKs.'],
  ['仅维护 Go / C# 两个官方 SDK 文档，内容直接对齐仓库中的最新源码签名与参数定义。', 'Only Go/C# official SDK docs are maintained, aligned directly with latest source signatures and parameter definitions in the repository.'],
  ['支持 7 种编程语言：Go、Java、Python、Node.js、C++、C#、Rust', 'Supports 7 programming languages: Go, Java, Python, Node.js, C++, C#, Rust'],
  ['提供文件下载功能（支持进度回调和 SHA256 校验）', 'Provides file download with progress callback and SHA256 verification'],
  ['截图附件（最多 3 张，单张 ≤ 5MB）', 'Screenshot attachments (up to 3 files, each <= 5MB)'],
  ['调用管理端 *Async 方法前必须执行 client.SetAuthToken(token)。', 'Call client.SetAuthToken(token) before invoking management-side *Async APIs.'],
  ['调用管理端 API 前先执行 client.SetAuthToken(token)。', 'Call client.SetAuthToken(token) before invoking management APIs.'],
  ['版本检查入口。请求体自动包含 Client 的 channel/platform/arch/device_id/user_id/attributes。', 'Entry point for update checks. Request body automatically includes Client channel/platform/arch/device_id/user_id/attributes.'],
  ['单条与批量事件上报。批量模式用于本地缓存补偿。', 'Single and batch event reporting. Batch mode can be used for local cache compensation.'],
  ['单条与批量事件上报。', 'Single and batch event reporting.'],
  ['设备在线心跳。DeviceID 不能为空，否则返回 device_id required。', 'Device online heartbeat. DeviceID cannot be empty, otherwise `device_id required` is returned.'],
  ['下载更新文件并执行校验；destPath 目录不存在会自动创建。', 'Download update file and verify it; destination directory is created automatically if missing.'],
  ['反馈上报，支持评分、联系方式、附件、metadata。', 'Submit feedback with optional rating, contact info, attachments, and metadata.'],
  ['建立更新推送流；handle.Stop() 主动关闭。', 'Create update push stream; call handle.Stop() to close it.'],
  ['收到推送后自动触发 CheckUpdate，发现更新才回调。', 'Automatically triggers CheckUpdate after push events and calls back only when updates are available.'],
  ['设置 Bearer Token（管理端 API 必须先设置）。', 'Set Bearer Token (required before calling management APIs).'],
  ['实时更新推送与自动二次检查。', 'Real-time update push and automatic secondary checks.'],
  ['反馈上报（含附件）。', 'Feedback reporting (with attachments).'],
  ['Go / C# SDK 方法与接口映射', 'Go / C# SDK Method and API Mapping'],
  ['Client 运行时字段', 'Client Runtime Fields'],
  ['CheckUpdate 返回字段', 'CheckUpdate Return Fields'],
  ['UpdateStreamOptions 参数', 'UpdateStreamOptions Parameters'],
  ['错误与异常模型', 'Error and Exception Model'],
  ['管理端 API 方法索引', 'Management API Method Index'],
  ['快速开始', 'Quick Start'],
  ['API 文档', 'API Docs'],
  ['SDK 参考', 'SDK Reference'],
  ['更新日志', 'Changelog'],
  ['选择编程语言', 'Choose Programming Language'],
  ['选择语言', 'Choose Language'],
  ['初始化', 'Initialization'],
  ['完整调用流程', 'Complete Flow'],
  ['检查更新', 'Check Update'],
  ['下载更新', 'Download Update'],
  ['更新推送流', 'Update Stream'],
  ['事件上报', 'Event Reporting'],
  ['心跳上报', 'Heartbeat Reporting'],
  ['用户反馈', 'User Feedback'],
  ['应用管理', 'App Management'],
  ['请求参数', 'Request Parameters'],
  ['响应参数', 'Response Parameters'],
  ['方法签名：', 'Method Signature:'],
  ['底层接口：', 'Underlying API:'],
  ['注意事项', 'Notes'],
  ['示例', 'Example'],
  ['发现新版本:', 'New version found:'],
  ['发现新版本: ', 'New version found: '],
  ['发现新版本: {}', 'New version found: {}'],
  ['发现新版本: {resp.Version}', 'New version found: {resp.Version}'],
  ['发现新版本: %s\\n', 'New version found: %s\\n'],
  ['下载地址:', 'Download URL:'],
  ['下载地址: ', 'Download URL: '],
  ['下载地址: {}', 'Download URL: {}'],
  ['下载地址: {resp.download_url}', 'Download URL: {resp.download_url}'],
  ['下载地址: {resp.DownloadUrl}', 'Download URL: {resp.DownloadUrl}'],
  ['下载地址: %s\\n', 'Download URL: %s\\n'],
  ['校验值: %s\\n', 'Checksum: %s\\n'],
  ['签名: %s\\n', 'Signature: %s\\n'],
  ['强制更新:', 'Mandatory update:'],
  ['强制更新: ', 'Mandatory update: '],
  ['强制更新: {}', 'Mandatory update: {}'],
  ['强制更新: {resp.Mandatory}', 'Mandatory update: {resp.Mandatory}'],
  ['强制更新: %v\\n', 'Mandatory update: %v\\n'],
  ['下载进度:', 'Download progress:'],
  ['下载进度: {progress:F2}%', 'Download progress: {progress:F2}%'],
  ['下载进度: %.2f%%\\n', 'Download progress: %.2f%%\\n'],
  ['设备已被封禁', 'Device is blocked'],
  ['当前地区不允许更新', 'Updates are not allowed in the current region'],
  ['反馈内容', 'Feedback content'],
  ['反馈内容。', 'Feedback content.'],
  ['事件名称。', 'Event name.'],
  ['设备在线心跳。', 'Device online heartbeat.'],
  ['初始版本发布', 'Initial release'],
  ['提供检查更新功能', 'Provides update check capability'],
  ['提供事件上报功能', 'Provides event reporting capability'],
  ['1. 初始化客户端', '1. Initialize Client'],
  ['2. 检查更新', '2. Check Update'],
  ['3. 下载更新', '3. Download Update'],
  ['4. 事件上报', '4. Event Reporting'],
  ['5. 心跳上报', '5. Heartbeat Reporting'],
  ['6. 用户反馈', '6. User Feedback'],
  ['返回', 'Return'],
  ['单条事件', 'Single Event'],
  ['批量事件', 'Batch Events'],
  ['反馈上报', 'Feedback Reporting'],
  ['应用 ID', 'App ID'],
  ['发布渠道', 'Release Channel'],
  ['当前版本号', 'Current Version'],
  ['版本代码', 'Version Code'],
  ['平台 (windows/linux/macos)', 'Platform (windows/linux/macos)'],
  ['架构 (x64/arm64)', 'Architecture (x64/arm64)'],
  ['设备唯一标识', 'Unique Device Identifier'],
  ['是否有可用更新', 'Whether an update is available'],
  ['是否强制更新', 'Whether update is mandatory'],
  ['新版本号', 'New Version'],
  ['下载地址', 'Download URL'],
  ['SHA256 校验值', 'SHA256 checksum'],
  ['文件大小（字节）', 'File size (bytes)'],
  ['更新说明', 'Release notes'],
  ['事件名称', 'Event name'],
  ['事件时间（ISO8601）', 'Event time (ISO8601)'],
  ['事件属性', 'Event properties'],
  ['应用版本', 'App version'],
  ['平台', 'Platform'],
  ['架构', 'Architecture'],
  ['用户ID', 'User ID'],
  ['设备属性', 'Device properties'],
  ['渠道代码', 'Channel code'],
  ['客户端版本', 'Client version'],
  ['评分 (1-5)', 'Rating (1-5)'],
  ['联系方式', 'Contact'],
  ['设备/自定义字段', 'Device/custom fields'],
  ['由 New(baseURL, appID, appSecret) 初始化，三者均为签名所需。', 'Initialized via New(baseURL, appID, appSecret); all three are required for signing.'],
  ['客户端上下文字段；更新检查、事件、心跳、反馈、流式更新均会使用。', 'Client context fields used by update checks, events, heartbeats, feedback, and update streaming.'],
  ['可选用户 ID。', 'Optional user ID.'],
  ['可选扩展属性，会并入请求体。', 'Optional extra attributes merged into request body.'],
  ['开启后会对 checksum 与 signature 做 Ed25519 校验。', 'When enabled, performs Ed25519 verification on checksum and signature.'],
  ['请求重试与退避参数，默认 2 次、500ms。', 'Request retry and backoff settings, default 2 retries and 500ms.'],
  ['可替换默认客户端（默认超时 30s）。', 'Default HTTP client can be replaced (default timeout 30s).'],
  ['管理端 API 鉴权 token，通过 SetAuthToken 设置。', 'Management API auth token, set via SetAuthToken.'],
  ['更新可用性与目标版本基础信息。', 'Update availability and target version base info.'],
  ['下载与校验字段。', 'Download and verification fields.'],
  ['展示与投递策略字段。', 'Display and delivery strategy fields.'],
  ['心跳建议与回滚能力。', 'Heartbeat hints and rollback capability.'],
  ['必需字段；为空时尝试从 Client 对应字段回填。', 'Required fields; if empty, fallback from corresponding Client fields.'],
  ['WatchUpdates 内二次调用 CheckUpdate 时使用。', 'Used when WatchUpdates invokes CheckUpdate internally.'],
  ['重连行为控制（默认自动重连+抖动）。', 'Reconnect behavior control (auto-reconnect + jitter by default).'],
  ['错误和控制事件回调。', 'Error and control event callbacks.'],
  ['请求上下文。', 'Request context.'],
  ['客户端当前版本。', 'Current client version.'],
  ['可选版本号 code。', 'Optional numeric version code.'],
  ['包含下载地址、签名、强更标记等字段。', 'Includes fields like download URL, signature, mandatory flag, etc.'],
  ['可能返回 ErrDeviceBlocked / ErrUpdateRegionBlocked。', 'May return ErrDeviceBlocked / ErrUpdateRegionBlocked.'],
  ['当 VerifySignature=true 且响应携带 signature+checksum_sha256 时，会自动验签。', 'When VerifySignature=true and response contains signature+checksum_sha256, verification runs automatically.'],
  ['下载地址与目标文件路径。', 'Download URL and destination file path.'],
  ['SHA256；为空则跳过校验。', 'SHA256; verification is skipped when empty.'],
  ['签名值，配合 VerifySignature 使用。', 'Signature value, used with VerifySignature.'],
  ['下载进度回调。', 'Download progress callback.'],
  ['下载失败或校验失败。', 'Download failed or verification failed.'],
  ['单条事件属性。', 'Single event properties.'],
  ['批量事件数组。', 'Batch event array.'],
  ['请求失败返回错误。', 'Returns error when request fails.'],
  ['客户端版本号。', 'Client version.'],
  ['设备唯一 ID。', 'Unique device ID.'],
  ['评分。', 'Rating.'],
  ['联系方式。', 'Contact info.'],
  ['本地文件路径列表。', 'Local file path list.'],
  ['业务元数据。', 'Business metadata.'],
  ['参数、文件读取或请求失败。', 'Failed due to invalid params, file reading, or request error.'],
  ['流选项，包含重连和回调配置。', 'Stream options including reconnect and callback configs.'],
  ['收到推送事件时触发。', 'Triggered when receiving push events.'],
  ['调用 Stop() 结束监听。', 'Call Stop() to end listening.'],
  ['参数不完整或连接初始化失败。', 'Incomplete params or failed connection initialization.'],
  ['二次检查使用的当前版本。', 'Current version used for second-stage checks.'],
  ['有更新时回调。', 'Callback when update is available.'],
  ['监听控制句柄与错误。', 'Listening handle and errors.'],
  ['管理端 API 共用', 'Shared management API'],
  ['后台登录返回的 access token。', 'Access token returned after backend login.'],
  ['仅更新 Client 状态。', 'Only updates Client state.'],
  ['其他 error', 'Other errors'],
  ['网络错误、状态码错误、验签失败等。', 'Network/status/signature verification errors, etc.'],
  ['应用信息查询与更新。', 'Query and update app info.'],
  ['渠道管理。', 'Channel management.'],
  ['成员管理。', 'Member management.'],
  ['发布基础生命周期。', 'Base release lifecycle.'],
  ['发布审核与通道操作。', 'Release approval and channel operations.'],
  ['制品上传与下载地址。', 'Artifact upload and download URL.'],
  ['区域规则、地理解析、模板与密钥管理。', 'Region rules, geo resolution, template and secret management.'],
  ['由构造函数传入，签名必需。', 'Passed by constructor, required for signature.'],
  ['客户端上下文。', 'Client context.'],
  ['默认用户 ID，可被方法参数覆盖。', 'Default user ID, can be overridden by method params.'],
  ['扩展属性。', 'Extended attributes.'],
  ['验签开关、公钥和自定义验签实现。', 'Signature verification toggle, public key, and custom verifier.'],
  ['重试参数，默认 2 次 + 500ms。', 'Retry params, default 2 retries + 500ms.'],
  ['可注入外部 HttpClient。', 'External HttpClient can be injected.'],
  ['管理端 Bearer Token。', 'Management Bearer Token.'],
  ['更新判定与目标版本信息。', 'Update decision and target version info.'],
  ['展示和投递相关字段。', 'Display and delivery related fields.'],
  ['为空时回退到 Client 对应字段。', 'Fallback to Client fields when empty.'],
  ['WatchUpdates 二次检查参数。', 'WatchUpdates second-check params.'],
  ['重连与退避控制。', 'Reconnect and backoff control.'],
  ['错误与控制事件回调。', 'Error and control event callbacks.'],
  ['客户端+管理端通用', 'Client + Management shared'],
  ['创建 SDK 客户端实例。', 'Create SDK client instance.'],
  ['API 根地址。', 'API base URL.'],
  ['应用身份与密钥。', 'App identity and secret.'],
  ['可选外部注入。', 'Optional external injection.'],
  ['可复用实例。', 'Reusable instance.'],
  ['执行更新检查。', 'Execute update check.'],
  ['当前版本。', 'Current version.'],
  ['可选数值版本号。', 'Optional numeric version code.'],
  ['覆盖 client.UserId。', 'Override client.UserId.'],
  ['取消令牌。', 'Cancellation token.'],
  ['更新结果对象。', 'Update result object.'],
  ['失败时抛出 SwmApiException 派生类型。', 'Throws derived SwmApiException on failure.'],
  ['下载并校验文件。', 'Download and verify file.'],
  ['下载 URL 与本地路径。', 'Download URL and local path.'],
  ['可选 checksum 和签名。', 'Optional checksum and signature.'],
  ['失败时抛出异常。', 'Throws exception on failure.'],
  ['单条必填事件名', 'Event name required for single event'],
  ['单条上报参数。', 'Single event payload params.'],
  ['批量必填', 'Batch required'],
  ['批量事件列表。', 'Batch event list.'],
  ['失败抛异常。', 'Throws exception on failure.'],
  ['版本号和用户 ID。', 'Version and user ID.'],
  ['DeviceId 为空会抛 SwmValidationException。', 'Throws SwmValidationException when DeviceId is empty.'],
  ['反馈正文。', 'Feedback body.'],
  ['可选评分和联系方式。', 'Optional rating and contact.'],
  ['本地文件路径集合。', 'Local file path list.'],
  ['扩展元数据。', 'Extended metadata.'],
  ['流配置。', 'Stream configuration.'],
  ['事件回调。', 'Event callback.'],
  ['Stop()/Dispose() 停止监听。', 'Stop listening with Stop()/Dispose().'],
  ['设置 Bearer Token。', 'Set Bearer Token.'],
  ['后台登录令牌。', 'Backend login token.'],
  ['仅更新客户端状态。', 'Only updates client state.'],
  ['401/403 或缺少 AuthToken。', '401/403 or missing AuthToken.'],
  ['4xx 参数类错误。', '4xx parameter validation errors.'],
  ['通用 SDK 异常基类。', 'Base SDK exception class.'],
  ['应用查询与更新。', 'App query and update.'],
  ['发布全生命周期。', 'Full release lifecycle.'],
  ['制品管理。', 'Artifact management.'],
  ['发布通道与指标。', 'Release channels and metrics.'],
  ['区域、地理、模板、密钥。', 'Region, geo, templates, and secrets.'],
  ['Release 系列（List/Create/Update/Delete/Submit/Approve/Reject/Publish/Rollback/Revoke）', 'Release series (List/Create/Update/Delete/Submit/Approve/Reject/Publish/Rollback/Revoke)'],
  ['Artifact 系列（Upload/List/GetDownloadUrl）', 'Artifact series (Upload/List/GetDownloadUrl)'],
  ['Region/Geo/Template/AppSecret 系列', 'Region/Geo/Template/AppSecret series'],
  ['字段', 'Field'],
  ['参数', 'Parameter'],
  ['名称', 'Name'],
  ['能力', 'Capability'],
  ['接口', 'Endpoint'],
  ['底层接口', 'Underlying API'],
  ['SDK 方法', 'SDK Method'],
  ['类型', 'Type'],
  ['说明', 'Description'],
  ['必填', 'Required'],
  ['已复制到剪贴板', 'Copied to clipboard'],
  ['是(单条)', 'Yes (single)'],
  ['是(批量)', 'Yes (batch)'],
  ['建议必填', 'Recommended'],
  ['无', 'None'],
  ['是', 'Yes'],
  ['否', 'No']
]

const sortedZhToEnDictionary = [...zhToEnDictionary].sort((a, b) => b[0].length - a[0].length)

const getDocsLanguage = (resolvedLanguage?: string): DocsLanguage => {
  if (resolvedLanguage?.toLowerCase().startsWith('en')) {
    return 'en'
  }
  if (resolvedLanguage?.toLowerCase().startsWith('zh')) {
    return 'zh'
  }
  if (typeof window !== 'undefined') {
    const savedLanguage = window.localStorage.getItem(DOCS_LANGUAGE_STORAGE_KEY)
    if (savedLanguage === 'zh' || savedLanguage === 'en') {
      return savedLanguage
    }
  }
  return 'zh'
}

const localizeText = (value: string, isEnglish: boolean): string => {
  if (!isEnglish) {
    return value
  }
  return sortedZhToEnDictionary.reduce((acc, [source, target]) => acc.split(source).join(target), value)
}

const localizeDeep = <T,>(value: T, isEnglish: boolean): T => {
  if (!isEnglish) {
    return value
  }
  if (typeof value === 'string') {
    return localizeText(value, true) as T
  }
  if (Array.isArray(value)) {
    return value.map((item) => localizeDeep(item, true)) as T
  }
  if (value && typeof value === 'object') {
    const localized: Record<string, unknown> = {}
    Object.entries(value as Record<string, unknown>).forEach(([key, item]) => {
      localized[key] = localizeDeep(item, true)
    })
    return localized as T
  }
  return value
}

const apiSdkMappingTableZh: DocTableData = {
  headers: ['能力', 'Go SDK', 'C# SDK', '底层接口'],
  rows: [
    ['检查更新', 'CheckUpdate(ctx, currentVersion, versionCode)', 'CheckUpdateAsync(currentVersion, versionCode, userId)', 'POST /api/client/update-check'],
    ['下载更新', 'Download(ctx, url, destPath, checksum, signature, progress)', 'DownloadAsync(url, destPath, checksum, signature, progress)', 'GET <download_url>'],
    ['单条事件', 'ReportEvent(ctx, eventName, props)', 'ReportEventAsync(eventName, properties)', 'POST /api/client/events'],
    ['批量事件', 'ReportEvents(ctx, events)', 'ReportEventsAsync(events)', 'POST /api/client/events'],
    ['心跳上报', 'ReportHeartbeat(ctx, appVersion)', 'ReportHeartbeatAsync(appVersion, userId)', 'POST /api/client/heartbeat'],
    ['反馈上报', 'ReportFeedback(ctx, content, rating, contact, attachments, metadata)', 'ReportFeedbackAsync(content, rating, contact, attachments, metadata)', 'POST /api/client/feedback'],
    ['更新推送流', 'StartUpdateStream / WatchUpdates', 'StartUpdateStream / WatchUpdates', 'GET /api/client/updates/stream']
  ]
}

const updateCheckRequestTableZh: DocTableData = {
  headers: ['参数', '类型', '必填', '说明'],
  rows: [
    ['X-App-Id (Header)', 'string(UUID)', '是', '应用 ID'],
    ['channel_code', 'string', '是', '发布渠道'],
    ['current_version', 'string', '是', '当前版本号'],
    ['version_code', 'integer', '否', '版本代码'],
    ['platform', 'string', '是', '平台 (windows/linux/macos)'],
    ['arch', 'string', '是', '架构 (x64/arm64)'],
    ['device_id', 'string', '是', '设备唯一标识']
  ]
}

const updateCheckResponseTableZh: DocTableData = {
  headers: ['参数', '类型', '说明'],
  rows: [
    ['update_available', 'boolean', '是否有可用更新'],
    ['mandatory', 'boolean', '是否强制更新'],
    ['version', 'string', '新版本号'],
    ['download_url', 'string', '下载地址'],
    ['checksum_sha256', 'string', 'SHA256 校验值'],
    ['size', 'integer', '文件大小（字节）'],
    ['notes', 'string', '更新说明']
  ]
}

const eventReportRequestTableZh: DocTableData = {
  headers: ['参数', '类型', '必填', '说明'],
  rows: [
    ['X-App-Id (Header)', 'string(UUID)', '是', '应用 ID'],
    ['device_id', 'string', '是', '设备唯一标识'],
    ['event_name', 'string', '是', '事件名称'],
    ['event_time', 'string', '是', '事件时间（ISO8601）'],
    ['properties', 'object', '否', '事件属性']
  ]
}

const heartbeatRequestTableZh: DocTableData = {
  headers: ['参数', '类型', '必填', '说明'],
  rows: [
    ['X-App-Id (Header)', 'string(UUID)', '是', '应用 ID'],
    ['device_id', 'string', '是', '设备唯一标识'],
    ['channel_code', 'string', '否', '发布渠道'],
    ['app_version', 'string', '否', '应用版本'],
    ['platform', 'string', '否', '平台'],
    ['arch', 'string', '否', '架构'],
    ['user_id', 'string', '否', '用户ID'],
    ['attributes', 'object', '否', '设备属性']
  ]
}

const feedbackRequestTableZh: DocTableData = {
  headers: ['参数', '类型', '必填', '说明'],
  rows: [
    ['X-App-Id (Header)', 'string(UUID)', '是', '应用 ID'],
    ['device_id', 'string', '是', '设备唯一标识'],
    ['content', 'string', '是', '反馈内容'],
    ['channel_code', 'string', '否', '渠道代码'],
    ['app_version', 'string', '否', '客户端版本'],
    ['rating', 'integer', '否', '评分 (1-5)'],
    ['contact', 'string', '否', '联系方式'],
    ['metadata', 'json string', '否', '设备/自定义字段'],
    ['attachments[]', 'file', '否', '截图附件（最多 3 张，单张 ≤ 5MB）'],
    ['feedback_disabled', 'error code', '否', '应用关闭用户反馈时返回，Go SDK 可通过 ErrFeedbackDisabled 判断']
  ]
}

function CodeBlock({ code, copySuccessText }: { code: string; copySuccessText: string }) {
  const handleCopy = () => {
    navigator.clipboard.writeText(code)
    message.success(copySuccessText)
  }

  return (
    <div style={{ position: 'relative' }}>
      <pre
        style={{
          background: '#1e1e1e',
          color: '#d4d4d4',
          padding: '16px',
          borderRadius: '8px',
          overflow: 'auto',
          fontSize: '13px',
          lineHeight: '1.6',
          fontFamily: 'Consolas, Monaco, monospace'
        }}
      >
        <code>{code}</code>
      </pre>
      <Button
        type="text"
        icon={<CopyOutlined />}
        onClick={handleCopy}
        style={{
          position: 'absolute',
          top: '8px',
          right: '8px',
          color: '#999',
          background: 'rgba(255,255,255,0.1)'
        }}
      />
    </div>
  )
}

export default function Docs() {
  const { i18n } = useTranslation()
  const [selectedLang, setSelectedLang] = useState<SupportedLang>('go')
  const [activeTab, setActiveTab] = useState<'quickstart' | 'api' | 'sdk' | 'changelog'>('quickstart')

  const docsLanguage = getDocsLanguage(i18n.resolvedLanguage)
  const isEnglish = docsLanguage === 'en'
  const tr = (value: string) => localizeText(value, isEnglish)
  const copySuccessText = tr('已复制到剪贴板')

  const examples = useMemo(() => localizeDeep(codeExamples[selectedLang], isEnglish), [selectedLang, isEnglish])
  const sdkDoc = useMemo(() => localizeDeep(sdkReferenceDocs[selectedLang], isEnglish), [selectedLang, isEnglish])
  const apiSdkMappingTable = useMemo(() => localizeDeep(apiSdkMappingTableZh, isEnglish), [isEnglish])
  const updateCheckRequestTable = useMemo(() => localizeDeep(updateCheckRequestTableZh, isEnglish), [isEnglish])
  const updateCheckResponseTable = useMemo(() => localizeDeep(updateCheckResponseTableZh, isEnglish), [isEnglish])
  const eventReportRequestTable = useMemo(() => localizeDeep(eventReportRequestTableZh, isEnglish), [isEnglish])
  const heartbeatRequestTable = useMemo(() => localizeDeep(heartbeatRequestTableZh, isEnglish), [isEnglish])
  const feedbackRequestTable = useMemo(() => localizeDeep(feedbackRequestTableZh, isEnglish), [isEnglish])

  const menuItems = [
    { key: 'quickstart', icon: <RocketOutlined />, label: tr('快速开始') },
    { key: 'api', icon: <ApiOutlined />, label: tr('API 文档') },
    { key: 'sdk', icon: <CodeOutlined />, label: tr('SDK 参考') },
    { key: 'changelog', icon: <FileTextOutlined />, label: tr('更新日志') }
  ]

  return (
    <div style={{ padding: '24px' }}>
      <Content style={{ maxWidth: 1200 }}>
        <Card style={{ marginBottom: 24 }}>
          <Menu
            mode="horizontal"
            selectedKeys={[activeTab]}
            items={menuItems}
            onClick={({ key }) => setActiveTab(key as 'quickstart' | 'api' | 'sdk' | 'changelog')}
          />
        </Card>

        {activeTab === 'quickstart' && (
          <div>
            <Title level={2}>{tr('快速开始')}</Title>
            <Paragraph>
              {tr('SWM (Software Web Manager) 当前主要维护 Go 与 C# 两个官方 SDK。本文档示例和方法签名已与仓库内最新 SDK 对齐，可直接按示例调用。')}
            </Paragraph>

            <Card style={{ marginTop: 24 }}>
              <Title level={4}>{tr('选择编程语言')}</Title>
              <Space wrap style={{ marginTop: 16 }}>
                {languages.map(lang => (
                  <Tag
                    key={lang.key}
                    color={selectedLang === lang.key ? lang.color : undefined}
                    style={{
                      cursor: 'pointer',
                      padding: '4px 12px',
                      fontSize: 14
                    }}
                    onClick={() => setSelectedLang(lang.key)}
                  >
                    {lang.label}
                  </Tag>
                ))}
              </Space>
            </Card>

            <Card style={{ marginTop: 24 }}>
              <Title level={4}>{tr('1. 初始化客户端')}</Title>
              <Paragraph>
                {tr('使用 API 基础 URL、`app_id` 与 `app_secret` 初始化客户端。您可以在')}
                <Link to="/apps">{tr('应用管理')}</Link>
                {tr('页面获取这两个凭据。')}
              </Paragraph>
              <CodeBlock code={examples.init} copySuccessText={copySuccessText} />
            </Card>

            <Card style={{ marginTop: 24 }}>
              <Title level={4}>{tr('2. 检查更新')}</Title>
              <Paragraph>
                {tr('调用 checkUpdate 方法检查是否有新版本可用。')}
              </Paragraph>
              <CodeBlock code={examples.checkUpdate} copySuccessText={copySuccessText} />
            </Card>

            <Card style={{ marginTop: 24 }}>
              <Title level={4}>{tr('3. 下载更新')}</Title>
              <Paragraph>
                {tr('如果检测到更新，使用 download 方法下载更新包。SDK 会自动验证文件完整性。')}
              </Paragraph>
              <CodeBlock code={examples.download} copySuccessText={copySuccessText} />
            </Card>

            <Card style={{ marginTop: 24 }}>
              <Title level={4}>{tr('4. 事件上报')}</Title>
              <Paragraph>
                {tr('使用 reportEvent 方法上报应用使用事件，帮助分析用户行为。')}
              </Paragraph>
              <CodeBlock code={examples.reportEvent} copySuccessText={copySuccessText} />
            </Card>

            <Card style={{ marginTop: 24 }}>
              <Title level={4}>{tr('5. 心跳上报')}</Title>
              <Paragraph>
                {tr('使用 reportHeartbeat 方法定期上报心跳，便于统计实时在线设备。')}
              </Paragraph>
              <CodeBlock code={examples.reportHeartbeat} copySuccessText={copySuccessText} />
            </Card>

            <Card style={{ marginTop: 24 }}>
              <Title level={4}>{tr('6. 用户反馈')}</Title>
              <Paragraph>
                {tr('使用 reportFeedback 方法上报用户反馈，支持携带截图附件。')}
              </Paragraph>
              <Paragraph type="secondary">
                {tr('后台“用户反馈”开关关闭后，服务端会拒绝新的 SDK 上报并返回 feedback_disabled；已上报的历史反馈仍可继续查看和处理。')}
              </Paragraph>
              <CodeBlock code={examples.reportFeedback} copySuccessText={copySuccessText} />
            </Card>
          </div>
        )}

        {activeTab === 'api' && (
          <div>
            <Title level={2}>{tr('API 文档')}</Title>
            <Paragraph>
              {tr('SWM 提供 RESTful API 接口，您可以直接调用 API 或使用 SDK 进行集成。')}
            </Paragraph>

            <Card title={tr('Go / C# SDK 方法与接口映射')} style={{ marginTop: 24 }}>
              <DocTable
                headers={apiSdkMappingTable.headers}
                rows={apiSdkMappingTable.rows}
              />
            </Card>

            <Card title={tr('检查更新')} style={{ marginTop: 24 }}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <div>
                  <Tag color="blue">POST</Tag>
                  <Text code>/api/client/update-check</Text>
                </div>
                <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  {tr('请求 Header 需包含：X-App-Id、X-Timestamp、X-Nonce、X-Signature、X-Sign-Version(v1)。')}
                </Paragraph>
                <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  {tr('metadata 会在后台反馈详情中展示；附件会作为截图或下载文件展示。应用关闭用户反馈时返回 feedback_disabled。')}
                </Paragraph>
                <Title level={5}>{tr('请求参数')}</Title>
                <DocTable headers={updateCheckRequestTable.headers} rows={updateCheckRequestTable.rows} />
                <Title level={5} style={{ marginTop: 24 }}>{tr('响应参数')}</Title>
                <DocTable headers={updateCheckResponseTable.headers} rows={updateCheckResponseTable.rows} />
              </Space>
            </Card>

              <Card title={tr('事件上报')} style={{ marginTop: 24 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <div>
                    <Tag color="blue">POST</Tag>
                    <Text code>/api/client/events</Text>
                </div>
                <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  {tr('请求 Header 需包含：X-App-Id、X-Timestamp、X-Nonce、X-Signature、X-Sign-Version(v1)。')}
                </Paragraph>
                <Title level={5}>{tr('请求参数')}</Title>
                <DocTable headers={eventReportRequestTable.headers} rows={eventReportRequestTable.rows} />
                </Space>
              </Card>

              <Card title={tr('心跳上报')} style={{ marginTop: 24 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <div>
                    <Tag color="blue">POST</Tag>
                    <Text code>/api/client/heartbeat</Text>
                  </div>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    {tr('请求 Header 需包含：X-App-Id、X-Timestamp、X-Nonce、X-Signature、X-Sign-Version(v1)。')}
                  </Paragraph>
                  <Title level={5}>{tr('请求参数')}</Title>
                  <DocTable headers={heartbeatRequestTable.headers} rows={heartbeatRequestTable.rows} />
                </Space>
              </Card>

              <Card title={tr('用户反馈')} style={{ marginTop: 24 }}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <div>
                  <Tag color="blue">POST</Tag>
                  <Text code>/api/client/feedback</Text>
                  <Text type="secondary" style={{ marginLeft: 8 }}>multipart/form-data</Text>
                </div>
                <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  {tr('请求 Header 需包含：X-App-Id、X-Timestamp、X-Nonce、X-Signature、X-Sign-Version(v1)。')}
                </Paragraph>
                <Title level={5}>{tr('请求参数')}</Title>
                <DocTable headers={feedbackRequestTable.headers} rows={feedbackRequestTable.rows} />
              </Space>
            </Card>
          </div>
        )}

        {activeTab === 'sdk' && (
          <div>
            <Title level={2}>{tr('SDK 参考')}</Title>
            <Paragraph>
              {tr('仅维护 Go / C# 两个官方 SDK 文档，内容直接对齐仓库中的最新源码签名与参数定义。')}
            </Paragraph>

            <Card style={{ marginTop: 24 }}>
              <Title level={4}>{tr('选择语言')}</Title>
              <Space wrap style={{ marginTop: 16 }}>
                {languages.map((lang) => (
                  <Tag
                    key={lang.key}
                    color={selectedLang === lang.key ? lang.color : undefined}
                    style={{ cursor: 'pointer', padding: '4px 12px', fontSize: 14 }}
                    onClick={() => setSelectedLang(lang.key)}
                  >
                    {lang.label}
                  </Tag>
                ))}
              </Space>
            </Card>

            <Card title={`${sdkDoc.languageName} ${tr('初始化')}`} style={{ marginTop: 24 }}>
              <CodeBlock code={sdkDoc.init} copySuccessText={copySuccessText} />
            </Card>

            <Card title={`${sdkDoc.languageName} ${tr('完整调用流程')}`} style={{ marginTop: 24 }}>
              <CodeBlock code={sdkDoc.fullFlow} copySuccessText={copySuccessText} />
            </Card>

            <Card title={tr('Client 运行时字段')} style={{ marginTop: 24 }}>
              <DocTable
                headers={[tr('字段'), tr('类型'), tr('说明')]}
                rows={sdkDoc.runtimeFields.map((item) => [item.name, item.type, item.description])}
              />
            </Card>

            <Card title={tr('CheckUpdate 返回字段')} style={{ marginTop: 24 }}>
              <DocTable
                headers={[tr('字段'), tr('类型'), tr('说明')]}
                rows={sdkDoc.updateResponse.map((item) => [item.name, item.type, item.description])}
              />
            </Card>

            {sdkDoc.methods.map((method) => (
              <Card key={method.name} title={method.name} style={{ marginTop: 24 }}>
                <Paragraph>{method.description}</Paragraph>
                <Paragraph style={{ marginBottom: 8 }}>
                  <Text strong>{tr('方法签名：')}</Text>
                </Paragraph>
                <CodeBlock code={method.signature} copySuccessText={copySuccessText} />
                <Paragraph style={{ marginTop: 12 }}>
                  <Text strong>{tr('底层接口：')}</Text> <Text code>{method.endpoint}</Text>
                </Paragraph>
                <Title level={5}>{tr('参数')}</Title>
                <DocTable
                  headers={[tr('参数'), tr('类型'), tr('必填'), tr('说明')]}
                  rows={method.params.map((item) => [item.name, item.type, item.required, item.description])}
                />
                <Title level={5} style={{ marginTop: 16 }}>{tr('返回')}</Title>
                <DocTable
                  headers={[tr('名称'), tr('类型'), tr('说明')]}
                  rows={method.returns.map((item) => [item.name, item.type, item.description])}
                />
                {method.notes && method.notes.length > 0 && (
                  <>
                    <Title level={5} style={{ marginTop: 16 }}>{tr('注意事项')}</Title>
                    <ul>
                      {method.notes.map((note) => (
                        <li key={note}>{note}</li>
                      ))}
                    </ul>
                  </>
                )}
                <Title level={5} style={{ marginTop: 16 }}>{tr('示例')}</Title>
                <CodeBlock code={method.example} copySuccessText={copySuccessText} />
              </Card>
            ))}

            <Card title={tr('UpdateStreamOptions 参数')} style={{ marginTop: 24 }}>
              <DocTable
                headers={[tr('字段'), tr('类型'), tr('说明')]}
                rows={sdkDoc.streamOptions.map((item) => [item.name, item.type, item.description])}
              />
            </Card>

            <Card title={tr('错误与异常模型')} style={{ marginTop: 24 }}>
              <DocTable
                headers={[tr('名称'), tr('类型'), tr('说明')]}
                rows={sdkDoc.errors.map((item) => [item.name, item.type, item.description])}
              />
            </Card>

            <Card title={tr('管理端 API 方法索引')} style={{ marginTop: 24 }}>
              <Paragraph>{sdkDoc.managementAuthNote}</Paragraph>
              <DocTable
                headers={[tr('SDK 方法'), tr('接口'), tr('说明')]}
                rows={sdkDoc.managementMethods.map((item) => [item.method, item.endpoint, item.description])}
              />
            </Card>
          </div>
        )}

        {activeTab === 'changelog' && (
          <div>
            <Title level={2}>{tr('更新日志')}</Title>

            <Card style={{ marginTop: 24 }}>
              <Title level={4}>v0.1.0 (2024-02-20)</Title>
              <ul>
                <li>{tr('初始版本发布')}</li>
                <li>{tr('支持 7 种编程语言：Go、Java、Python、Node.js、C++、C#、Rust')}</li>
                <li>{tr('提供检查更新功能')}</li>
                <li>{tr('提供事件上报功能')}</li>
                <li>{tr('提供文件下载功能（支持进度回调和 SHA256 校验）')}</li>
              </ul>
            </Card>
          </div>
        )}
      </Content>
    </div>
  )
}
