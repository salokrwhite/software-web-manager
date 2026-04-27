# SwmSdk (C#)

`SwmSdk` is the C# SDK for Software Web Manager.

## Target Frameworks

- `net8.0`
- `netstandard2.0`

## Core Features

- Update check / SSE update stream
- Device shutdown control event (`device_shutdown`)
- Device blocked contract (`error.code = device_blocked`)
- Heartbeat / events / feedback / download
- Full management APIs aligned with Go SDK

## Quick Start

```csharp
using SwmSdk;

var client = new Client("http://localhost:8080", "your_app_id", "your_app_secret")
{
    Channel = "stable",
    Platform = "windows",
    Arch = "amd64",
    DeviceId = "device-001"
};

var update = await client.CheckUpdateAsync("1.0.0", 100);
```

## Signature Verification

When `VerifySignature=true` and `PublicKey` is configured, the SDK verifies update signatures with Ed25519 by default.
If you need custom verification logic, set `SignatureVerifier` to override the default behavior.

## Analytics Event Convention

`CheckUpdateAsync` / `DownloadAsync` do not auto-report analytics events. You need to call `ReportEventAsync` manually.

Recommended event names used by the built-in analytics page:

- `check_update`
- `update_available`
- `download_started`
- `download_completed`
- `install_completed`
- `app_started`
- `update_failed`

For release-channel metrics, include `release_id` in `properties` (from `CheckUpdateAsync` response). Also ensure `client.Channel` matches an existing channel code (for example `stable`).

## Device Blocked Handling

```csharp
try
{
    await client.ReportHeartbeatAsync("1.0.0");
}
catch (SwmDeviceBlockedException)
{
    Environment.Exit(23);
}
```

## SSE Control Event

```csharp
var handle = client.StartUpdateStream(
    new UpdateStreamOptions
    {
        CurrentVersion = "1.0.0",
        VersionCode = 100,
        OnControlEvent = evt =>
        {
            if (evt.Type == Client.ControlEventShutdown)
            {
                Environment.Exit(23);
            }
        }
    },
    evt => { /* release events */ });
```

## Management APIs

```csharp
client.SetAuthToken("jwt_token");
var app = await client.GetAppAsync("app_id");
var channels = await client.ListChannelsAsync("app_id");
```

## NativeAOT

See [README.NativeAOT.md](README.NativeAOT.md).
