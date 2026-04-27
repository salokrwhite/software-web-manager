package com.swm.sdk;

public final class ControlEvent {
    private String type;
    private String deviceId;
    private String reason;

    public String getType() {
        return type;
    }

    public ControlEvent setType(String type) {
        this.type = type;
        return this;
    }

    public String getDeviceId() {
        return deviceId;
    }

    public ControlEvent setDeviceId(String deviceId) {
        this.deviceId = deviceId;
        return this;
    }

    public String getReason() {
        return reason;
    }

    public ControlEvent setReason(String reason) {
        this.reason = reason;
        return this;
    }
}
