package com.swm.sdk;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.bouncycastle.crypto.params.Ed25519PublicKeyParameters;
import org.bouncycastle.crypto.signers.Ed25519Signer;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.io.BufferedReader;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.URI;
import java.net.URLDecoder;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpHeaders;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.GeneralSecurityException;
import java.security.MessageDigest;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Base64;
import java.util.HexFormat;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ThreadLocalRandom;
import java.util.function.BiConsumer;
import java.util.function.Consumer;

public class Client {
    public static final String CONTROL_EVENT_SHUTDOWN = "device_shutdown";
    public static final String API_ERROR_CODE_DEVICE_BLOCKED = "device_blocked";
    public static final String API_ERROR_CODE_UPDATE_REGION_BLOCKED = "update_region_blocked";

    private static final String SIGN_HEADER_APP_ID = "X-App-Id";
    private static final String SIGN_HEADER_TIMESTAMP = "X-Timestamp";
    private static final String SIGN_HEADER_NONCE = "X-Nonce";
    private static final String SIGN_HEADER_SIGNATURE = "X-Signature";
    private static final String SIGN_HEADER_VERSION = "X-Sign-Version";
    private static final String SIGN_VERSION_V1 = "v1";
    private static final TypeReference<Map<String, Object>> MAP_TYPE = new TypeReference<>() {};

    private static final ObjectMapper MAPPER = new ObjectMapper()
            .registerModule(new JavaTimeModule())
            .setSerializationInclusion(JsonInclude.Include.NON_NULL)
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);

    private final String baseUrl;
    private final String appId;
    private final String appSecret;
    private final HttpClient httpClient;

    private String authToken = "";
    private String channel = "";
    private String platform = "";
    private String arch = "";
    private String deviceId = "";
    private String userId = "";
    private Map<String, Object> attributes = new LinkedHashMap<>();
    private String publicKey = "";
    private boolean verifySignature;
    private int retries = 2;
    private Duration backoff = Duration.ofMillis(500);

    public Client(String baseUrl, String appId, String appSecret) {
        this(baseUrl, appId, appSecret, HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(30))
                .followRedirects(HttpClient.Redirect.NEVER)
                .build());
    }

    public Client(String baseUrl, String appId, String appSecret, HttpClient httpClient) {
        this.baseUrl = trimTrailingSlash(baseUrl);
        this.appId = appId;
        this.appSecret = appSecret;
        this.httpClient = httpClient;
    }

    public String getBaseUrl() { return baseUrl; }
    public String getAppId() { return appId; }
    public String getAppSecret() { return appSecret; }
    public String getAuthToken() { return authToken; }
    public void setAuthToken(String authToken) { this.authToken = trim(authToken); }
    public String getChannel() { return channel; }
    public void setChannel(String channel) { this.channel = trim(channel); }
    public String getPlatform() { return platform; }
    public void setPlatform(String platform) { this.platform = trim(platform); }
    public String getArch() { return arch; }
    public void setArch(String arch) { this.arch = trim(arch); }
    public String getDeviceId() { return deviceId; }
    public void setDeviceId(String deviceId) { this.deviceId = trim(deviceId); }
    public String getUserId() { return userId; }
    public void setUserId(String userId) { this.userId = trim(userId); }
    public Map<String, Object> getAttributes() { return attributes; }
    public void setAttributes(Map<String, Object> attributes) { this.attributes = attributes == null ? new LinkedHashMap<>() : new LinkedHashMap<>(attributes); }
    public String getPublicKey() { return publicKey; }
    public void setPublicKey(String publicKey) { this.publicKey = trim(publicKey); }
    public boolean isVerifySignature() { return verifySignature; }
    public void setVerifySignature(boolean verifySignature) { this.verifySignature = verifySignature; }
    public int getRetries() { return retries; }
    public void setRetries(int retries) { this.retries = retries; }
    public Duration getBackoff() { return backoff; }
    public void setBackoff(Duration backoff) { this.backoff = backoff; }

    public UpdateCheckResponse checkUpdate(String currentVersion, Integer versionCode) {
        UpdateCheckRequest payload = new UpdateCheckRequest()
                .setChannelCode(channel)
                .setCurrentVersion(currentVersion)
                .setVersionCode(versionCode)
                .setPlatform(platform)
                .setArch(arch)
                .setDeviceId(deviceId)
                .setUserId(userId)
                .setAttributes(attributes);
        UpdateCheckResponse response = parseJson(doRequest("POST", "/api/client/update-check", jsonBytes(payload), "application/json"), UpdateCheckResponse.class);
        verifyUpdateSignature(response);
        return response;
    }

    public void reportEvent(String eventName, Map<String, Object> props) {
        Event payload = new Event()
                .setDeviceId(deviceId)
                .setEventName(eventName)
                .setEventTime(Instant.now())
                .setChannelCode(channel)
                .setProperties(props == null ? new LinkedHashMap<>() : props)
                .setAttributes(attributes);
        doRequest("POST", "/api/client/events", jsonBytes(payload), "application/json");
    }

    public void reportHeartbeat(String appVersion) {
        if (isBlank(deviceId)) {
            throw new SwmValidationException(400, null, "device_id required");
        }
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("device_id", deviceId);
        putIfNotBlank(payload, "channel_code", channel);
        putIfNotBlank(payload, "app_version", appVersion);
        putIfNotBlank(payload, "platform", platform);
        putIfNotBlank(payload, "arch", arch);
        putIfNotBlank(payload, "user_id", userId);
        if (!attributes.isEmpty()) {
            payload.put("attributes", attributes);
        }
        doRequest("POST", "/api/client/heartbeat", jsonBytes(payload), "application/json");
    }

    public void reportEvents(List<Event> events) {
        doRequest("POST", "/api/client/events", jsonBytes(Map.of("events", events)), "application/json");
    }

    public void reportFeedback(String content, Integer rating, String contact, List<String> attachments, Map<String, Object> metadata) {
        if (isBlank(content)) {
            throw new SwmValidationException(400, null, "content required");
        }
        String boundary = "----swm-" + UUID.randomUUID();
        try {
            byte[] body = buildFeedbackMultipart(boundary, content, rating, contact, attachments, metadata);
            doRequest("POST", "/api/client/feedback", body, "multipart/form-data; boundary=" + boundary);
        } catch (IOException e) {
            throw new SwmApiException(0, null, e.getMessage());
        }
    }

    public void download(String url, String destPath, String checksum, String signature, BiConsumer<Long, Long> progress) {
        HttpResponse<InputStream> response = send(HttpRequest.newBuilder(URI.create(url)).GET().build(), HttpResponse.BodyHandlers.ofInputStream());
        if (response.statusCode() >= 300) {
            throw new SwmApiException(response.statusCode(), null, "download failed: " + response.statusCode());
        }
        try {
            Path path = Path.of(destPath);
            if (path.getParent() != null) {
                Files.createDirectories(path.getParent());
            }
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            long total = response.headers().firstValueAsLong("Content-Length").orElse(-1L);
            long written = 0;
            try (InputStream in = response.body(); OutputStream out = Files.newOutputStream(path)) {
                byte[] buffer = new byte[32 * 1024];
                int read;
                while ((read = in.read(buffer)) >= 0) {
                    if (read == 0) {
                        continue;
                    }
                    out.write(buffer, 0, read);
                    digest.update(buffer, 0, read);
                    written += read;
                    if (progress != null) {
                        progress.accept(written, total);
                    }
                }
            }
            if (!isBlank(checksum)) {
                String got = HexFormat.of().formatHex(digest.digest());
                if (!got.equalsIgnoreCase(checksum)) {
                    throw new SwmApiException(0, null, "checksum mismatch: " + got + " != " + checksum);
                }
            }
            if (verifySignature && !isBlank(signature) && !isBlank(checksum)) {
                verifySignature(checksum, signature);
            }
        } catch (IOException | GeneralSecurityException e) {
            throw new SwmApiException(0, null, e.getMessage());
        }
    }

    public UpdateWatchHandle startUpdateStream(UpdateStreamOptions options, Consumer<UpdatePushEvent> onEvent) {
        String resolvedChannel = firstNonBlank(options.getChannelCode(), channel);
        String resolvedPlatform = firstNonBlank(options.getPlatform(), platform);
        String resolvedArch = firstNonBlank(options.getArch(), arch);
        String resolvedDeviceId = firstNonBlank(options.getDeviceId(), deviceId);
        if (isBlank(resolvedChannel) || isBlank(resolvedPlatform) || isBlank(resolvedArch) || isBlank(resolvedDeviceId)) {
            throw new SwmValidationException(400, null, "channel_code/platform/arch/device_id required");
        }
        Thread worker = new Thread(() -> consumeUpdateStream(options, resolvedChannel, resolvedPlatform, resolvedArch, resolvedDeviceId, onEvent), "swm-update-stream");
        worker.setDaemon(true);
        worker.start();
        return new UpdateWatchHandle(worker);
    }

    public UpdateWatchHandle watchUpdates(UpdateStreamOptions options, Consumer<UpdateCheckResponse> onUpdateAvailable) {
        return startUpdateStream(options, evt -> {
            if (CONTROL_EVENT_SHUTDOWN.equalsIgnoreCase(trim(evt.getEventType()))) {
                return;
            }
            try {
                UpdateCheckResponse response = checkUpdate(options.getCurrentVersion(), options.getVersionCode());
                if (response.isUpdateAvailable()) {
                    onUpdateAvailable.accept(response);
                }
            } catch (Throwable ex) {
                if (options.getOnError() != null) {
                    options.getOnError().accept(ex);
                }
            }
        });
    }

    public Map<String, Object> getApp(String appId) { return unwrapObject(authJson("GET", "/api/apps/" + appId, null, null), "app"); }
    public Map<String, Object> updateApp(String appId, Map<String, Object> payload) { return unwrapObject(authJson("PATCH", "/api/apps/" + appId, payload, null), "app"); }
    public List<Map<String, Object>> listChannels(String appId) { return unwrapItems(authJson("GET", "/api/apps/" + appId + "/channels", null, null)); }
    public Map<String, Object> createChannel(String appId, Map<String, Object> payload) { return unwrapObject(authJson("POST", "/api/apps/" + appId + "/channels", payload, null), "channel"); }
    public List<Map<String, Object>> listAppMembers(String appId) { return unwrapItems(authJson("GET", "/api/apps/" + appId + "/members", null, null)); }
    public Map<String, Object> addAppMember(String appId, Map<String, Object> payload) { return parseJson(authJson("POST", "/api/apps/" + appId + "/members", payload, null), MAP_TYPE); }
    public List<Map<String, Object>> listReleases(String appId) { return unwrapItems(authJson("GET", "/api/apps/" + appId + "/releases", null, null)); }
    public Map<String, Object> createRelease(String appId, Map<String, Object> payload) { return unwrapObject(authJson("POST", "/api/apps/" + appId + "/releases", payload, null), "release"); }
    public Map<String, Object> updateRelease(String releaseId, Map<String, Object> payload) { return unwrapObject(authJson("PATCH", "/api/releases/" + releaseId, payload, null), "release"); }
    public void deleteRelease(String releaseId) { authJson("DELETE", "/api/releases/" + releaseId, null, null); }
    public void submitRelease(String releaseId, String note) { reviewReleaseAction(releaseId, "submit", note); }
    public void approveRelease(String releaseId, String note) { reviewReleaseAction(releaseId, "approve", note); }
    public void rejectRelease(String releaseId, String note) { reviewReleaseAction(releaseId, "reject", note); }
    public Map<String, Object> publishRelease(String releaseId, Map<String, Object> payload) { return unwrapObject(authJson("POST", "/api/releases/" + releaseId + "/publish", payload, null), "release_channel"); }
    public Map<String, Object> rollbackRelease(String releaseId, Map<String, Object> payload) { return unwrapObject(authJson("POST", "/api/releases/" + releaseId + "/rollback", payload, null), "release_channel"); }
    public void revokeRelease(String releaseId) { authJson("POST", "/api/releases/" + releaseId + "/revoke", Map.of(), null); }
    public List<Map<String, Object>> listArtifacts(String releaseId) { return unwrapItems(authJson("GET", "/api/releases/" + releaseId + "/artifacts", null, null)); }
    public List<Map<String, Object>> listReleaseChannels(String appId) { return unwrapItems(authJson("GET", "/api/apps/" + appId + "/release-channels", null, null)); }
    public Map<String, Object> createReleaseChannel(String appId, Map<String, Object> payload) { return unwrapObject(authJson("POST", "/api/apps/" + appId + "/release-channels", payload, null), "release_channel"); }
    public Map<String, Object> updateReleaseChannel(String releaseChannelId, Map<String, Object> payload) { return unwrapObject(authJson("PATCH", "/api/release-channels/" + releaseChannelId, payload, null), "release_channel"); }
    public List<Map<String, Object>> listReleaseTemplates() { return unwrapItems(authJson("GET", "/api/release-templates", null, null)); }
    public Map<String, Object> createReleaseTemplate(Map<String, Object> payload) { return unwrapObject(authJson("POST", "/api/release-templates", payload, null), "template"); }
    public Map<String, Object> updateReleaseTemplate(String templateId, Map<String, Object> payload) { return unwrapObject(authJson("PATCH", "/api/release-templates/" + templateId, payload, null), "template"); }
    public void deleteReleaseTemplate(String templateId) { authJson("DELETE", "/api/release-templates/" + templateId, null, null); }
    public Map<String, Object> setReleaseTemplate(String releaseId, String templateId) { return unwrapObject(authJson("PATCH", "/api/releases/" + releaseId + "/template", Map.of("template_id", templateId), null), "release"); }
    public List<Map<String, Object>> listAppSecrets(String appId) { return unwrapItems(authJson("GET", "/api/apps/" + appId + "/app-secrets", null, null)); }
    public AppSecretCreateResponse createAppSecret(String appId, Map<String, Object> payload) { return parseJson(authJson("POST", "/api/apps/" + appId + "/app-secrets", payload, null), AppSecretCreateResponse.class); }
    public void revokeAppSecret(String keyId) { authJson("DELETE", "/api/app-secrets/" + keyId, null, null); }
    public Map<String, Object> listGeoRegions() { return parseJson(authJson("GET", "/api/geo/regions", null, null), MAP_TYPE); }
    public Map<String, Object> resolveGeo(String ip) { return parseJson(authJson("GET", "/api/geo/resolve", null, Map.of("ip", ip)), MAP_TYPE); }

    public Map<String, Object> uploadArtifact(String releaseId, UploadArtifactOptions options) {
        if (isBlank(options.getPlatform()) || isBlank(options.getArch()) || isBlank(options.getFileType())) {
            throw new SwmValidationException(400, null, "platform, arch, file_type required");
        }
        String boundary = "----swm-" + UUID.randomUUID();
        try {
            byte[] body = buildArtifactMultipart(boundary, options);
            return unwrapObject(doAuthRequest("POST", "/api/releases/" + releaseId + "/artifacts", null, body, "multipart/form-data; boundary=" + boundary), "artifact");
        } catch (IOException e) {
            throw new SwmApiException(0, null, e.getMessage());
        }
    }

    public String getArtifactDownloadURL(String artifactId) {
        ensureAuthToken();
        String path = "/api/artifacts/" + artifactId + "/download";
        URI uri = URI.create(baseUrl + path);
        HttpRequest.Builder builder = HttpRequest.newBuilder(uri)
                .GET()
                .header("Authorization", "Bearer " + authToken)
                .timeout(Duration.ofSeconds(30));
        signAuthRequestHeaders(builder, "GET", uri, new byte[0]);
        HttpResponse<Void> response = send(builder.build(), HttpResponse.BodyHandlers.discarding());
        if (response.statusCode() >= 300 && response.statusCode() < 400) {
            return response.headers().firstValue("Location")
                    .orElseThrow(() -> new SwmApiException(response.statusCode(), null, "missing redirect location"));
        }
        throw mapError(response.statusCode(), response.headers(), "");
    }

    public Map<String, Object> getReleaseChannelMetrics(String appId, String releaseChannelId, String from, String to) {
        Map<String, String> query = new LinkedHashMap<>();
        putIfNotBlank(query, "from", from);
        putIfNotBlank(query, "to", to);
        return parseJson(authJson("GET", "/api/apps/" + appId + "/release-channels/" + releaseChannelId + "/metrics", null, query), MAP_TYPE);
    }

    public Object getAppRegionRules(String appId) {
        return unwrapValue(authJson("GET", "/api/apps/" + appId + "/region-rules", null, null), "region_rules");
    }

    public Object updateAppRegionRules(String appId, Object rules) {
        return unwrapValue(authJson("PATCH", "/api/apps/" + appId + "/region-rules", Map.of("region_rules", rules), null), "region_rules");
    }

    private void consumeUpdateStream(UpdateStreamOptions options, String channelCode, String platform, String arch, String deviceId, Consumer<UpdatePushEvent> onEvent) {
        Duration currentBackoff = positiveDuration(options.getReconnectBackoff(), Duration.ofMillis(1500));
        Duration maxBackoff = positiveDuration(options.getReconnectMaxBackoff(), Duration.ofSeconds(20));
        if (maxBackoff.compareTo(currentBackoff) < 0) {
            maxBackoff = currentBackoff;
        }
        while (!Thread.currentThread().isInterrupted()) {
            try {
                connectAndReadSse(options, channelCode, platform, arch, deviceId, onEvent);
                return;
            } catch (Throwable ex) {
                if (options.getOnError() != null) {
                    options.getOnError().accept(ex);
                }
                if (ex instanceof SwmDeviceBlockedException || ex instanceof SwmUpdateRegionBlockedException || ex instanceof SwmUnauthorizedException || !options.isReconnect()) {
                    return;
                }
                sleepWithBackoff(currentBackoff, options.isJitter());
                currentBackoff = currentBackoff.multipliedBy(2);
                if (currentBackoff.compareTo(maxBackoff) > 0) {
                    currentBackoff = maxBackoff;
                }
            }
        }
    }
