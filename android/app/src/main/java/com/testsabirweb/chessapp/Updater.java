package com.testsabirweb.chessapp;

import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.net.Uri;
import android.util.Log;

import androidx.core.content.FileProvider;

import com.testsabirweb.chessapp.mobile.Mobile;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;

public class Updater {
    private static final String TAG = "Updater";
    private static final String API =
            "https://api.github.com/repos/testsabirweb/chess-app/releases/latest";

    private final Context ctx;
    private File pendingApk;

    public Updater(Context ctx) {
        this.ctx = ctx.getApplicationContext();
        new Thread(this::checkAndDownload).start();
    }

    private void checkAndDownload() {
        try {
            JSONObject release = new JSONObject(httpGet(API));
            long remoteCode = tagToVersionCode(release.getString("tag_name"));
            if (remoteCode <= getLocalVersionCode()) {
                return;
            }

            JSONArray assets = release.getJSONArray("assets");
            String downloadUrl = null;
            for (int i = 0; i < assets.length(); i++) {
                JSONObject asset = assets.getJSONObject(i);
                if (asset.getString("name").endsWith(".apk")) {
                    downloadUrl = asset.getString("browser_download_url");
                    break;
                }
            }
            if (downloadUrl == null) {
                return;
            }

            File dir = ctx.getExternalFilesDir(null);
            if (dir == null) {
                return;
            }
            File apk = new File(dir, "update.apk");
            downloadFile(downloadUrl, apk);
            pendingApk = apk;
            Mobile.setUpdateReady(true);
        } catch (Exception e) {
            Log.w(TAG, "update check failed", e);
        }
    }

    static long tagToVersionCode(String tag) {
        String v = tag.startsWith("v") ? tag.substring(1) : tag;
        String[] parts = v.split("\\.");
        int ma = parts.length > 0 ? parseInt(parts[0]) : 0;
        int mi = parts.length > 1 ? parseInt(parts[1]) : 0;
        int pa = parts.length > 2 ? parseInt(parts[2]) : 0;
        return ma * 10000L + mi * 100L + pa;
    }

    private static int parseInt(String s) {
        try {
            return Integer.parseInt(s);
        } catch (NumberFormatException e) {
            return 0;
        }
    }

    private long getLocalVersionCode() throws PackageManager.NameNotFoundException {
        PackageInfo info = ctx.getPackageManager().getPackageInfo(ctx.getPackageName(), 0);
        return info.getLongVersionCode();
    }

    private String httpGet(String urlStr) throws Exception {
        HttpURLConnection conn = (HttpURLConnection) new URL(urlStr).openConnection();
        conn.setInstanceFollowRedirects(true);
        conn.setRequestProperty("Accept", "application/vnd.github+json");
        conn.connect();
        try (InputStream in = conn.getInputStream()) {
            return readUtf8(in);
        } finally {
            conn.disconnect();
        }
    }

    private void downloadFile(String urlStr, File dest) throws Exception {
        HttpURLConnection conn = (HttpURLConnection) new URL(urlStr).openConnection();
        conn.setInstanceFollowRedirects(true);
        conn.connect();
        try (InputStream in = conn.getInputStream();
             FileOutputStream out = new FileOutputStream(dest)) {
            byte[] buf = new byte[8192];
            int n;
            while ((n = in.read(buf)) != -1) {
                out.write(buf, 0, n);
            }
        } finally {
            conn.disconnect();
        }
    }

    private static String readUtf8(InputStream in) throws Exception {
        ByteArrayOutputStream out = new ByteArrayOutputStream();
        byte[] buf = new byte[8192];
        int n;
        while ((n = in.read(buf)) != -1) {
            out.write(buf, 0, n);
        }
        return out.toString(StandardCharsets.UTF_8.name());
    }

    public void installPending() {
        if (pendingApk == null || !pendingApk.exists()) {
            return;
        }
        try {
            Uri uri = FileProvider.getUriForFile(
                    ctx, ctx.getPackageName() + ".fileprovider", pendingApk);
            Intent intent = new Intent(Intent.ACTION_VIEW)
                    .setDataAndType(uri, "application/vnd.android.package-archive")
                    .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_ACTIVITY_NEW_TASK);
            ctx.startActivity(intent);
        } catch (Exception e) {
            Log.w(TAG, "install failed", e);
        }
    }
}
