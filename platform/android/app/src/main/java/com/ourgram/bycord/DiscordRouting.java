package com.ourgram.bycord;

import android.net.VpnService;
import java.io.IOException;
import java.net.InetAddress;
import java.util.HashSet;
import java.util.Set;

final class DiscordRouting {
    private static final String[] PROXY_HOSTS = {
        "discord.com",
        "gateway.discord.gg"
    };

    private DiscordRouting() {}

    static void apply(VpnService.Builder builder) throws IOException {
        Set<String> routes = new HashSet<>();
        for(String host : PROXY_HOSTS) {
            try {
                for(InetAddress address : InetAddress.getAllByName(host)) {
                    String value = address.getHostAddress();
                    if(routes.add(value)) builder.addRoute(address, address.getAddress().length * 8);
                }
            } catch (IOException ignored) {
            }
        }
        if(routes.isEmpty()) throw new IOException("Discord routes could not be resolved");
    }
}