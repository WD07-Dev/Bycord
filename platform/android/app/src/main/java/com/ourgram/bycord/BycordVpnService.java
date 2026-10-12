package com.ourgram.bycord;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.net.VpnService;
import android.os.ParcelFileDescriptor;
import java.io.File;
import java.io.FileWriter;
import java.io.IOException;
import hev.htproxy.TProxyService;

public final class BycordVpnService extends VpnService {
    public static final String ACTION_START = "com.ourgram.bycord.START";
    public static final String ACTION_STOP = "com.ourgram.bycord.STOP";
    public static final String EXTRA_HOST = "host";
    public static final String EXTRA_PORT = "port";

    private static final int NOTIFICATION_ID = 7;
    private static final String NOTIFICATION_CHANNEL = "bycord-vpn";
    private static final String[] DISCORD_PACKAGES = {
        "com.discord",
        "com.discord.ptb",
        "com.discord.canary"
    };

    private static volatile boolean connected;
    private static volatile boolean starting;
    private static volatile String lastError;

    private ParcelFileDescriptor tunnel;

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        if(intent == null) {
            return START_NOT_STICKY;
        }
        if(ACTION_STOP.equals(intent.getAction())) {
            stopTunnel();
            return START_NOT_STICKY;
        }
        if(!ACTION_START.equals(intent.getAction())) {
            return START_NOT_STICKY;
        }

        String host = intent.getStringExtra(EXTRA_HOST);
        int port = intent.getIntExtra(EXTRA_PORT, 0);
        if(host == null || host.isBlank() || port < 1 || port > 65535) {
            stopSelf();
            return START_NOT_STICKY;
        }

        connected = false;
        starting = true;
        lastError = null;
        startForeground(NOTIFICATION_ID, notification(getString(R.string.notification_starting)));
        new Thread(() -> startTunnelInBackground(host, port), "BycordVpnStart").start();
        return START_STICKY;
    }

    private void startTunnelInBackground(String host, int port) {
        try {
            startTunnel(host, port);
        } catch (Exception error) {
            lastError = error.getMessage() == null ? error.getClass().getSimpleName() : error.getMessage();
            stopTunnel();
        } finally {
            starting = false;
        }
    }

    private void startTunnel(String host, int port) throws Exception {
        stopNativeOnly();

        Builder builder = new Builder()
        .setSession("Bycord")
        .setMtu(8500)
        .addAddress("198.18.0.1", 32)
        .addAddress("fc00::1", 128)
        .setBlocking(true);
        DiscordRouting.apply(builder);
        addDiscordApps(builder);

        tunnel = builder.establish();
        if(tunnel == null) {
            throw new IOException(getString(R.string.error_vpn_interface));
        }

        File config = writeConfig(host, port);
        if(!TProxyService.TProxyStartService(config.getAbsolutePath(), tunnel.getFd())) {
            throw new IOException(getString(R.string.error_tun2socks));
        }

        connected = true;
        NotificationManager manager = getSystemService(NotificationManager.class);
        manager.notify(NOTIFICATION_ID, notification(getString(R.string.notification_connected)));
    }

    private void addDiscordApps(Builder builder) {
        int installed = 0;
        for (String packageName : DISCORD_PACKAGES) {
            try {
                builder.addAllowedApplication(packageName);
                installed++;
            } catch (PackageManager.NameNotFoundException ignored) {
            }
        }
        if(installed == 0) {
            throw new IllegalStateException(getString(R.string.error_discord_missing));
        }
    }

    private File writeConfig(String host, int port) throws IOException {
        File config = new File(getFilesDir(), "hev.yml");
        try (FileWriter writer = new FileWriter(config, false)) {
            writer.write("tunnel:\n");
            writer.write("  mtu: 8500\n");
            writer.write("  ipv4: 198.18.0.1\n");
            writer.write("  ipv6: 'fc00::1'\n");
            writer.write("socks5:\n");
            writer.write("  address: '" + yaml(host) + "'\n");
            writer.write("  port: " + port + "\n");
            writer.write("  udp: 'udp'\n");
            writer.write("misc:\n");
            writer.write("  log-level: warn\n");
        }
        return config;
    }

    private String yaml(String value) {
        return value.replace("'", "''");
    }

    private Notification notification(String message) {
        NotificationManager manager = getSystemService(NotificationManager.class);
        NotificationChannel channel = new NotificationChannel(
            NOTIFICATION_CHANNEL,
            getString(R.string.notification_channel),
            NotificationManager.IMPORTANCE_LOW
        );
        manager.createNotificationChannel(channel);

        return new Notification.Builder(this, NOTIFICATION_CHANNEL)
        .setSmallIcon(android.R.drawable.stat_sys_warning)
        .setContentTitle(getString(R.string.app_name))
        .setContentText(message)
        .setOngoing(true)
        .build();
    }

    public static boolean isConnected() {
        return connected;
    }

    public static boolean isStarting() {
        return starting;
    }

    public static String getLastError() {
        return lastError;
    }

    private void stopNativeOnly() {
        connected = false;
        if(TProxyService.TProxyIsRunning()) {
            TProxyService.TProxyStopService();
        }
        if(tunnel == null) return;
        try {
            tunnel.close();
        } catch (IOException ignored) {
        }
        tunnel = null;
    }

    private void stopTunnel() {
        stopNativeOnly();
        stopForeground(STOP_FOREGROUND_REMOVE);
        stopSelf();
    }

    @Override
    public void onRevoke() {
        stopTunnel();
    }

    @Override
    public void onDestroy() {
        stopNativeOnly();
        super.onDestroy();
    }
}