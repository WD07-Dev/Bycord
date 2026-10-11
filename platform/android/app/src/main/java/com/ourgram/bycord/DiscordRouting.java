package com.ourgram.bycord;

import android.net.IpPrefix;
import android.net.VpnService;
import android.os.Build;
import java.io.IOException;
import java.net.InetAddress;
import java.util.ArrayList;
import java.util.List;

final class DiscordRouting {
    private static final String[] DIRECT_CIDRS = {
        "173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
        "141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
        "197.234.240.0/22", "198.41.128.0/17", "104.16.0.0/13", "104.24.0.0/14",
        "172.64.0.0/13", "131.0.72.0/22",
        "2400:cb00::/32", "2606:4700::/32", "2803:f800::/32", "2405:b500::/32",
        "2405:8100::/32", "2a06:98c0::/29", "2c0f:f248::/32"
    };

    private DiscordRouting() {}

    static void apply(VpnService.Builder builder) throws IOException {
        List<Route> ipv4 = new ArrayList<>();
        List<Route> ipv6 = new ArrayList<>();
        for(String cidr : DIRECT_CIDRS) {
            Route route = Route.parse(cidr);
            (route.address.length == 4 ? ipv4 : ipv6).add(route);
        }

        if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            builder.addRoute("0.0.0.0", 0).addRoute("::", 0);
            exclude(builder, ipv4);
            exclude(builder, ipv6);
            return;
        }

        addRoutesExcept(builder, ipv4, new byte[4], 0);
        addRoutesExcept(builder, ipv6, new byte[16], 0);
    }

    private static void exclude(VpnService.Builder builder, List<Route> routes) throws IOException {
        for(Route route : routes) {
            builder.excludeRoute(route.toIpPrefix());
        }
    }

    private static void addRoutesExcept(
        VpnService.Builder builder,
        List<Route> excluded,
        byte[] prefix,
        int bit
    ) throws IOException {
        if(excluded.isEmpty()) {
            builder.addRoute(InetAddress.getByAddress(prefix), bit);
            return;
        }
        for(Route route : excluded) {
            if(route.prefixLength == bit) return;
        }
        if(bit == prefix.length * 8) return;

        int index = bit / 8;
        int mask = 1 << (7 - bit % 8);
        List<Route> zero = new ArrayList<>();
        List<Route> one = new ArrayList<>();
        for (Route route : excluded) {
            if ((route.address[index] & mask) == 0) {
                zero.add(route);
            } else {
                one.add(route);
            }
        }

        addRoutesExcept(builder, zero, prefix.clone(), bit + 1);
        byte[] onePrefix = prefix.clone();
        onePrefix[index] = (byte) (onePrefix[index] | mask);
        addRoutesExcept(builder, one, onePrefix, bit + 1);
    }

    private static final class Route {
        private final byte[] address;
        private final int prefixLength;

        private Route(byte[] address, int prefixLength) {
            this.address = address;
            this.prefixLength = prefixLength;
        }

        private static Route parse(String cidr) throws IOException {
            int slash = cidr.lastIndexOf('/');
            InetAddress address = InetAddress.getByName(cidr.substring(0, slash));
            int prefixLength = Integer.parseInt(cidr.substring(slash + 1));
            return new Route(address.getAddress(), prefixLength);
        }

        private IpPrefix toIpPrefix() throws IOException {
            return new IpPrefix(InetAddress.getByAddress(address), prefixLength);
        }
    }
}