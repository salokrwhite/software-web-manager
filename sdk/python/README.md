# SWM Python SDK

Usage example:

```python
from swm_sdk import Client

client = Client("http://localhost:8080", "YOUR_APP_KEY")
client.channel = "stable"
client.platform = "windows"
client.arch = "x64"
client.device_id = "device-1"
client.attributes = {"country": "CN", "os_version": "10.0"}

resp = client.check_update("1.0.0")
print(resp)

client.report_event("app_started", {"version": "1.0.0"})

client.report_events([
    {"event_name": "check_update", "device_id": "device-1", "properties": {"version": "1.0.0"}},
    {"event_name": "update_failed", "device_id": "device-1", "properties": {"reason": "network"}}
])
```
