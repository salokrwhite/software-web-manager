# SwmSdk (Java)

`SwmSdk` is the Java SDK for Software Web Manager.

## Requirements

- Java 17+
- Maven 3.9+

## Features

- Update check / download / heartbeat / events / feedback
- SSE update stream with reconnect support
- Device shutdown control event (`device_shutdown`)
- Device blocked / region blocked error mapping
- Management APIs aligned with `sdk/go`

## Quick Start

```java
import com.swm.sdk.Client;

var client = new Client("http://localhost:8080", "your_app_id", "your_app_secret");
client.setChannel("stable");
client.setPlatform("windows");
client.setArch("amd64");
client.setDeviceId("device-001");

var update = client.checkUpdate("1.0.0", 100);
```

## Management API

```java
client.setAuthToken("jwt_token");
var app = client.getApp("app_id");
var channels = client.listChannels("app_id");
```
