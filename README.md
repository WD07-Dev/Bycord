# Bycord

<img src="./icon.png" alt="Bycord Logo" width="128">

Bycord is a connection-routing tool for Discord on Windows, Linux, and Android. It is intended for cases where Discord itself works, but a feature or embedded page fails because of the route or region used by the current network.

## What it does

Bycord can find a public proxy automatically or use a proxy supplied by the user. Before using an automatically selected proxy, it checks that the proxy:

- is currently reachable;
- can connect to the Discord gateway;
- exits through a country that is not excluded; and
- has the lowest measured latency among the valid candidates.

Proxy candidates are obtained from ProxyScrape. Bycord reads the current public-IP country from `ifconfig.co/country-iso` and uses it as the default excluded country. If the lookup fails, it falls back to South Korea (`KR`). Proxy exit countries are also checked through `ifconfig.co`, and multiple country codes can be entered with commas.

## How the routing works

### Windows and Linux

Bycord finds the installed Discord client, closes it if it is running, and starts it again with Chromium's `--proxy-server` option. Discord's main web connections then use the selected proxy.

Media and file delivery hosts such as Discord CDN, Discord media, and Google storage remain on the direct connection. This keeps large downloads and media traffic away from the proxy. Applications other than Discord are not affected.

Windows supports standard Discord, Discord PTB, and Discord Canary installations. Linux detects Discord from `PATH`, Flatpak, common AppImage folders, or a path entered manually.

### Android

Android does not provide Chromium launch options for an installed Discord app, so Bycord uses Android's local VPN API instead. The VPN captures Discord API and Gateway routes from installed Discord, Discord PTB, or Discord Canary packages and forwards them through a SOCKS5 tunnel.

## Automatic and manual modes

**Automatic mode** downloads a list of public SOCKS5 proxies, filters it, tests the remaining candidates, and chooses a working route. This can take up to a few minutes because public proxy availability changes constantly.

**Manual mode** skips public proxy discovery and uses the server entered by the user. Desktop builds accept these forms:

```text
socks5://host:port
http://host:port
https://host:port
```

The Android build accepts a SOCKS5 host and port.

## Important notes

- Public proxies are operated by unknown third parties. Do not consider them private or trustworthy.
- A proxy can disappear, become overloaded, change country, or stop reaching Discord at any time.
- Bycord verifies connectivity before connecting, but that check cannot guarantee continued availability.
- Connection speed and voice, stream, or embedded-app quality depend on the selected proxy and its distance.
- Desktop Discord must be restarted to apply a different proxy route. Bycord handles this automatically when connecting.
- Android requires VPN permission and Android 10 or later. Battery restrictions may stop the VPN service in the background on some devices.
- Only one Android VPN service can normally be active at a time, so Bycord may conflict with another VPN or firewall app.
- This project does not guarantee access to every Discord feature or network because Discord and its infrastructure can change.

## Downloads and basic use

- [Download](https://github.com/WD07-Dev/Bycord/releases)

| Platform | File |
| --- | --- |
| Windows | `bycord-window.exe` |
| Linux | `bycord-linux.zip` |
| Android | `bycord-android.apk` |

1. Open Bycord and choose automatic or manual mode.
2. Set the countries that should not be used as proxy exits.
3. Connect and wait for the route check to complete.
4. On Android, approve the VPN request when prompted.

The interface supports English and Korean.

## Credits

- Development: **wheedo07**
- App icon: [**bon-bon**](https://discord.com/users/820599655255375893)

## License

Bycord is available under the [MIT License](LICENSE).

Discord is a trademark of Discord Inc. Bycord is an independent project and is not affiliated with Discord Inc.
